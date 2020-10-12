package main

import (
	"flag"
	"os"
	"sync"
	"testing"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gopkg.in/urfave/cli.v1"
)

func TestFlags(t *testing.T) {
	testFlags := flag.NewFlagSet("testflags", flag.PanicOnError)
	testFlags.String("config", "", "where to find the config")
	testFlags.Int64("freq", 100, "how often to poll")
	testFlags.String("log-level", "info", "log level")

	// Ensure we require config
	testFlags.Set("config", "")
	testFlags.Set("freq", "100")
	testFlags.Set("log-level", "info")

	testCtx := cli.NewContext(app, testFlags, nil)
	err := runApp(testCtx)
	require.Error(t, err)
}

func TestLogLevel(t *testing.T) {
	testFlags := flag.NewFlagSet("testflags", flag.PanicOnError)
	testFlags.String("log-level", "", "log level")

	// Check we can set it to warn
	testFlags.Set("log-level", "warn")

	testCtx := cli.NewContext(app, testFlags, nil)
	runApp(testCtx)
	require.Equal(t, log.GetLevel(), log.WarnLevel)

	// Setting it to the wrong thing breaks
	testFlags.Set("log-level", "notalevel")

	testCtx = cli.NewContext(app, testFlags, nil)
	err := runApp(testCtx)
	require.Equal(t, err.Error(), "Could not parse log level")

	// Leaving it unset defaults to info
	testFlags.Set("log-level", "")

	testCtx = cli.NewContext(app, testFlags, nil)
	runApp(testCtx)
	require.Equal(t, log.GetLevel(), log.InfoLevel)
}

// WaitQueue implements Queue
type WaitQueue struct {
	sem  *sync.WaitGroup
	seq  chan int
	wait chan int
}

func (w *WaitQueue) Semaphore() *sync.WaitGroup {
	return w.sem
}

// Poll sends `1` on the sequence channel, then waits on `wait`, then sends `2`
// then downs the semaphore.
func (w *WaitQueue) Poll() {
	defer w.sem.Done()
	w.seq <- 1
	<-w.wait
	w.seq <- 2
}

// waitListen just calls listen and then sends `3 on the sequence channel when
// the call exits
func waitListen(q Queue, freq <-chan time.Time, seq chan int) {
	setContext()
	daemonContext.writePIDFile()
	Listen(q, freq)
	seq <- 3
	daemonContext.removePIDFile()
}

// This test is pretty complicated because I'm basically using it to ensure that
// the loop in Listen will only exit once all of the in flight work is done. The
// basic structure of theis test is to have a mock whose call to Poll() blocks
// until I send it a signal. We then send something on the ticker channel to
// kick off a job (which blocks). Then we send on the signal channel to start
// the graceful exit. Then we tell the queue's Poll() to exit, and then the
// Listen should exit. We use a separate sequence channel to ensure the order
// of everything
func TestSignals(t *testing.T) {
	seq := make(chan int)
	wait := make(chan int)
	freq := make(chan time.Time)

	queue := &WaitQueue{
		sem:  new(sync.WaitGroup),
		seq:  seq,
		wait: wait,
	}

	setContext()
	daemonContext.writePIDFile()

	// begin listening
	go waitListen(queue, freq, seq)

	// send a tick, this should start a call to Poll()
	freq <- time.Now()

	// when that call starts, we should get `1` on the sequence channel
	val := <-seq
	require.Equal(t, val, 1)

	// send a signal, this should start the graceful exit
	signals <- os.Interrupt

	// tell Poll() that it can exit
	wait <- 1

	// first Poll() should exit
	val = <-seq
	require.Equal(t, val, 2)

	// then Listen() should exit
	val = <-seq
	require.Equal(t, val, 3)

	daemonContext.removePIDFile()
}
