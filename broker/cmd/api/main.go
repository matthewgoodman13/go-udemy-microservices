package main

import (
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const webPort = "80"

type Config struct {
	Rabbit *amqp.Connection
}

func main() {
	// Try to connect to rabbitmq
	rabbitConn, err := connect()
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	defer rabbitConn.Close()

	app := Config{
		Rabbit: rabbitConn,
	}

	log.Printf("Starting broker service on port %s\n", webPort)

	// Define HTTP Server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", webPort),
		Handler: app.routes(),
	}

	// Start HTTP Server
	err = srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
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
