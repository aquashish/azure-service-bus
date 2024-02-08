package queue

import (
	"context"
	"strings"
	"time"
)

//go:generate mockgen -source=$GOFILE -destination=mock/$GOFILE
type ReadWriteInterface interface {
	ReadInterface
	WriteInterface
}

// ReceivedMessage Message received from service bus
type ReceivedMessage struct {
	MessageID       string
	SequenceNumber  string
	Body            []byte
	EnqueuedTime    *time.Time
	DeliveryCount   uint32
	InternalMessage any
}

// ReadInterface interface for interacting with existing messages on the bus
type ReadInterface interface {
	BlockUntilMessage(ctx context.Context) (*ReceivedMessage, error)
	ReceiveOne(ctx context.Context) (*ReceivedMessage, error)
	ReceiveMany(ctx context.Context, count int) ([]*ReceivedMessage, error)
	Abandon(ctx context.Context, message *ReceivedMessage) error
	Complete(ctx context.Context, message *ReceivedMessage) error
	DeadLetter(ctx context.Context, message *ReceivedMessage) error
	RenewLock(ctx context.Context, message *ReceivedMessage) error
}

// WriteInterface interface for interacting with existing messages
type WriteInterface interface {
	SendOne(ctx context.Context, correlationID string, contentType string, subject string, message []byte) error
	ScheduleOne(ctx context.Context, correlationID string, contentType string, subject string, message []byte, scheduleTime time.Time) error
}

// initGenericServiceBusQueue initialise azure service bus client tied to a given queue
func InitGenericBusQueue(ctx context.Context, queue, connString string) (ReadWriteInterface, error) {
	if useLocalTransport(connString) {
		return initRabbitMQBus(queue, connString)
	}
	return initGenericServiceBusQueue(ctx, queue, connString)
}

// initWriteOnlyServiceBusQueue initialise azure service bus client tied to a given queue
func InitWriteOnlyBusQueue(ctx context.Context, queue, connString string) (WriteInterface, error) { // queueOrTopic works for both Azure and RabbitMQ?
	if useLocalTransport(connString) {
		return initRabbitMQBus(queue, connString) // ?
	}
	return initWriteOnlyServiceBus(ctx, queue, connString)
}

func InitReadOnlyBusQueue(ctx context.Context, queue, connString string) (ReadInterface, error) {
	if useLocalTransport(connString) {
		return initRabbitMQBus(queue, connString)
	}
	return initReadOnlyServiceBusQueue(ctx, queue, connString)
}

func InitSubscribeOnlyBusTopic(ctx context.Context, topic, subscription, connString string) (ReadInterface, error) {
	if useLocalTransport(connString) {
		return initRabbitMQPubSub(true, topic, connString)
	}
	return initSubscribeOnlyServiceBusTopic(ctx, topic, subscription, connString)
}

func InitPublishOnlyBusTopic(ctx context.Context, topic, connString string) (WriteInterface, error) {
	if useLocalTransport(connString) {
		return initRabbitMQPubSub(false, topic, connString)
	}
	return initWriteOnlyServiceBus(ctx, topic, connString)
}

func useLocalTransport(connString string) bool {
	return strings.Contains(connString, "amqp")
}
