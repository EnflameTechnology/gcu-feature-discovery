/*
 * Copyright (C) 2024 Enflame Technologies, Inc.
 */

package main

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
)

// EfmlLibText : Implementation of Efml using efsmi -q text output parsing
type EfmlLibText struct {
	devices []SmiDevice
}

// Init : Run efsmi -q and parse text output (refreshes e.devices).
func (e *EfmlLibText) Init() error {
	output, err := runSmiText()
	if err != nil {
		return fmt.Errorf("failed to run efsmi -q: %v", err)
	}
	e.devices, err = parseSmiTextOutput(output)
	if err != nil {
		return fmt.Errorf("failed to parse efsmi text output: %v", err)
	}
	log.Printf("efsmi(text): discovered %d device(s)", len(e.devices))
	return nil
}

func (e *EfmlLibText) Shutdown() error { return nil }

func (e *EfmlLibText) GetDriverVer() (string, error) {
	if len(e.devices) == 0 {
		return "", fmt.Errorf("no devices found")
	}
	return e.devices[0].DriverVer, nil
}

func (e *EfmlLibText) CheckDevCountState() (uint, error) {
	return uint(len(e.devices)), nil
}

func (e *EfmlLibText) GetDeviceCount() (uint32, error) {
	return uint32(len(e.devices)), nil
}

func (e *EfmlLibText) GetVDeviceCount() (uint32, error) {
	count := uint32(0)
	for i := 0; i < GCU_DEV_MAX; i++ {
		devicePath := fmt.Sprintf("/dev/vgcu%d", i)
		if fileIsExist(devicePath) {
			count++
		}
	}
	return count, nil
}

func (e *EfmlLibText) GetDeviceHandle(id uint32) (DeviceHandle, error) {
	if int(id) >= len(e.devices) {
		return 0, fmt.Errorf("device index %d out of range (total %d)", id, len(e.devices))
	}
	return DeviceHandle(id), nil
}

func (e *EfmlLibText) GetDeviceSKU(h DeviceHandle) (string, error) {
	if int(h) >= len(e.devices) {
		return "", fmt.Errorf("invalid device handle: %d", h)
	}
	return e.devices[h].SKU, nil
}

func (e *EfmlLibText) GetDeviceMemoryTotalGBSize(h DeviceHandle) (uint64, error) {
	if int(h) >= len(e.devices) {
		return 0, fmt.Errorf("invalid device handle: %d", h)
	}
	return e.devices[h].MemTotalMiB / 1024, nil
}

func (e *EfmlLibText) GetDevInfo(h DeviceHandle) (*DeviceInfo, error) {
	if int(h) >= len(e.devices) {
		return nil, fmt.Errorf("invalid device handle: %d", h)
	}
	d := &e.devices[h]
	return &DeviceInfo{
		Name:      d.DevName,
		Vendor_Id: d.VendorID,
		Device_Id: d.DeviceID,
		Domain_Id: d.Domain,
		Bus_Id:    d.Bus,
		Dev_Id:    d.Dev,
		Func_Id:   d.Func,
	}, nil
}

func (e *EfmlLibText) GetFwVersion(h DeviceHandle) (string, error) {
	if int(h) >= len(e.devices) {
		return "", fmt.Errorf("invalid device handle: %d", h)
	}
	return e.devices[h].FwVersion, nil
}

func (e *EfmlLibText) GetDeviceHwArchName(h DeviceHandle) (string, error) {
	if int(h) >= len(e.devices) {
		return "", fmt.Errorf("invalid device handle: %d", h)
	}
	family, ok := skuToFamily(normalizeSKU(e.devices[h].SKU))
	if !ok {
		return "UNKNOWN", nil
	}
	return family, nil
}

// ===== efsmi -q text execution and parsing =====

func runSmiText() (string, error) {
	cmd := exec.Command("efsmi", "-q")
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return "", fmt.Errorf("efsmi -q failed: %v, stderr: %s", err, errMsg)
		}
		return "", fmt.Errorf("efsmi -q failed: %v", err)
	}
	return stdout.String(), nil
}

// parseSmiTextOutput parses the text output of efsmi -q into SmiDevice structs.
// The format uses "DEV ID N" as device delimiter, and "Key : Value" lines.
func parseSmiTextOutput(output string) ([]SmiDevice, error) {
	lines := strings.Split(output, "\n")
	var devices []SmiDevice
	var current *SmiDevice

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// New device block
		if strings.HasPrefix(trimmed, "DEV ID ") {
			if current != nil {
				devices = append(devices, *current)
			}
			idx := parseTextUint32(strings.TrimPrefix(trimmed, "DEV ID "))
			current = &SmiDevice{Index: idx}
			continue
		}

		if current == nil {
			continue
		}

		// Parse "Key : Value" lines
		if !strings.Contains(trimmed, ":") {
			continue
		}
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "Ver":
			current.DriverVer = val
		case "Dev Name":
			current.DevName = val
			current.SKU = extractSKU(val)
		case "SSM Boot FW Version":
			current.FwVersion = val
		case "Vendor ID":
			current.VendorID = parseHexStr(val)
		case "Device ID":
			current.DeviceID = parseHexStr(val)
		case "Domain":
			current.Domain = parseHexStr(val)
		case "Bus":
			current.Bus = parseHexStr(val)
		case "Dev":
			current.Dev = parseHexStr(val)
		case "Func":
			current.Func = parseDecStr(val)
		case "Total Size", "Mem Size":
			// Device Mem Info: newer efsmi uses "Total Size", older drivers use "Mem Size".
			// Only the first match per device (BAR1 section also has "Total Size").
			if current.MemTotalMiB == 0 {
				if n, ok := parseSmiTextMemSizeMiB(val); ok {
					current.MemTotalMiB = n
				}
			}
		}
	}

	// Don't forget the last device
	if current != nil {
		devices = append(devices, *current)
	}

	return devices, nil
}

// parseSmiTextMemSizeMiB parses efsmi text values such as "49152 MiB".
func parseSmiTextMemSizeMiB(val string) (uint64, bool) {
	if !strings.Contains(val, "MiB") {
		return 0, false
	}
	numStr := strings.TrimSpace(strings.Replace(val, "MiB", "", 1))
	n, err := strconv.ParseUint(numStr, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

func parseTextUint32(s string) uint32 {
	s = strings.TrimSpace(s)
	val, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0
	}
	return uint32(val)
}
