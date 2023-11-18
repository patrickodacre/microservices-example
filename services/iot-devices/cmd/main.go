package main

// use context in everything
// ctx to goroutines -> routines check ctx.Done() to exit
// close all channels

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/gorilla/mux"
)

var dataCount = 0
var numOfDevices = 3

const (
	DELAY_MS = 100
)

type Reading struct {
	UserId                int64
	AccountId             int64
	LocationId            int64
	Ph                    int32
	TemperatureCelsius    int32
	TemperatureFahrenheit int32
	Shp                   int32
	Iron                  int32
}

func main() {

	router := mux.NewRouter()
	ctx, cancel := context.WithCancel(context.Background())

	router.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		// TODO: list all devices and their reading count
		fmt.Fprintf(w, "%d data points created.\n", dataCount)
	})

	server := &http.Server{
		Addr:    ":3000",
		Handler: router,
	}
	serverShutdownChan := make(chan struct{})

	go func() {
		if err := server.ListenAndServe(); err != nil {
			if err.Error() != "http: Server closed" {
				log.Printf("HTTP Server Closed With %+v\n", err)
			}

			log.Println("HTTP Server Shutdown")
			serverShutdownChan <- struct{}{}
		}
	}()
	log.Println("Server is running")

	sigchan := make(chan os.Signal, 1)
	defer close(sigchan)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	stopchan := make(chan struct{}, 1)
	defer close(stopchan)
	// shutdown signals...
	{
		go func() {
			<-stopchan
			fmt.Println("Stop Channel Received Signal")
			server.Shutdown(context.Background())
			cancel()
		}()

		go func() {
			sig := <-sigchan
			log.Printf("Caught signal %v: terminating...\n", sig)
			server.Shutdown(context.Background())
			cancel()
		}()
	}

	// start producing data::
	// use the waitgroup to ensure all our producers shutdown
	// before exiting our program
	wg := &sync.WaitGroup{}

	for i := 0; i < numOfDevices; i++ {
		select {
		case <-ctx.Done():
			// do not create any more devices
			// if we continue to create devices, our goroutines won't get to wg.Done()
			break
		default:
			wg.Add(1)
			feed := NewDataFeed()
			topic := "iot_data"
			sleep := time.Duration(DELAY_MS)
			time.Sleep(sleep * time.Millisecond)
			var device_id int64 = int64(i + 1)

			go feed.Run(ctx, device_id, topic, wg, stopchan)
			log.Println("Spawned IoT device")
		}
	}

	log.Println("Program running...")

	// blocks until the server is shut down
	<-serverShutdownChan

	log.Println("Server is shut down.")
	log.Println("IoT Devices are shutting down...")

	wg.Wait()
	log.Println("Done")
}

type DataFeed struct{}

func NewDataFeed() DataFeed {
	return DataFeed{}
}

func (f *DataFeed) Run(ctx context.Context, device_id int64, topic string, wg *sync.WaitGroup, stopchan chan<- struct{}) {

	s1 := rand.NewSource(time.Now().UnixNano())
	randomizer := rand.New(s1)

	// use RETURN to finish this run() and then close this routine
	// RETURN when I receive a cancel request
	defer wg.Done()

	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": ":9092",
		"client.id":         "some_client_id",
		"acks":              "all",
	})

	if err != nil {
		log.Printf("Failed to create Producer: %s\n", err)
		stopchan <- struct{}{}
		return
	}

	defer func() {
		log.Printf("Closing Producer...\n")
		p.Close()
	}()

	// Listen to all the events on the default events channel
	go func(ctx context.Context) {
		defer func() {
			log.Println("Done p events")
		}()

		for e := range p.Events() {

			select {
			case <-ctx.Done():
				log.Println("Not listing to P Events any longer")
				return
			}

			switch ev := e.(type) {
			case *kafka.Message:
				// The message delivery report, indicating success or
				// permanent failure after retries have been exhausted.
				// Application level retries won't help since the client
				// is already configured to do that.
				m := ev
				if m.TopicPartition.Error != nil {
					log.Printf("Delivery failed: %v\n", m.TopicPartition.Error)
				} else {
					log.Printf("Delivered message to topic %s [%d] at offset %v\n",
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
				log.Printf("Error: %v\n", ev)
			default:
				log.Printf("Ignored event: %s\n", ev)
			}
		}
	}(ctx)

	messageCount := 0
	for {
		select {
		case <-ctx.Done():
			log.Println("Exiting RUN Goroutine")
			return
		default:
			// TODO:: remove dataCount - this was just a cheap way to
			// check if the goroutines were still running
			dataCount += 1

			ph := randomizer.Int31n(10)
			temperature_celsius := randomizer.Int31n(22)
			temperature_fahrenheit := randomizer.Int31n(90)
			shp := randomizer.Int31n(100)
			iron := randomizer.Int31n(100)

			reading := Reading{
				UserId:                device_id,
				AccountId:             device_id,
				LocationId:            device_id,
				Ph:                    ph,
				TemperatureCelsius:    temperature_celsius,
				TemperatureFahrenheit: temperature_fahrenheit,
				Shp:                   shp,
				Iron:                  iron,
			}

			reading_json, err := json.Marshal(reading)

			if err != nil {
				log.Printf("Error JSONing the reading :: %+v\n", err)
				stopchan <- struct{}{}
				return
			}

			value := string(reading_json)

			err = p.Produce(&kafka.Message{
				TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
				Value:          []byte(value),
				Headers:        []kafka.Header{{Key: "myTestHeader", Value: []byte("header values are binary")}},
			}, nil)

			if err != nil {
				if err.(kafka.Error).Code() == kafka.ErrQueueFull {
					// Producer queue is full, wait 1s for messages
					// to be delivered then try again.
					time.Sleep(time.Second)
					continue
				}

				log.Printf("Produce Message Failed: %v\n", err)
				stopchan <- struct{}{}
				return
			}

			log.Printf("Message Sent: %+v\n", value)
			messageCount += 1
			// KPI samples are taken every 15 seconds
			time.Sleep(DELAY_MS * time.Millisecond)

		}
	}
}
