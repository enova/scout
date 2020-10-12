package main

import (
	"errors"
	"io/ioutil"
	"os"
	"strconv"
)

type context struct {
	DaemonName  string
	PidFileName string
	WorkDir     string
}

func setContext() {
	workDir, _ := os.Getwd()

	daemonContext = &context{
		DaemonName:  app.Name,
		PidFileName: app.Name + ".pid",
		WorkDir:     workDir,
	}
}

func (c *context) writePIDFile() (err error) {
	filePath := c.WorkDir + "/" + c.PidFileName

	if _, err := os.Stat(filePath); err == nil {
		return errors.New("PID file exists, scout daemon already running")
	} else if os.IsNotExist(err) {
		pid := strconv.Itoa(os.Getpid())
		return ioutil.WriteFile(filePath, []byte(pid), 0644)
	}

	return nil
}

func (c *context) findPIDFile() (err error) {
	filePath := c.WorkDir + "/" + c.PidFileName
	if _, err := os.Stat(filePath); err == nil {
		return nil
	}

	return errors.New("No PID File found")
}

func (c *context) removePIDFile() (err error) {
	filePath := c.WorkDir + "/" + c.PidFileName
	if _, err = os.Stat(filePath); err == nil {
		return os.Remove(c.PidFileName)
	}
	return err
}
