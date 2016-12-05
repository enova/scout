package main

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"

	log "github.com/Sirupsen/logrus"
)

// Queue is an encasulation for processing an SQS queue and enqueueing the
// results in sidekiq
type Queue interface {
	// Poll gets the next batch of messages from SQS and processes them.
	// When it's finished, it downs the sempahore
	Poll()

	// Semaphore returns the lock used to ensure that all the work is
	// done before terminating the queue
	Semaphore() *sync.WaitGroup
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

func (q *queue) Poll() {
	if q.Sem != nil {
		defer q.Sem.Done()
	}

	messages, err := q.SQSClient.Fetch()
	if err != nil {
		log.Error("Error fetching messages: ", err.Error())
	}

	for _, msg := range messages {
		log.Info("Processing message: ", msg.MessageID)
		deletable := q.EnqueueMessage(msg)
		if deletable {
			q.DeleteMessage(msg)
		}
	}
}

// DeleteMessage deletes a single message from SQS
func (q *queue) DeleteMessage(msg Message) {
	err := q.SQSClient.Delete(msg)
	if err != nil {
		log.Error("Couldn't delete message: ", msg.MessageID)
	} else {
		log.Info("Deleted message: ", msg.MessageID)
	}
}

// EnqueueMessage pushes a single message from SQS into redis
func (q *queue) EnqueueMessage(msg Message) bool {
	ctx := log.WithField("MessageID", msg.MessageID)
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
		ctx.Error("Couldn't enqueue worker: ", workerClass)
		return false
	}

	ctx.WithField("Args", body["Message"]).Info("Enqueued job: ", jid)
	return true
}

func topicName(topicARN string) string {
	toks := strings.Split(topicARN, ":")
	return toks[len(toks)-1]
}
