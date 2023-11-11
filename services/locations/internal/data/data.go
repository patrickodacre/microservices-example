package data

import (
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type DataFeed struct{}

func (f *DataFeed) Run() {
	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

		select {
		case sig := <-sigchan:
			fmt.Printf("Caught signal %v: terminating\n", sig)
			os.Exit(1)
		}
	}()

	// TODO: Figure out how to catch an error resulting
	// from a failure to connect. Presently, this program
	// will just keep on running, producing messages,
	// but Kafka won't pick them up.
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": ":9092",
		"client.id":         "some_client_id",
		"acks":              "all",
	})

	if err != nil {
		fmt.Printf("Failed to create Producer: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("Created Producer %+v", p)

	// Listen to all the events on the default events channel
	go func() {
		for e := range p.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				// The message delivery report, indicating success or
				// permanent failure after retries have been exhausted.
				// Application level retries won't help since the client
				// is already configured to do that.
				m := ev
				if m.TopicPartition.Error != nil {
					fmt.Printf("Delivery failed: %v\n", m.TopicPartition.Error)
				} else {
					fmt.Printf("Delivered message to topic %s [%d] at offset %v\n",
						*m.TopicPartition.Topic, m.TopicPartition.Partition, m.TopicPartition.Offset)
				}
			case kafka.Error:
				// Generic client instance-level errors, such as
				// broker connection failures, authentication issues, etc.
				//
				// These errors should generally be considered informational
				// as the underlying client will automatically try to
				// recover from any errors encountered, the application
				// does not need to take action on them.
				fmt.Printf("Error: %v\n", ev)
			default:
				fmt.Printf("Ignored event: %s\n", ev)
			}
		}
	}()

	// create consumer
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": ":9092",
		// Avoid connecting to IPv6 brokers:
		// This is needed for the ErrAllBrokersDown show-case below
		// when using localhost brokers on OSX, since the OSX resolver
		// will return the IPv6 addresses first.
		// You typically don't need to specify this configuration property.
		"broker.address.family": "v4",
		"group.id":              "some_group_id",
		"session.timeout.ms":    6000,
		// Start reading from the first message of each assigned
		// partition if there are no previously committed offsets
		// for this group.
		"auto.offset.reset": "earliest",
		// Whether or not we store offsets automatically.
		"enable.auto.offset.store": false,
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create consumer: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created a Consumer %v\n", c)

	topic := "HVSE"
	topics := []string{topic}
	err = c.SubscribeTopics(topics[:], nil)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to subscribe to topic %+v\n", err)
		os.Exit(1)
	}

	// Consumer listening...
	go func() {

		run := true
		for run {
			ev := c.Poll(3000)
			if ev == nil {
				fmt.Printf("No events in queue.\n")
				continue
			}

			switch e := ev.(type) {
			case *kafka.Message:
				// Process the message received.
				fmt.Printf("%% Message on Topic :: %s:\n MESSAGE :: %s\n",
					e.TopicPartition, string(e.Value))
				if e.Headers != nil {
					fmt.Printf("%% Headers: %v\n", e.Headers)
				}

				// We can store the offsets of the messages manually or let
				// the library do it automatically based on the setting
				// enable.auto.offset.store. Once an offset is stored, the
				// library takes care of periodically committing it to the broker
				// if enable.auto.commit isn't set to false (the default is true).
				// By storing the offsets manually after completely processing
				// each message, we can ensure atleast once processing.
				_, err := c.StoreMessage(e)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%% Error storing offset after message %s:\n",
						e.TopicPartition)
				}
			case kafka.Error:
				// Errors should generally be considered
				// informational, the client will try to
				// automatically recover.
				// But in this example we choose to terminate
				// the application if all brokers are down.
				fmt.Fprintf(os.Stderr, "%% Error: %v: %v\n", e.Code(), e)
				if e.Code() == kafka.ErrAllBrokersDown {
					run = false
				}
			default:
				fmt.Printf("Ignored %v\n", e)
			}
		}

		fmt.Printf("Closing consumer...\n")
		c.Close()
	}()

	// produce messages
	go func() {

		totalMessageCount := 3
		messageCount := 0

		for messageCount < totalMessageCount {
			value := fmt.Sprintf("Producer example, message #%d", messageCount)

			err = p.Produce(&kafka.Message{
				TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
				Value:          []byte(value),
				Headers:        []kafka.Header{{Key: "myTestHeader", Value: []byte("header values are binary")}},
			}, nil)

			fmt.Printf("Produced a message %+v\n", value)

			if err != nil {
				if err.(kafka.Error).Code() == kafka.ErrQueueFull {
					// Producer queue is full, wait 1s for messages
					// to be delivered then try again.
					time.Sleep(time.Second)
					continue
				}
				fmt.Printf("Failed to produce message: %v\n", err)
			}

			fmt.Printf("Successfully sent\n")
			messageCount++
		}

		// Flush and close the producer and the events channel
		messagesRemaining := p.Flush(3000)
		for messagesRemaining > 0 {
			fmt.Printf("Messages remaining :: %+v\n", messagesRemaining)
			messagesRemaining = p.Flush(3000)
			fmt.Print("Still waiting to flush outstanding messages\n")
		}

		fmt.Printf("Closing Producer...\n")
		p.Close()

	}()

}

func NewDataFeed() DataFeed {
	return DataFeed{}
}
