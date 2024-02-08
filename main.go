package main

import (
	"azure-service-bus/work_number"
	"context"
	"fmt"
)

// func GetClient() *azservicebus.Client {
// 	namespace, ok := os.LookupEnv("AZURE_SERVICEBUS_HOSTNAME") //ex: myservicebus.servicebus.windows.net
// 	if !ok {
// 		panic("AZURE_SERVICEBUS_HOSTNAME environment variable not found")
// 	}

// 	cred, err := azidentity.NewDefaultAzureCredential(nil)
// 	if err != nil {
// 		panic(err)
// 	}

// 	client, err := azservicebus.NewClient(namespace, cred, nil)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return client
// }

// func SendMessage(message string, client *azservicebus.Client) {
// 	sender, err := client.NewSender("myqueue", nil)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer sender.Close(context.TODO())

// 	sbMessage := &azservicebus.Message{
// 		Body: []byte(message),
// 	}
// 	err = sender.SendMessage(context.TODO(), sbMessage, nil)
// 	if err != nil {
// 		panic(err)
// 	}
// }

// func SendMessageBatch(messages []string, client *azservicebus.Client) {
// 	sender, err := client.NewSender("myqueue", nil)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer sender.Close(context.TODO())

// 	batch, err := sender.NewMessageBatch(context.TODO(), nil)
// 	if err != nil {
// 		panic(err)
// 	}

// 	for _, message := range messages {
// 		err := batch.AddMessage(&azservicebus.Message{Body: []byte(message)}, nil)
// 		if errors.Is(err, azservicebus.ErrMessageTooLarge) {
// 			fmt.Printf("Message batch is full. We should send it and create a new one.\n")
// 		}
// 	}

// 	if err := sender.SendMessageBatch(context.TODO(), batch, nil); err != nil {
// 		panic(err)
// 	}
// }

// func GetMessage(count int, client *azservicebus.Client) {
// 	receiver, err := client.NewReceiverForQueue("myqueue", nil)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer receiver.Close(context.TODO())

// 	messages, err := receiver.ReceiveMessages(context.TODO(), count, nil)
// 	if err != nil {
// 		panic(err)
// 	}

// 	for _, message := range messages {
// 		body := message.Body
// 		fmt.Printf("%s\n", string(body))

// 		err = receiver.CompleteMessage(context.TODO(), message, nil)
// 		if err != nil {
// 			panic(err)
// 		}
// 	}
// }

// func DeadLetterMessage(client *azservicebus.Client) {
// 	deadLetterOptions := &azservicebus.DeadLetterOptions{
// 		ErrorDescription: to.Ptr("exampleErrorDescription"),
// 		Reason:           to.Ptr("exampleReason"),
// 	}

// 	receiver, err := client.NewReceiverForQueue("myqueue", nil)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer receiver.Close(context.TODO())

// 	messages, err := receiver.ReceiveMessages(context.TODO(), 1, nil)
// 	if err != nil {
// 		panic(err)
// 	}

// 	if len(messages) == 1 {
// 		err := receiver.DeadLetterMessage(context.TODO(), messages[0], deadLetterOptions)
// 		if err != nil {
// 			panic(err)
// 		}
// 	}
// }

// func GetDeadLetterMessage(client *azservicebus.Client) {
// 	receiver, err := client.NewReceiverForQueue(
// 		"myqueue",
// 		&azservicebus.ReceiverOptions{
// 			SubQueue: azservicebus.SubQueueDeadLetter,
// 		},
// 	)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer receiver.Close(context.TODO())

// 	messages, err := receiver.ReceiveMessages(context.TODO(), 1, nil)
// 	if err != nil {
// 		panic(err)
// 	}

// 	for _, message := range messages {
// 		fmt.Printf("DeadLetter Reason: %s\nDeadLetter Description: %s\n", *message.DeadLetterReason, *message.DeadLetterErrorDescription)
// 		err := receiver.CompleteMessage(context.TODO(), message, nil)
// 		if err != nil {
// 			panic(err)
// 		}
// 	}
// }

func main() {
	ctx := context.Background()
	worknumberClient, err := work_number.NewWorkNumberClient(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}
	ip := work_number.WorkNumberInput{
		ConsumerID:    "1234567",
		ApplicationID: "1234567890",
		SSN:           "123456789",
		FirstName:     "qeqwew",
		LastName:      "dgfret",
		EmployerName:  "tgrr",
		EmployerCode:  "9090",
		SalaryKey:     "ertert",
	}
	fmt.Println("send a single message...")
	err = worknumberClient.SendWorkNumberRequest(ctx, &ip)
	if err != nil {
		fmt.Println(err)
		return
	}
	op, err := worknumberClient.ReceiveWorkNumberResponse(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("op from topic", op)

}
