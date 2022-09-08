package main

import (
	"encoding/json"
	"testing"

	"github.com/jrallison/go-workers"
	"github.com/stretchr/testify/require"
	"gopkg.in/redis.v5"
)

// These tests require an existing redis database that matches the config below.
// If one doesn't exist they'll probably fail

var config = RedisConfig{
	Host:      "localhost:6379",
	Namespace: "integration",
	Queue:     "testq",
}

func TestWorker_Init(t *testing.T) {
	_, err := NewRedisWorkerClient(config)
	require.NoError(t, err)

	// no host
	_, err = NewRedisWorkerClient(
		RedisConfig{
			Host:      "",
			Namespace: "integration",
			Queue:     "testq",
		},
	)
	require.Error(t, err)

	// no queue
	_, err = NewRedisWorkerClient(
		RedisConfig{
			Host:      "localhost:6379",
			Namespace: "integration",
			Queue:     "",
		},
	)
	require.Error(t, err)

	// no namespace, this doesn't error
	_, err = NewRedisWorkerClient(
		RedisConfig{
			Host:      "localhost:6379",
			Namespace: "",
			Queue:     "testq",
		},
	)
	require.NoError(t, err)
}

func TestWorker_Push(t *testing.T) {
	redisHandle := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	err := redisHandle.Del("integration:queue:testq").Err()
	require.NoError(t, err)

	client, err := NewRedisWorkerClient(config)
	require.NoError(t, err)

	fooMessage := `{"msg":"foo"}`
	barMessage := `{"msg":"bar"}`

	fooJID, err := client.Push("FooWorker", fooMessage)
	require.NoError(t, err)
	barJID, err := client.Push("BarWorker", barMessage)
	require.NoError(t, err)

	require.NotEqual(t, fooJID, barJID)

	// first thing we enqueued
	data, err := redisHandle.LPop("integration:queue:testq").Bytes()
	require.NoError(t, err)

	fooEnqueued := &workers.EnqueueData{}
	err = json.Unmarshal(data, fooEnqueued)
	require.NoError(t, err)

	require.Equal(t, fooEnqueued.Jid, fooJID)
	require.Equal(t, fooEnqueued.Class, "FooWorker")
	require.Equal(t, fooEnqueued.Args, []interface{}{map[string]interface{}{"msg": "foo"}})
	require.Equal(t, fooEnqueued.EnqueueOptions.Retry, true)

	// second thing we enqueued
	data, err = redisHandle.LPop("integration:queue:testq").Bytes()
	require.NoError(t, err)

	barEnqueued := &workers.EnqueueData{}
	err = json.Unmarshal(data, barEnqueued)
	require.NoError(t, err)

	require.Equal(t, barEnqueued.Jid, barJID)
	require.Equal(t, barEnqueued.Class, "BarWorker")
	require.Equal(t, barEnqueued.Args, []interface{}{map[string]interface{}{"msg": "bar"}})
	require.Equal(t, fooEnqueued.EnqueueOptions.Retry, true)

	// verify retry is flat
	barFlat := make(map[string]interface{})
	err = json.Unmarshal(data, &barFlat)
	require.NoError(t, err)

	require.Equal(t, barFlat["retry"], true)
}
