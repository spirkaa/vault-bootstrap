package main

import (
	"flag"
	"os"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spirkaa/vault-bootstrap/bootstrap"
)

func main() {
	runningMode := flag.String("mode", "job", "running mode: job or init-container")
	flag.Parse()
	if *runningMode == "job" {
		log.Info("Running in job mode...")
		bootstrap.Run()
	} else if *runningMode == "init-container" {
		log.Info("Running in init-container mode...")
		bootstrap.InitContainer()
	} else {
		panic("Running mode must be 'sidecar' or 'job'")
	}
}

func init() {
	const DefaultLogLevel = "Info"

	logLevel, ok := os.LookupEnv("LOG_LEVEL")
	if !ok {
		logLevel = DefaultLogLevel
	}
	level, err := log.ParseLevel(strings.Title(logLevel))
	if err != nil {
		return
	}

	// Output everything including stderr to stdout
	log.SetOutput(os.Stdout)
	log.SetLevel(level)

	log.Info("LogLevel set to " + level.String())
	log.Info(runtime.Version())
}
