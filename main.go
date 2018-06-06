package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/urfave/cli.v1"
)

var (
	app     *cli.App
	signals chan os.Signal
)

func init() {
	app = cli.NewApp()

	app.Name = "scout"
	app.Usage = `SQS Listener
Poll SQS queues specified in a config and enqueue Sidekiq jobs with the queue items.
It gracefully stops when sent SIGTERM.`

	app.Version = "1.4"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config, c",
			Usage: "Load config from `FILE`, required",
		},
		cli.Int64Flag{
			Name:  "freq, f",
			Value: 100,
			Usage: "Poll SQS every `N` milliseconds",
		},
		cli.StringFlag{
			Name:  "log-level, l",
			Usage: "Sets log level. Accepts one of: debug, info, warn, error",
		},
		cli.BoolFlag{
			Name:  "json, j",
			Usage: "Log in json format",
		},
	}

	app.Action = runApp

	signals = make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
}

func main() {
	app.Run(os.Args)
}

func runApp(ctx *cli.Context) error {
	configFile := ctx.String("config")
	frequency := ctx.Int64("freq")

	if ctx.Bool("json") {
		log.SetFormatter(&log.JSONFormatter{})
	}

	logLevel := ctx.String("log-level")
	if logLevel == "" {
		logLevel = "info"
	}

	level, err := log.ParseLevel(logLevel)
	if err != nil {
		return cli.NewExitError("Could not parse log level", 1)
	}

	log.SetLevel(level)

	if configFile == "" {
		return cli.NewExitError("Missing required flag --config. Run `scout --help` for more information", 1)
	}

	log.Infof("Reading config from %s", configFile)
	log.Infof("Polling every %d milliseconds", frequency)

	config, err := ReadConfig(configFile)
	if err != nil {
		return cli.NewExitError("Failed to parse config file", 1)
	}

	queue, err := NewQueue(config)
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("Initialization error: %s", err.Error()), 1)
	}

	log.Info("Now listening on queue: ", config.Queue.Name)
	for topic, worker := range config.Queue.Topics {
		log.Infof("%s -> %s", topic, worker)
	}

	Listen(queue, time.Tick(time.Duration(frequency)*time.Millisecond))
	return nil
}

// Listen does the work. It only returns if we get a signal
func Listen(queue Queue, freq <-chan time.Time) {
	for {
		select {
		case <-signals:
			log.Info("Got TERM")
			queue.Semaphore().Wait()
			return
		case tick := <-freq:
			log.Debug("Polling at: ", tick)
			queue.Semaphore().Add(1)
			go queue.Poll()
		}
	}
}
