package main

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
)

// Queue is an encasulation for processing an SQS queue and enqueueing the
// results in sidekiq
type Queue interface {
	// Poll gets the next batch of messages from SQS and enqueues them in
	// the out channel
	Poll(out chan *Message)

	// Semaphore returns the lock used to ensure that all the work is
	// done before terminating the queue
	Semaphore() *sync.WaitGroup

	// ProcessMessage takes a single message, enqueues it in redis, and then
	// deletes it out of SQS. It downs the Semaphore on exit.
	ProcessMessage(msg *Message)
}

// queue is the actual implementation
type queue struct {
	WorkerClient WorkerClient
	SQSClient    SQSClient
	Topics       map[string]string
	Sem          *sync.WaitGroup
}

// NewQueue creates a new Queue from the given Config. Returns an error if
// something about the config is invalid
func NewQueue(config *Config) (Queue, error) {
	queue := new(queue)
	var err error

	queue.SQSClient, err = NewAWSSQSClient(config.AWS, config.Queue.Name)
	if err != nil {
		return nil, err
	}

	queue.WorkerClient, err = NewRedisWorkerClient(config.Redis)
	if err != nil {
		return nil, err
	}

	queue.Topics = config.Queue.Topics
	if len(queue.Topics) == 0 {
		return nil, errors.New("No topics defined")
	}

	queue.Sem = new(sync.WaitGroup)

	return queue, nil
}

func (q *queue) Semaphore() *sync.WaitGroup {
	return q.Sem
}

func (q *queue) Poll(out chan *Message) {
	log.Debug("Polling at: ", time.Now())
	messages, err := q.SQSClient.Fetch()
	if err != nil {
		log.Error("Error fetching messages: ", err.Error())
	}

	for _, msg := range messages {
		out <- msg
	}
}

func (q *queue) ProcessMessage(msg *Message) {
	if q.Sem != nil {
		defer q.Sem.Done()
	}

	ctx := log.WithField("MessageID", msg.MessageID)
	ctx.Info("Processing message")
	deletable := q.enqueueMessage(msg, ctx)
	if deletable {
		q.deleteMessage(msg, ctx)
	}
}

// deleteMessage deletes a single message from SQS
func (q *queue) deleteMessage(msg *Message, ctx log.FieldLogger) {
	err := q.SQSClient.Delete(msg)
	if err != nil {
		ctx.Error("Couldn't delete message: ", err.Error())
	} else {
		ctx.Info("Deleted message")
	}
}

// enqueueMessage pushes a single message from SQS into redis
func (q *queue) enqueueMessage(msg *Message, ctx log.FieldLogger) bool {
	body := make(map[string]string)
	err := json.Unmarshal([]byte(msg.Body), &body)
	if err != nil {
		ctx.Warn("Message body could not be parsed: ", err.Error())
		return true
	}

	workerClass, ok := q.Topics[topicName(body["TopicArn"])]
	if !ok {
		ctx.Warn("No worker for topic: ", topicName(body["TopicArn"]))
		return true
	}

	jid, err := q.WorkerClient.Push(workerClass, body["Message"])
	if err != nil {
		ctx.WithField("Class", workerClass).Error("Couldn't enqueue worker: ", err.Error())
		return false
	}

	ctx.WithField("Args", body["Message"]).Info("Enqueued job: ", jid)
	return true
}

func topicName(topicARN string) string {
	toks := strings.Split(topicARN, ":")
	return toks[len(toks)-1]
}
