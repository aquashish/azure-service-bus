package queue

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQBus struct {
	SubChannel *amqp.Channel
	PubChannel *amqp.Channel
	Queue      amqp.Queue
}

func initRabbitMQBus(queueName, connString string) (ReadWriteInterface, error) {
	conn, err := amqp.Dial(connString)
	if err != nil {
		return nil, err
	}
	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	queue, err := channel.QueueDeclare(queueName, true, false, false, true, nil)
	if err != nil {
		return nil, err
	}

	pubChan, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	return &RabbitMQBus{
		SubChannel: channel,
		PubChannel: pubChan,
		Queue:      queue,
	}, nil
}

func (r *RabbitMQBus) SendOne(ctx context.Context, correlationID string, contentType string, _ string, message []byte) error {
	err := r.PubChannel.PublishWithContext(
		ctx,
		"",
		r.Queue.Name,
		false,
		false,
		amqp.Publishing{
			Body:          message,
			ContentType:   contentType,
			CorrelationId: correlationID,
		})

	return err
}

func (r *RabbitMQBus) ScheduleOne(ctx context.Context, correlationID string, contentType string, subject string, message []byte, scheduledAt time.Time) error {
	// this isn't something rabbit natively supports, so fake it with a go-routine
	go func() {
		time.Sleep(time.Until(scheduledAt))
		if err := r.SendOne(context.Background(), correlationID, contentType, subject, message); err != nil {
			fmt.Printf("Failed to send scheduled message %s: %v", correlationID, err)
		}
	}()
	return nil
}

func (r *RabbitMQBus) BlockUntilMessage(_ context.Context) (*ReceivedMessage, error) {
	consumerKey := uuid.New().String()
	del, err := r.SubChannel.Consume(
		r.Queue.Name,
		consumerKey,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = r.SubChannel.Cancel(consumerKey, false)
	}()

	msg := <-del

	return &ReceivedMessage{
		SequenceNumber:  strconv.FormatUint(msg.DeliveryTag, 10),
		MessageID:       msg.MessageId,
		Body:            msg.Body,
		DeliveryCount:   0,
		EnqueuedTime:    &msg.Timestamp,
		InternalMessage: &msg,
	}, nil
}

func (r *RabbitMQBus) ReceiveOne(_ context.Context) (*ReceivedMessage, error) {
	consumerKey := uuid.New().String()
	del, err := r.SubChannel.Consume(
		r.Queue.Name,
		consumerKey,
		false,
		false,
		false,
		false,
		nil,
	)
	defer func() {
		_ = r.SubChannel.Cancel(consumerKey, false)
	}()

	if err != nil {
		return nil, err
	}
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	select {
	case <-ticker.C:
		//timeout
		return nil, errors.New("timeout")
	case msg := <-del:
		//msg
		return &ReceivedMessage{
			SequenceNumber:  msg.MessageId,
			MessageID:       msg.MessageId,
			Body:            msg.Body,
			DeliveryCount:   0,
			EnqueuedTime:    &msg.Timestamp,
			InternalMessage: &msg,
		}, nil
	}
}

func (r *RabbitMQBus) ReceiveMany(ctx context.Context, _ int) ([]*ReceivedMessage, error) {
	one, err := r.ReceiveOne(ctx)
	if err != nil {
		return nil, err
	}
	return []*ReceivedMessage{one}, nil
}

func (r *RabbitMQBus) DeadLetter(_ context.Context, message *ReceivedMessage) error {
	return r.SubChannel.Nack((message.InternalMessage.(*amqp.Delivery)).DeliveryTag, false, false)
}

func (r *RabbitMQBus) Abandon(_ context.Context, message *ReceivedMessage) error {
	//rabbitmq has no concept of a delivery count, so it will indefinitely retry with requeue set to true
	err := r.SubChannel.Nack((message.InternalMessage.(*amqp.Delivery)).DeliveryTag, false, false)
	return err
}

func (r *RabbitMQBus) Complete(_ context.Context, message *ReceivedMessage) error {
	err := r.SubChannel.Ack((message.InternalMessage.(*amqp.Delivery)).DeliveryTag, false)
	return err
}

func (r *RabbitMQBus) RenewLock(_ context.Context, _ *ReceivedMessage) error {
	// not implemented by rabbitmq
	return nil
}
