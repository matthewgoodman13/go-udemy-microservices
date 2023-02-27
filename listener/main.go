package main

import (
	"errors"
	"fmt"
	"listener-service/event"
	"log"
	"math"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	// Try to connect to rabbitmq
	rabbitConn, err := connect()
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	defer rabbitConn.Close()

	// Start listening for messages
	log.Println("Listening for and consuming RabbitMQ messages...")

	// Create consumer
	consumer, err := event.NewConsumer(rabbitConn)
	if err != nil {
		panic(err)
	}

	// Watch queue and consume event messages
	err = consumer.Listen([]string{"log.INFO", "log.WARNING", "log.ERROR"})
	if err != nil {
		log.Println(err)
	}
}

func connect() (*amqp.Connection, error) {
	var counts int64
	var backOff = 1 * time.Second
	var connection *amqp.Connection

	// Don't connect to rabbitmq until it's ready
	for {
		conn, err := amqp.Dial("amqp://guest:guest@rabbitmq")
		if err != nil {
			counts++
			fmt.Printf("RabbitMQ is not ready yet...")
		} else {
			log.Println("Connected to RabbitMQ!")
			connection = conn
			break
		}

		if counts > 5 {
			fmt.Println(err)
			return nil, errors.New("RabbitMQ is not ready")
		}

		backOff = time.Duration(math.Pow(float64(counts), 2)) * time.Second
		log.Println("Backing off... Retrying in ", backOff)
		time.Sleep(backOff)

	}

	return connection, nil
}
