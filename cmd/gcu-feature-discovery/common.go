/*
 * Copyright (C) 2024 Enflame Technologies, Inc.
 */
package main

import (
	"fmt"
	"enflame/gcu-feature-discovery/internal/eflib"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func getExitSignal(exitChan chan bool) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		s := <-sigChan
		log.Printf("Received signal \"%v\", shutting down.", s)
		exitChan <- true
	}()
}

func getDeployedTime(labels map[string]string) {
	labels[DeployedTimestamp] = time.Now().Format("2006-01-02-15-04-05")
}

func fileIsExist(file string) bool {
	_, err := os.Stat(file)
	return err == nil || os.IsExist(err)
}

// countDevicesPassRunSmi runs `efsmi -L` and returns a string suitable for hashing as a device fingerprint.
// On non-zero exit it still returns non-empty material derived from stdout/stderr so callers that only
// have efsmi on PATH do not fall back to /dev node counts when -L fails.
func countDevicesPassRunSmi() (string, error) {
	cmd := exec.Command("efsmi", "-L")
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := strings.TrimSpace(stdout.String())
	se := strings.TrimSpace(stderr.String())
	if err != nil {
		material := out
		switch {
		case material != "" && se != "":
			material += "\n---\n" + se
		case material == "" && se != "":
			material = se
		case material == "":
			material = fmt.Sprintf("(efsmi-L-failed:%v)", err)
		}
		errMsg := se
		if errMsg != "" {
			return material, fmt.Errorf("efsmi -L failed: %v, stderr: %s", err, errMsg)
		}
		return material, fmt.Errorf("efsmi -L failed: %v", err)
	}
	if out == "" {
		return "(efsmi-L-empty)", nil
	}
	return out, nil
}

// countHostGcuDevices returns the number of present /dev/gcu* nodes (same rule as EfmlLibLegacy.GetDeviceCount).
func countHostGcuDevices() uint32 {
	count := 0
	for i := 0; i < GCU_DEV_MAX; i++ {
		devicePath := fmt.Sprintf("%s%d", GCU_DEV_NAME, i)
		if fileIsExist(devicePath) {
			count++
		}
	}
	return uint32(count)
}

// countHostVgcuDevices returns the number of  eflib.GetVDeviceCount.
func countHostVgcuDevices() uint32 {
	vcount, _ := eflib.GetVDeviceCount()
	return vcount
}
