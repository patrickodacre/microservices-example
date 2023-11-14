package main

import (
	"context"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())

	stopchan := make(chan struct{})
	defer close(stopchan)

	sigchan := make(chan os.Signal, 1)
	defer close(sigchan)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	// shutdown signals
	{
		go func() {
			<-stopchan
			cancel()
		}()

		go func() {
			sig := <-sigchan
			log.Printf("Caught signal %v: terminating...\n", sig)
			cancel()
		}()
	}

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

	defer func() {
		log.Printf("Closing consumer...\n")
		c.Close()
	}()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create consumer: %s\n", err)
		os.Exit(1)
	}

	log.Printf("Created a Consumer %v\n", c)

	topic := "iot_data"
	topics := []string{topic}
	err = c.SubscribeTopics(topics[:], nil)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to subscribe to topic %+v\n", err)
		os.Exit(1)
	}

	log.Println("Subscribed to Topic :: ", topic)

	// Consumer listening...
	run := true
	for run {

		select {
		case <-ctx.Done():
			run = false

		default:
			ev := c.Poll(3000)
			log.Println("polling")
			if ev == nil {
				log.Printf("No events in queue.\n")
				continue
			}

			switch e := ev.(type) {
			case *kafka.Message:
				// Process the message received.
				log.Printf("%% Message on Topic :: %s:\n MESSAGE :: %s\n",
					e.TopicPartition, string(e.Value))
				if e.Headers != nil {
					log.Printf("%% Headers: %v\n", e.Headers)
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
				log.Printf("Ignored %v\n", e)
			}

		}
	}

}
