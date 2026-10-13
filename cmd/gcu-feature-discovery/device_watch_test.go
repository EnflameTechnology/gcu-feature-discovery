/*
 * Copyright (C) 2024 Enflame Technologies, Inc.
 */

package main

import (
	"sync"
	"testing"
	"time"
)

func TestEfmlFingerprint_sameForAnyBackendType(t *testing.T) {
	a, err := efmlFingerprint(&EfmlLib{})
	if err != nil {
		t.Fatalf("efmlFingerprint: %v", err)
	}
	b, err := efmlFingerprint(&EfmlLibText{})
	if err != nil {
		t.Fatalf("efmlFingerprint: %v", err)
	}
	c, err := efmlFingerprint(&mockEfml{})
	if err != nil {
		t.Fatalf("efmlFingerprint: %v", err)
	}
	if a != b || b != c {
		t.Fatalf("fingerprint should depend only on host /dev nodes, got %x %x %x", a, b, c)
	}
}

func TestCountHostDevices_noPanic(t *testing.T) {
	_ = countHostGcuDevices()
	_ = countHostVgcuDevices()
}

// efmlInitCounter wraps mockEfml to count Init calls (device watch refresh path).
type efmlInitCounter struct {
	*mockEfml
	initCalls int
}

func (e *efmlInitCounter) Init() error {
	e.initCalls++
	return e.mockEfml.Init()
}

func TestRunDeviceWatchLoopWithFingerprinter_noInitWhenFingerprintStable(t *testing.T) {
	base := &mockEfml{
		devices: []SmiDevice{
			{SKU: "L600", DriverVer: "1.7.2.25", MemTotalMiB: 65536, DevName: "Enflame L600", FwVersion: "3.0.1"},
		},
	}
	ef := &efmlInitCounter{mockEfml: base}

	probe := func(Efml) ([32]byte, error) { return [32]byte{7, 8, 9}, nil }
	stop := make(chan struct{})
	var mu sync.Mutex

	go func() {
		time.Sleep(45 * time.Millisecond)
		close(stop)
	}()

	runDeviceWatchLoopWithFingerprinter(ef, 12*time.Millisecond, &mu, stop, nil, probe)
	if ef.initCalls != 0 {
		t.Fatalf("expected no Init when fingerprint unchanged, got %d", ef.initCalls)
	}
}

func TestRunDeviceWatchLoopWithFingerprinter_initWhenFingerprintChanges(t *testing.T) {
	base := &mockEfml{
		devices: []SmiDevice{
			{SKU: "L600", DriverVer: "1.7.2.25", MemTotalMiB: 65536, DevName: "Enflame L600", FwVersion: "3.0.1"},
		},
	}
	ef := &efmlInitCounter{mockEfml: base}

	var n int
	probe := func(Efml) ([32]byte, error) {
		n++
		return [32]byte{byte(n)}, nil
	}
	stop := make(chan struct{})
	var mu sync.Mutex

	go func() {
		time.Sleep(50 * time.Millisecond)
		close(stop)
	}()

	runDeviceWatchLoopWithFingerprinter(ef, 10*time.Millisecond, &mu, stop, nil, probe)
	if ef.initCalls < 2 {
		t.Fatalf("expected Init after fingerprint changes, got initCalls=%d (probe calls=%d)", ef.initCalls, n)
	}
}
