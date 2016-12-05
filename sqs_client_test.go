// +build sqsint

package main

import (
	"testing"

	"github.com/goamz/goamz/sqs"
	"github.com/stretchr/testify/require"
)

// These tests are hardcoded to point to actual sqs queues and sns topics.
// You should think of them as "integration" tests, and in general you don't
// want to be running these all the time. You will need to provide your own
// credentials below to run these.

var config = AWSConfig{
	AccessKey: "",
	SecretKey: "",
	Region:    "",
}

var queueName = ""

func queueHandle() *sqs.Queue {
	sqsHandle, err := sqs.NewFrom(config.AccessKey, config.SecretKey, config.Region)
	if err != nil {
		panic(err)
	}

	queue, err := sqsHandle.GetQueue(queueName)
	if err != nil {
		panic(err)
	}

	return queue
}

func TestSQS_Init(t *testing.T) {
	_, err := NewAWSSQSClient(config, queueName)
	require.NoError(t, err)

	// wrong region
	_, err = NewAWSSQSClient(
		AWSConfig{
			AccessKey: "AKIAJNRPNGF7HIWQ5C6Q",
			SecretKey: "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",
			Region:    "us.best",
		},
		"test-queue-integration",
	)
	require.Error(t, err)

	// wrong creds
	_, err = NewAWSSQSClient(
		AWSConfig{
			AccessKey: "super",
			SecretKey: "secret",
			Region:    "us.west.2",
		},
		"test-queue-integration",
	)
	require.Error(t, err)

	// wrong queue
	_, err = NewAWSSQSClient(
		AWSConfig{
			AccessKey: "AKIAJNRPNGF7HIWQ5C6Q",
			SecretKey: "myTX5YypzjqjgtZJ2ABvwqotGazqxtj37yQwyZpa",
			Region:    "us.west.2",
		},
		"fake-queue", // hopefully nobody ever makes this
	)
	require.Error(t, err)
}

func TestSQS_FetchDelete(t *testing.T) {
	recd := map[string]int{
		"foo": 0,
		"bar": 0,
		"baz": 0,
	}
	queue := queueHandle()
	_, err := queue.SendMessage("foo")
	require.NoError(t, err)
	_, err = queue.SendMessage("bar")
	require.NoError(t, err)
	_, err = queue.SendMessage("baz")
	require.NoError(t, err)

	client, err := NewAWSSQSClient(config, queueName)
	require.NoError(t, err)

	// Loop over and read from the queue unitl there are no messages left.
	// Doing it this way because even though we set max messages to 10, it
	// seems that aws almost always gives us back only one anyway
	for {
		messages, err := client.Fetch()
		require.NoError(t, err)
		if len(messages) == 0 {
			break
		}

		for _, msg := range messages {
			recd[msg.Body] += 1
			err := client.Delete(msg)
			require.NoError(t, err)
		}
	}

	require.Equal(t, recd["foo"], 1)
	require.Equal(t, recd["bar"], 1)
	require.Equal(t, recd["baz"], 1)
}

func TestSQS_FetchMany(t *testing.T) {
	queue := queueHandle()

	// We're filling up the queue to ensure that a call to Fetch will
	// actually return 10 messages. More sanity than anything else, don't
	// be too concerned if this fails
	for i := 0; i < 100; i++ {
		_, err := queue.SendMessage("foo")
		require.NoError(t, err)
	}

	client, err := NewAWSSQSClient(config, queueName)
	require.NoError(t, err)

	messages, err := client.Fetch()
	require.NoError(t, err)
	require.Equal(t, len(messages), 10)

	_, err = queue.Purge()
	require.NoError(t, err)
}
