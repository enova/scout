package main

import (
	"encoding/json"
)

func MockMessage(body, topic string) Message {
	msg := map[string]string{
		"Message":  body,
		"TopicArn": topic,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}

	return Message{Body: string(data)}
}

type MockSQSClient struct {
	Fetchable   []Message
	FetchError  error
	Deleted     []Message
	DeleteError error
}

func (m *MockSQSClient) Fetch() ([]Message, error) {
	if m.FetchError != nil {
		return nil, m.FetchError
	}

	return m.Fetchable, nil
}

func (m *MockSQSClient) Delete(message Message) error {
	m.Deleted = append(m.Deleted, message)
	return m.DeleteError
}

type MockWorkerClient struct {
	Enqueued     [][]string
	EnqueuedJID  string
	EnqueueError error
}

func (m *MockWorkerClient) Push(class, args string) (string, error) {
	m.Enqueued = append(m.Enqueued, []string{class, args})
	return m.EnqueuedJID, m.EnqueueError
}
