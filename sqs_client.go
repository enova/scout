package main

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// SQSClient is an interface for SQS
type SQSClient interface {
	// Fetch returns the next batch of SQS messages
	Fetch() ([]*Message, error)

	// Delete deletes a single message from SQS
	Delete(*Message) error
}

// Message is the internal representation of an SQS message
type Message struct {
	MessageID     string
	Body          string
	ReceiptHandle string
}

type sdkClient struct {
	service *sqs.SQS
	url     string
}

// NewAWSSQSClient creates an SQS client that talks to AWS on the given queue
func NewAWSSQSClient(conf AWSConfig, queueName string) (SQSClient, error) {
	creds := credentials.NewStaticCredentials(conf.AccessKey, conf.SecretKey, "")
	sess := session.New(&aws.Config{Region: formatRegion(conf.Region), Credentials: creds})

	client := new(sdkClient)
	client.service = sqs.New(sess)

	resp, err := client.service.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: &queueName,
	})

	if err != nil {
		return nil, err
	}

	client.url = *resp.QueueUrl
	return client, nil
}

func (s *sdkClient) Fetch() ([]*Message, error) {
	res, err := s.service.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl:            &s.url,
		MaxNumberOfMessages: aws.Int64(10),
		WaitTimeSeconds:     aws.Int64(20),
	})
	if err != nil {
		return nil, err
	}

	msgs := make([]*Message, len(res.Messages))

	for i, m := range res.Messages {
		msgs[i] = &Message{
			MessageID:     *m.MessageId,
			Body:          *m.Body,
			ReceiptHandle: *m.ReceiptHandle,
		}
	}

	return msgs, nil
}

func (s *sdkClient) Delete(message *Message) error {
	_, err := s.service.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      &s.url,
		ReceiptHandle: &message.ReceiptHandle,
	})
	return err
}

func formatRegion(region string) *string {
	newRegion := strings.NewReplacer(".", "-", "_", "-").Replace(region)
	return &newRegion
}
