/*
 * Copyright (c) 2019, NVIDIA CORPORATION. All rights reserved.
 * Modifications Copyright (C) 2024 Enflame Technologies, Inc.
 */
package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/docopt/docopt-go"
)

// Conf : Type to represent options
type Conf struct {
	Oneshot             bool
	FailOnInitError     bool
	MigStrategy         string
	OutputFilePath      string
	SleepInterval       time.Duration
	DeviceWatchInterval time.Duration
	NoTimestamp         bool
}

func (conf *Conf) getConfFromArgv(argv []string) {
	usage := fmt.Sprintf(`%[1]s:
Usage:
  %[1]s [--fail-on-init-error=<bool>] [--mig-strategy=<strategy>] [--device-watch-interval=<seconds>] [--oneshot | --sleep-interval=<seconds>] [--no-timestamp] [--output-file=<file> | -o <file>]
  %[1]s -h | --help
  %[1]s --version

Options:
  -h --help                       Show this help message and exit
  --version                       Display version and exit
  --oneshot                       Label once and exit
  --no-timestamp                  Do not add the timestamp to the labels
  --fail-on-init-error=<bool>     Fail if there is an error during initialization of any label sources [Default: true]
  --sleep-interval=<seconds>      Time to sleep between labeling [Default: 60s]
  --device-watch-interval=<seconds>  Poll this often: host /dev/gcu* node count + vGPU count (same as EFML GetVDeviceCount); on change run EFML Init to refresh cache [Default: 60s]
                                  Use 0 to disable (run EFML Init every label cycle; higher load)
  --mig-strategy=<strategy>       Strategy to use for MIG-related labels [Default: none]
  -o <file> --output-file=<file>  Path to output file
                                  [Default: /etc/kubernetes/node-feature-discovery/features.d/gcu-feature-discovery]

Arguments:
  <strategy>: none | single | mixed`,
		ComponentName)

	opts, err := docopt.ParseArgs(usage, argv[1:], ComponentName+" "+Version)
	if err != nil {
		log.Fatal("Error while parsing command line options: ", err)
	}

	conf.Oneshot, err = opts.Bool("--oneshot")
	if err != nil {
		log.Fatal("Error while parsing command line options: ", err)
	}
	failOnInitErrorTmp, err := opts.String("--fail-on-init-error")
	if err != nil {
		log.Fatal("Error while parsing command line options: ", err)
	}
	if failOnInitErrorTmp != "true" && failOnInitErrorTmp != "false" {
		log.Fatal("Error while parsing command line options: value of --fail-on-init-error must be 'true' or 'false'")
	}
	conf.FailOnInitError = strings.EqualFold(failOnInitErrorTmp, "true")
	conf.MigStrategy, err = opts.String("--mig-strategy")
	if err != nil {
		log.Fatal("Error while parsing command line options: ", err)
	}
	sleepIntervalString, err := opts.String("--sleep-interval")
	if err != nil {
		log.Fatal("Error while parsing command line options: ", err)
	}
	conf.OutputFilePath, err = opts.String("--output-file")
	if err != nil {
		log.Fatal("Error while parsing command line options: ", err)
	}

	conf.SleepInterval, err = time.ParseDuration(sleepIntervalString)
	if err != nil {
		log.Fatal("Invalid value for --sleep-interval option: ", err)
	}

	deviceWatchString, err := opts.String("--device-watch-interval")
	if err != nil {
		log.Fatal("Error while parsing command line options: ", err)
	}
	conf.DeviceWatchInterval, err = time.ParseDuration(deviceWatchString)
	if err != nil {
		log.Fatal("Invalid value for --device-watch-interval option: ", err)
	}

	conf.NoTimestamp, err = opts.Bool("--no-timestamp")
	if err != nil {
		log.Fatal("Error while parsing command line options: ", err)
	}
}

func (conf *Conf) getConfFromEnv() {
	val, ok := os.LookupEnv("GFD_ONESHOT")
	if ok && strings.EqualFold(val, "true") {
		conf.Oneshot = true
	}
	val, ok = os.LookupEnv("GFD_NO_TIMESTAMP")
	if ok && strings.EqualFold(val, "true") {
		conf.NoTimestamp = true
	}
	val, ok = os.LookupEnv("GFD_FAIL_ON_INIT_ERROR")
	if ok && strings.EqualFold(val, "false") {
		conf.FailOnInitError = false
	}
	migStrategyTmp, ok := os.LookupEnv("GFD_MIG_STRATEGY")
	if ok {
		conf.MigStrategy = migStrategyTmp
	}
	sleepIntervalString, ok := os.LookupEnv("GFD_SLEEP_INTERVAL")
	if ok {
		var err error
		conf.SleepInterval, err = time.ParseDuration(sleepIntervalString)
		if err != nil {
			log.Fatal("Invalid value from env for sleep-interval option: ", err)
		}
	}
	outputFilePathTmp, ok := os.LookupEnv("GFD_OUTPUT_FILE")
	if ok {
		conf.OutputFilePath = outputFilePathTmp
	}
	deviceWatchEnv, ok := os.LookupEnv("GFD_DEVICE_WATCH_INTERVAL")
	if ok {
		var err error
		conf.DeviceWatchInterval, err = time.ParseDuration(deviceWatchEnv)
		if err != nil {
			log.Fatal("Invalid value from env for device-watch-interval option: ", err)
		}
	}
}
