package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

const (
	POLL_FREQUENCY_MS = 10 // how often will we poll Kafka queue
	BUFFER_SIZE       = 50
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())

	back_buffer := make([]string, BUFFER_SIZE*2)
	back_buffer_cursor := 0
	db_buffer := make([]string, BUFFER_SIZE*2)

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

	wg := &sync.WaitGroup{}

	// Consumer listening...
	run := true
	for run {

		select {
		case <-ctx.Done():
			// shutting down, so we must clear out the
			// current buffer to avoid data loss
			wg.Add(1)
			go func(buf []string, wg *sync.WaitGroup) {
				log.Println("Clearing the back buffer")
				defer wg.Done()
				for i, message := range buf {
					fmt.Printf("writing %d to db :: %+v\n", i, message)
				}

			}(back_buffer[:back_buffer_cursor], wg)

			run = false

		default:
			ev := c.Poll(POLL_FREQUENCY_MS)

			log.Println("polling")

			if ev == nil {
				continue
			}

			switch e := ev.(type) {
			case *kafka.Message:

				if back_buffer_cursor == BUFFER_SIZE {

					log.Println("Clearing back buffer...")

					for i := 0; i < back_buffer_cursor; i++ {
						db_buffer[i] = back_buffer[i]
					}

					// make sure we wait for our buffer to be written
					// to the db before we exit the program
					wg.Add(1)
					go func(buf []string, wg *sync.WaitGroup) {
						defer wg.Done()
						for i, message := range buf {
							fmt.Printf("writing %d to db :: %+v\n", i, message)
						}

					}(db_buffer[:], wg)

					log.Println("Back buffer cleared")
					back_buffer_cursor = 0
				}

				back_buffer[back_buffer_cursor] = string(e.Value)
				back_buffer_cursor += 1

				// Process the message received.
				// log.Printf("%% Message on Topic :: %s:\n MESSAGE :: %s\n",
				// e.TopicPartition, string(e.Value))
				// if e.Headers != nil {
				// log.Printf("%% Headers: %v\n", e.Headers)
				// }

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

	log.Println("Shutting down consumer...")
	wg.Wait()
	log.Println("Done")
}
