package queue

import (
	"azure-service-bus/metrics"
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"nhooyr.io/websocket"

	azlog "github.com/Azure/azure-sdk-for-go/sdk/azcore/log"
)

// AzureWriteQueue Write-only handle on service bus queue
type AzureWriteQueue struct {
	queueOrTopicName string
	*azservicebus.Sender
}

// AzureReadQueue Read-only handle on service bus queue
type AzureReadQueue struct {
	queueOrTopicName string
	subscriptionName string
	*azservicebus.Receiver
}

// AzureReadWriteQueue Handle on service bus queue
type AzureReadWriteQueue struct {
	queueName string
	AzureWriteQueue
	AzureReadQueue
}

// initGenericServiceBusQueue initialise azure service bus client tied to a given queue
func initGenericServiceBusQueue(ctx context.Context, queue, connString string) (*AzureReadWriteQueue, error) {
	receiver, err := initReadOnlyServiceBusQueue(
		ctx,
		queue,
		connString,
	)
	if err != nil {
		return &AzureReadWriteQueue{}, err
	}
	sender, err := initWriteOnlyServiceBus(
		ctx,
		queue,
		connString,
	)
	if err != nil {
		return &AzureReadWriteQueue{}, err
	}
	return &AzureReadWriteQueue{
		AzureWriteQueue: *sender,
		AzureReadQueue:  *receiver,
		queueName:       queue,
	}, nil
}

// initWriteOnlyServiceBus initialise azure service bus client tied to send to a given queue or topic
func initWriteOnlyServiceBus(_ context.Context, queueOrTopic, connString string) (*AzureWriteQueue, error) {
	if true {
		enableSBLogs()
	}
	client, err := newClient(connString, 500*time.Millisecond, 1*time.Second)
	if err != nil {
		return &AzureWriteQueue{}, err
	}

	sender, err := client.NewSender(queueOrTopic, nil)

	if err != nil {
		return &AzureWriteQueue{}, err
	}
	return &AzureWriteQueue{
		Sender:           sender,
		queueOrTopicName: queueOrTopic,
	}, nil
}

// initReadOnlyServiceBusQueue initialise azure service bus client tied to receive from a given queue
func initReadOnlyServiceBusQueue(_ context.Context, queue, connString string) (*AzureReadQueue, error) {
	if true {
		enableSBLogs()
	}
	client, err := newClient(connString, 1*time.Second, 2*time.Second)
	if err != nil {
		return &AzureReadQueue{}, err
	}

	receiver, err := client.NewReceiverForQueue(
		queue,
		&azservicebus.ReceiverOptions{
			ReceiveMode: azservicebus.ReceiveModePeekLock,
		},
	)

	if err != nil {
		return &AzureReadQueue{}, err
	}
	return &AzureReadQueue{
		Receiver:         receiver,
		queueOrTopicName: queue,
	}, nil
}

// initSubscribeOnlyServiceBusTopic initialise azure service bus client tied to receive from a given topic
func initSubscribeOnlyServiceBusTopic(_ context.Context, topic, subscription, connString string) (*AzureReadQueue, error) {
	if true {
		enableSBLogs()
	}
	client, err := newClient(connString, 1*time.Second, 2*time.Second)
	if err != nil {
		return &AzureReadQueue{}, err
	}

	receiver, err := client.NewReceiverForSubscription(
		topic,
		subscription,
		&azservicebus.ReceiverOptions{
			ReceiveMode: azservicebus.ReceiveModePeekLock,
		},
	)
	if err != nil {
		return &AzureReadQueue{}, err
	}

	return &AzureReadQueue{
		Receiver:         receiver,
		queueOrTopicName: topic,
		subscriptionName: subscription,
	}, nil
}

func newClient(connString string, retryDelay, maxRetryDelay time.Duration) (*azservicebus.Client, error) {
	return azservicebus.NewClientFromConnectionString(connString, &azservicebus.ClientOptions{
		RetryOptions: azservicebus.RetryOptions{
			MaxRetries:    3,
			RetryDelay:    retryDelay,
			MaxRetryDelay: maxRetryDelay,
		},
		NewWebSocketConn: func(ctx context.Context, args azservicebus.NewWebSocketConnArgs) (net.Conn, error) {
			opts := &websocket.DialOptions{
				HTTPClient: &http.Client{
					Transport: &http.Transport{
						// mirror the parent timeout on the read queue.
						TLSHandshakeTimeout: 30 * time.Second,
					},
				},
				Subprotocols: []string{"amqp"},
			}

			// we don't want to trace this low level connection, so use a new context here
			connTimeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			wssConn, _, err := websocket.Dial(connTimeout, args.Host, opts)

			if err != nil {
				return nil, err
			}

			return websocket.NetConn(context.Background(), wssConn, websocket.MessageBinary), nil
		},
	})
}

func enableSBLogs() {
	once := sync.Once{}
	once.Do(func() {
		azlog.SetEvents(
			// EventConn is used whenever we create a connection or any links (ie: receivers, senders).
			azservicebus.EventConn,
			// EventAuth is used when we're doing authentication/claims negotiation.
			azservicebus.EventAuth,
			// EventReceiver represents operations that happen on Receivers.
			azservicebus.EventReceiver,
			// EventSender represents operations that happen on Senders.
			azservicebus.EventSender,
			// EventAdmin is used for operations in the azservicebus/admin.Client
			azservicebus.EventAdmin,
		)

		// pass logs to our logging library
		azlog.SetListener(func(event azlog.Event, s string) {
			fmt.Printf("Service bus event: [%s] %s\n", event, s)
		})
	})
}

// BlockUntilMessage receive one from the bus queue, and blocks until it does so
func (r *AzureReadQueue) BlockUntilMessage(ctx context.Context) (*ReceivedMessage, error) {
	start := time.Now()

	messages, err := r.ReceiveMessages(ctx,
		1,
		nil,
	)

	tags := map[string]string{
		"queue_name": r.queueOrTopicName,
		"operation":  "block_until_message",
	}

	if err != nil {
		metrics.IncrWithTags("service_bus_error", tags)
		return nil, err
	}
	if len(messages) == 0 {
		metrics.IncrWithTags("service_bus_success", tags)
		return nil, nil
	}
	metrics.TimingWithTags("service_bus_latency", time.Since(start), tags)

	metrics.IncrWithTags("service_bus_success", tags)

	return &ReceivedMessage{
		SequenceNumber:  strconv.Itoa(int(*messages[0].SequenceNumber)),
		MessageID:       messages[0].MessageID,
		Body:            messages[0].Body,
		DeliveryCount:   messages[0].DeliveryCount,
		EnqueuedTime:    messages[0].EnqueuedTime,
		InternalMessage: messages[0],
	}, nil
}

// ReceiveOne receive one or zero messages from the bus queue
func (r *AzureReadQueue) ReceiveOne(ctx context.Context) (*ReceivedMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	start := time.Now()

	messages, err := r.ReceiveMessages(ctx,
		1,
		nil,
	)

	tags := map[string]string{
		"queue_name": r.queueOrTopicName,
		"operation":  "receive_one",
	}

	if err != nil {
		metrics.IncrWithTags("service_bus_error", tags)
		return nil, err
	}
	if len(messages) == 0 {
		return nil, nil
	}
	metrics.TimingWithTags("service_bus_latency", time.Since(start), tags)

	metrics.IncrWithTags("service_bus_success", tags)

	return &ReceivedMessage{
		SequenceNumber:  strconv.Itoa(int(*messages[0].SequenceNumber)),
		MessageID:       messages[0].MessageID,
		Body:            messages[0].Body,
		DeliveryCount:   messages[0].DeliveryCount,
		EnqueuedTime:    messages[0].EnqueuedTime,
		InternalMessage: messages[0],
	}, nil
}

// SendOne to queue
func (r *AzureWriteQueue) SendOne(ctx context.Context, correlationID string, contentType string, subject string, message []byte) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var err error
	span, tctx := tracer.StartSpanFromContext(ctx, "send", tracer.ServiceName("azure"), tracer.ResourceName("servicebus"))
	defer span.Finish(tracer.WithError(err))

	start := time.Now()

	messageStruct := azservicebus.Message{
		CorrelationID: &correlationID,
		Subject:       &subject,
		Body:          message,
		ContentType:   &contentType,
	}
	err = r.SendMessage(tctx, &messageStruct, nil)

	tags := map[string]string{
		"queue_name": r.queueOrTopicName,
		"operation":  "send_one",
	}

	metrics.TimingWithTags("service_bus_latency", time.Since(start), tags)
	if err != nil {
		metrics.IncrWithTags("service_bus_error", tags)
		return err
	}
	metrics.IncrWithTags("service_bus_success", tags)
	return nil
}

// ScheduleOne on queue
func (r *AzureWriteQueue) ScheduleOne(ctx context.Context, correlationID string, contentType string, subject string, message []byte, scheduleTime time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var err error
	span, tctx := tracer.StartSpanFromContext(ctx, "schedule", tracer.ServiceName("azure"), tracer.ResourceName("servicebus"))
	defer span.Finish(tracer.WithError(err))

	messageStruct := []*azservicebus.Message{
		{
			CorrelationID: &correlationID,
			Subject:       &subject,
			Body:          message,
			ContentType:   &contentType,
		},
	}

	start := time.Now()

	_, err = r.ScheduleMessages(tctx, messageStruct, scheduleTime, nil)

	tags := map[string]string{
		"queue_name": r.queueOrTopicName,
		"operation":  "schedule_one",
	}

	metrics.TimingWithTags("service_bus_latency", time.Since(start), tags)
	if err != nil {
		metrics.IncrWithTags("service_bus_error", tags)
		return err
	}
	metrics.IncrWithTags("service_bus_success", tags)
	return nil
}

// ReceiveMany receive up to count messages from the bus queue
func (r *AzureReadQueue) ReceiveMany(ctx context.Context, count int) ([]*ReceivedMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	start := time.Now()
	messages, err := r.ReceiveMessages(ctx,
		count,
		nil,
	)

	tags := map[string]string{
		"queue_name": r.queueOrTopicName,
		"operation":  "receive_many",
	}
	metrics.TimingWithTags("service_bus_latency", time.Since(start), tags)
	if err != nil {
		metrics.IncrWithTags("service_bus_error", tags)
		return nil, err
	}
	metrics.CountWithTags("service_bus_success", int64(len(messages)), tags)

	wrappers := make([]*ReceivedMessage, 0, cap(messages))
	for _, x := range messages {
		wrappers = append(wrappers, &ReceivedMessage{
			SequenceNumber:  strconv.Itoa(int(*x.SequenceNumber)),
			MessageID:       messages[0].MessageID,
			Body:            x.Body,
			DeliveryCount:   x.DeliveryCount,
			EnqueuedTime:    x.EnqueuedTime,
			InternalMessage: x,
		})
	}
	return wrappers, nil
}

// Complete completes processing of message, removing it from the bus queue
func (r *AzureReadQueue) Complete(ctx context.Context, message *ReceivedMessage) error {
	var err error
	span, tctx := tracer.StartSpanFromContext(ctx, "complete", tracer.ServiceName("azure"), tracer.ResourceName("servicebus"))
	defer span.Finish(tracer.WithError(err))

	tags := map[string]string{
		"queue_name": r.queueOrTopicName,
		"operation":  "complete",
	}

	start := time.Now()
	err = r.CompleteMessage(tctx, message.InternalMessage.(*azservicebus.ReceivedMessage), nil)
	metrics.TimingWithTags("service_bus_latency", time.Since(start), tags)

	if err != nil {
		metrics.IncrWithTags("service_bus_error", tags)
		fmt.Printf("attempt to complete message with id: %s failed, error: %v", message.SequenceNumber, err)
		return err
	}

	metrics.IncrWithTags("service_bus_success", tags)

	fmt.Printf("message with id: %s was completed", message.SequenceNumber)

	return nil
}

// Abandon abandon processing of message
func (r *AzureReadQueue) Abandon(ctx context.Context, message *ReceivedMessage) error {
	var err error
	span, tctx := tracer.StartSpanFromContext(ctx, "abandon", tracer.ResourceName("azure"), tracer.ResourceName("servicebus"))
	defer span.Finish(tracer.WithError(err))

	tags := map[string]string{
		"queue_name": r.queueOrTopicName,
		"operation":  "abandon",
	}

	start := time.Now()
	err = r.AbandonMessage(tctx, message.InternalMessage.(*azservicebus.ReceivedMessage), nil)
	metrics.TimingWithTags("service_bus_latency", time.Since(start), tags)

	if err != nil {
		metrics.IncrWithTags("service_bus_error", tags)
		fmt.Printf("attempt to abandon message with id: %s failed", message.SequenceNumber)
		return err
	}

	metrics.IncrWithTags("service_bus_success", tags)
	fmt.Printf("message with id: %s was abandoned, error: %v", message.SequenceNumber, err)
	return nil
}

func (r *AzureReadQueue) DeadLetter(ctx context.Context, message *ReceivedMessage) error {
	var err error
	span, tctx := tracer.StartSpanFromContext(ctx, "deadletter", tracer.ServiceName("azure"), tracer.ResourceName("servicebus"))
	defer span.Finish(tracer.WithError(err))

	tags := map[string]string{
		"queue_name": r.queueOrTopicName,
		"operation":  "deadletter",
	}

	start := time.Now()
	err = r.DeadLetterMessage(tctx, message.InternalMessage.(*azservicebus.ReceivedMessage), nil)
	metrics.TimingWithTags("service_bus_latency", time.Since(start), tags)

	if err != nil {
		metrics.IncrWithTags("service_bus_error", tags)
		fmt.Printf("attempt to dead letter message with id: %s failed, error: %v", message.SequenceNumber, err)
		return err
	}

	metrics.IncrWithTags("service_bus_success", tags)
	fmt.Printf("message with id: %s was dead letter'd", message.SequenceNumber)

	return nil
}

// RenewLock renew the lock on a message
func (r *AzureReadQueue) RenewLock(ctx context.Context, message *ReceivedMessage) error {
	var err error
	span, tctx := tracer.StartSpanFromContext(ctx, "renew", tracer.ServiceName("azure"), tracer.ResourceName("servicebus"))
	defer span.Finish(tracer.WithError(err))

	tags := map[string]string{
		"queue_name": r.queueOrTopicName,
		"operation":  "renew",
	}

	start := time.Now()
	err = r.RenewMessageLock(tctx, message.InternalMessage.(*azservicebus.ReceivedMessage), nil)
	metrics.TimingWithTags("service_bus_latency", time.Since(start), tags)

	if err != nil {
		metrics.IncrWithTags("service_bus_error", tags)
		fmt.Printf("attempt to relock message with id: %s failed, error: %v", message.SequenceNumber, err)
		return err
	}

	metrics.IncrWithTags("service_bus_success", tags)
	fmt.Printf("message with id: %s was relocked", message.SequenceNumber)
	return nil
}
