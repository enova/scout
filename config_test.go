package main

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestConfig(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}

type ConfigTestSuite struct {
	suite.Suite
	tempfile *os.File
	assert   *require.Assertions
}

func (c *ConfigTestSuite) SetupTest() {
	c.assert = require.New(c.T())

	var err error
	c.tempfile, err = ioutil.TempFile("", "config")
	c.assert.NoError(err)
}

func (c *ConfigTestSuite) TearDownTest() {
	os.Remove(c.tempfile.Name())
}

func (c *ConfigTestSuite) WriteTemp(content string) {
	_, err := c.tempfile.Write([]byte(content))
	c.assert.NoError(err)
	ReadConfig(c.tempfile.Name())
	err = c.tempfile.Close()
	c.assert.NoError(err)
}

var validConfig = `
redis:
  host: "localhost:9000"
  namespace: "test"
  queue: "background"
aws:
  access_key: "super"
  secret_key: "secret"
  region: "us_best"
queue:
  name: "myapp_queue"
  topics:
    foo_topic: "FooWorker"
    bar_topic: "BazWorker"`

func (c *ConfigTestSuite) TestConfig_Valid() {
	c.WriteTemp(validConfig)
	config, err := ReadConfig(c.tempfile.Name())
	c.assert.NoError(err)

	// More to convince myself that the yaml package works than anything
	c.assert.Equal(config.Redis.Host, "localhost:9000")
	c.assert.Equal(config.Redis.Queue, "background")
	c.assert.Equal(config.AWS.Region, "us_best")
	c.assert.Equal(config.Queue.Name, "myapp_queue")
	c.assert.Equal(config.Queue.Topics["foo_topic"], "FooWorker")
}

var sparseConfig = `
redis:
  host: "localhost:9000"
aws:
  access_key: "super"
  secret_key: "secret"
  region: "us_best"`

// It's ok for stuff to be missing, we'll check that elsewhere
func (c *ConfigTestSuite) TestConfig_Sparse() {
	c.WriteTemp(sparseConfig)
	config, err := ReadConfig(c.tempfile.Name())
	c.assert.NoError(err)

	c.assert.Equal(config.Redis.Namespace, "")
	c.assert.Equal(config.AWS.Region, "us_best")
	c.assert.Equal(len(config.Queue.Topics), 0)
}
