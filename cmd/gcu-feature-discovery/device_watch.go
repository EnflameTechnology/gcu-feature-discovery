/*
 * Copyright (C) 2024 Enflame Technologies, Inc.
 */

package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"
)

// efmlFingerprint uses `efsmi -L` output whenever the efsmi binary is on PATH; otherwise it falls back
// to /dev/gcu* and /dev/vgcu* presence counts only (no efsmi in the image/node).
func efmlFingerprint(_ Efml) ([32]byte, error) {
	if _, err := exec.LookPath("efsmi"); err == nil {
		material, _ := countDevicesPassRunSmi()
		sha256Info := sha256.Sum256([]byte(material))
		//log.Printf("efmlFingerprint: %x", sha256Info)
		return sha256Info, nil
	}
	g := countHostGcuDevices()
	v := countHostVgcuDevices()
	countInfo := fmt.Sprintf("%d:%d", g, v)
	sha256Info := sha256.Sum256([]byte(countInfo))
	//log.Printf("efmlFingerprint: %s -> %x", countInfo, sha256Info)
	return sha256Info, nil
}

type efmlFingerprinter func(Efml) ([32]byte, error)

func runDeviceWatchLoop(efml Efml, interval time.Duration, mu *sync.Mutex, stop <-chan struct{}, watchInitFail chan<- error) {
	runDeviceWatchLoopWithFingerprinter(efml, interval, mu, stop, watchInitFail, efmlFingerprint)
}

func runDeviceWatchLoopWithFingerprinter(efml Efml, interval time.Duration, mu *sync.Mutex, stop <-chan struct{}, watchInitFail chan<- error, probe efmlFingerprinter) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var last [32]byte
	first := true

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			fp, err := probe(efml)
			if err != nil {
				log.Printf("device watch: fingerprint failed: %v", err)
				continue
			}
			if first {
				last = fp
				first = false
				continue
			}
			if fp == last {
				continue
			}
			mu.Lock()
			if err := efml.Init(); err != nil {
				log.Printf("device watch: EFML Init after change failed: %v", err)
				if watchInitFail != nil {
					select {
					case watchInitFail <- err:
					default:
					}
				}
			} else {
				last = fp
				log.Printf("device watch: EFML cache refreshed (host GCU/vGCU device count changed)")
			}
			mu.Unlock()
		}
	}
}
