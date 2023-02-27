package event

import (
	"log"

	ampq "github.com/rabbitmq/amqp091-go"
)

type Producer struct {
	connection *ampq.Connection
}

func (p *Producer) setup() error {
	channel, err := p.connection.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	return declareExchange(channel)
}

func (p *Producer) Publish(event string, severity string) error {
	channel, err := p.connection.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	log.Println("Publishing event: ", event, " with severity: ", severity, " to channel")

	err = channel.Publish(
		"logs_topic", // exchange
		severity,     // routing key
		false,        // mandatory
		false,        // immediate
		ampq.Publishing{
			ContentType: "text/plain",
			Body:        []byte(event),
		})
	if err != nil {
		return err
	}

	return nil
}

func NewEventProducer(conn *ampq.Connection) (Producer, error) {
	producer := Producer{
		connection: conn,
	}

	err := producer.setup()
	if err != nil {
		return Producer{}, err
	}

	return producer, nil
}
