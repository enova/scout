package main

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestQueue(t *testing.T) {
	suite.Run(t, new(QueueTestSuite))
}

type QueueTestSuite struct {
	suite.Suite
	queue        *queue
	sqsClient    *MockSQSClient
	workerClient *MockWorkerClient
	assert       *require.Assertions
}

func (q *QueueTestSuite) SetupTest() {
	q.assert = require.New(q.T())

	q.queue = new(queue)

	q.sqsClient = &MockSQSClient{
		Fetchable:   make([]Message, 0),
		FetchError:  nil,
		Deleted:     make([]Message, 0),
		DeleteError: nil,
	}

	q.queue.SQSClient = q.sqsClient

	q.workerClient = &MockWorkerClient{
		Enqueued:     make([][]string, 0),
		EnqueuedJID:  "jid",
		EnqueueError: nil,
	}

	q.queue.WorkerClient = q.workerClient

	q.queue.Topics = make(map[string]string)
}

func (q *QueueTestSuite) TestQueue_Broken() {
	q.assert.Equal(true, false, "this should fail")
}

func (q *QueueTestSuite) TestQueue_Success() {
	// make some messages
	message1 := MockMessage(`{"foo":"bar"}`, "topicA")
	message2 := MockMessage(`{"bar":"baz"}`, "topicA")
	message3 := MockMessage(`{"key":"val"}`, "topicB")

	// set the mock to return those
	q.sqsClient.Fetchable = []Message{message1, message3, message2}

	// make some topics
	q.queue.Topics["topicA"] = "WorkerA"
	q.queue.Topics["topicB"] = "WorkerB"

	// do the work
	q.queue.Poll()

	// The workers should be enqueued
	q.assert.Contains(q.workerClient.Enqueued, []string{"WorkerA", `{"foo":"bar"}`})
	q.assert.Contains(q.workerClient.Enqueued, []string{"WorkerA", `{"bar":"baz"}`})
	q.assert.Contains(q.workerClient.Enqueued, []string{"WorkerB", `{"key":"val"}`})

	// The messages should be deleted
	q.assert.Contains(q.sqsClient.Deleted, message1)
	q.assert.Contains(q.sqsClient.Deleted, message2)
	q.assert.Contains(q.sqsClient.Deleted, message3)
}

func (q *QueueTestSuite) TestQueue_NoTopic() {
	// make some messages
	message1 := MockMessage(`{"foo":"bar"}`, "topicA")
	message2 := MockMessage(`{"bar":"baz"}`, "topicA")
	message3 := MockMessage(`{"key":"val"}`, "topicB")

	// set the mock to return those
	q.sqsClient.Fetchable = []Message{message1, message3, message2}

	// make some topics
	// note: there is no topicB
	q.queue.Topics["topicA"] = "WorkerA"

	// do the work
	q.queue.Poll()

	// The workers should be enqueued
	// note: the topic B message is not enqueued
	q.assert.Contains(q.workerClient.Enqueued, []string{"WorkerA", `{"foo":"bar"}`})
	q.assert.Contains(q.workerClient.Enqueued, []string{"WorkerA", `{"bar":"baz"}`})

	// The messages should be deleted
	// note: message 3 is still deleted
	q.assert.Contains(q.sqsClient.Deleted, message1)
	q.assert.Contains(q.sqsClient.Deleted, message2)
	q.assert.Contains(q.sqsClient.Deleted, message3)
}

func (q *QueueTestSuite) TestQueue_UnparseableBody() {
	// make some messages
	message1 := MockMessage(`{"foo":"bar"}`, "topicA")
	message2 := MockMessage(`{"bar":"baz"}`, "topicB")

	// this message has an unparseable body
	badMessage := Message{
		Body: `thisain'tjson`,
	}

	// set the mock to return those
	q.sqsClient.Fetchable = []Message{message1, badMessage, message2}

	// make some topics
	q.queue.Topics["topicA"] = "WorkerA"
	q.queue.Topics["topicB"] = "WorkerB"

	// do the work
	q.queue.Poll()

	// The workers should be enqueued
	// note: the unparseable worker is not enqueued
	q.assert.Contains(q.workerClient.Enqueued, []string{"WorkerA", `{"foo":"bar"}`})
	q.assert.Contains(q.workerClient.Enqueued, []string{"WorkerB", `{"bar":"baz"}`})

	// The messages should be deleted
	// note: the badMessage is deleted
	q.assert.Contains(q.sqsClient.Deleted, message1)
	q.assert.Contains(q.sqsClient.Deleted, message2)
	q.assert.Contains(q.sqsClient.Deleted, badMessage)
}

func (q *QueueTestSuite) TestQueue_ComplexBody() {
	// make a message with a body that cannot be represented as map[string]string
	msg := map[string]interface{}{
		"Message":  `{"foo":"bar"}`,
		"TopicArn": "topicA",
		"Some":     map[string]string{"other": "data"},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}

	message := Message{Body: string(data)}

	// set the mock to return that message
	q.sqsClient.Fetchable = []Message{message}

	// make a some topics
	q.queue.Topics["topicA"] = "WorkerA"

	// do the work
	q.queue.Poll()

	// The worker should be enqueued
	q.assert.Contains(q.workerClient.Enqueued, []string{"WorkerA", `{"foo":"bar"}`})

	// The message should be deleted
	q.assert.Contains(q.sqsClient.Deleted, message)
}

func (q *QueueTestSuite) TestQueue_EnqueueError() {
	// make a messages
	message1 := MockMessage(`{"foo":"bar"}`, "topicA")

	// set the mock to return that
	q.sqsClient.Fetchable = []Message{message1}

	// make a topic
	q.queue.Topics["topicA"] = "WorkerA"

	// set the worker client to error out
	q.workerClient.EnqueueError = errors.New("oops")

	// do the work
	q.queue.Poll()

	// nothing should be deleted
	q.assert.Empty(q.sqsClient.Deleted)
}

func (q *QueueTestSuite) TestQueue_Semaphore() {
	q.queue.Sem = new(sync.WaitGroup)
	q.queue.Semaphore().Add(1)
	q.queue.Poll()

	// Calling Done() on a waitgroup that's at 0 will segfault
	q.assert.Panics(func() {
		q.queue.Semaphore().Done()
	})
}

func TestTopicName(t *testing.T) {
	// from http://docs.aws.amazon.com/sns/latest/dg/SendMessageToSQS.html
	require.Equal(t, topicName("arn:aws:sns:us-west-2:123456789012:MyTopic"), "MyTopic")
}
