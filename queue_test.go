package main

import (
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
		Fetchable:   make([]*Message, 0),
		FetchError:  nil,
		Deleted:     make([]*Message, 0),
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

func (q *QueueTestSuite) TestPoll_Success() {
	// make some messages
	message1 := MockMessage(`{"foo":"bar"}`, "topicA")
	message2 := MockMessage(`{"bar":"baz"}`, "topicA")
	message3 := MockMessage(`{"key":"val"}`, "topicB")
	badMessage := &Message{
		Body: `thisain'tjson`,
	}

	// set the mock to return those
	q.sqsClient.Fetchable = []*Message{message1, message3, message2, badMessage}

	// do the work
	messageQueue := make(chan *Message, 10)
	go q.queue.Poll(messageQueue)

	// they should all be in the queue
	q.assert.Equal(<-messageQueue, message1)
	q.assert.Equal(<-messageQueue, message3)
	q.assert.Equal(<-messageQueue, message2)
	q.assert.Equal(<-messageQueue, badMessage)
}

func (q *QueueTestSuite) TestProcess_Success() {
	// make some messages
	message1 := MockMessage(`{"foo":"bar"}`, "topicA")
	message2 := MockMessage(`{"bar":"baz"}`, "topicA")
	message3 := MockMessage(`{"key":"val"}`, "topicB")

	// make some topics
	q.queue.Topics["topicA"] = "WorkerA"
	q.queue.Topics["topicB"] = "WorkerB"

	// process some messages
	q.queue.ProcessMessage(message1)
	q.queue.ProcessMessage(message2)
	q.queue.ProcessMessage(message3)

	// The workers should be enqueued
	q.assert.Contains(q.workerClient.Enqueued, []string{"WorkerA", `{"foo":"bar"}`})
	q.assert.Contains(q.workerClient.Enqueued, []string{"WorkerA", `{"bar":"baz"}`})
	q.assert.Contains(q.workerClient.Enqueued, []string{"WorkerB", `{"key":"val"}`})

	// The messages should be deleted
	q.assert.Contains(q.sqsClient.Deleted, message1)
	q.assert.Contains(q.sqsClient.Deleted, message2)
	q.assert.Contains(q.sqsClient.Deleted, message3)
}

func (q *QueueTestSuite) TestProcess_NoTopic() {
	// make some messages
	message1 := MockMessage(`{"foo":"bar"}`, "topicA")
	message2 := MockMessage(`{"key":"val"}`, "topicB")

	// make some topics
	// note: there is no topicB
	q.queue.Topics["topicA"] = "WorkerA"

	// do the work
	q.queue.ProcessMessage(message1)
	q.queue.ProcessMessage(message2)

	// The workers should be enqueued
	// note: the topic B message is not enqueued
	q.assert.Contains(q.workerClient.Enqueued, []string{"WorkerA", `{"foo":"bar"}`})
	q.assert.Equal(len(q.workerClient.Enqueued), 1)

	// The messages should be deleted
	// note: message 2 is still deleted
	q.assert.Contains(q.sqsClient.Deleted, message1)
	q.assert.Contains(q.sqsClient.Deleted, message2)
}

func (q *QueueTestSuite) TestProcess_UnparseableBody() {
	// make some messages
	message1 := MockMessage(`{"foo":"bar"}`, "topicA")
	message2 := MockMessage(`{"bar":"baz"}`, "topicB")

	// this message has an unparseable body
	badMessage := &Message{
		Body: `thisain'tjson`,
	}

	// make some topics
	q.queue.Topics["topicA"] = "WorkerA"
	q.queue.Topics["topicB"] = "WorkerB"

	// do the work
	q.queue.ProcessMessage(message1)
	q.queue.ProcessMessage(message2)
	q.queue.ProcessMessage(badMessage)

	// The workers should be enqueued
	// note: the unparseable worker is not enqueued
	q.assert.Contains(q.workerClient.Enqueued, []string{"WorkerA", `{"foo":"bar"}`})
	q.assert.Contains(q.workerClient.Enqueued, []string{"WorkerB", `{"bar":"baz"}`})
	q.assert.Equal(len(q.workerClient.Enqueued), 2)

	// The messages should be deleted
	// note: the badMessage is deleted
	q.assert.Contains(q.sqsClient.Deleted, message1)
	q.assert.Contains(q.sqsClient.Deleted, message2)
	q.assert.Contains(q.sqsClient.Deleted, badMessage)
}

func (q *QueueTestSuite) TestProcess_EnqueueError() {
	// make a messages
	message1 := MockMessage(`{"foo":"bar"}`, "topicA")

	// set the mock to return that
	q.sqsClient.Fetchable = []*Message{message1}

	// make a topic
	q.queue.Topics["topicA"] = "WorkerA"

	// set the worker client to error out
	q.workerClient.EnqueueError = errors.New("oops")

	// process the message
	q.queue.ProcessMessage(message1)

	// nothing should be deleted
	q.assert.Empty(q.sqsClient.Deleted)
}

func (q *QueueTestSuite) TestProcess_Semaphore() {
	q.queue.Sem = new(sync.WaitGroup)
	q.queue.Semaphore().Add(1)
	q.queue.ProcessMessage(MockMessage("fake", "data"))

	// Calling Done() on a waitgroup that's at 0 will segfault
	q.assert.Panics(func() {
		q.queue.Semaphore().Done()
	})
}

func TestTopicName(t *testing.T) {
	// from http://docs.aws.amazon.com/sns/latest/dg/SendMessageToSQS.html
	require.Equal(t, topicName("arn:aws:sns:us-west-2:123456789012:MyTopic"), "MyTopic")
}
