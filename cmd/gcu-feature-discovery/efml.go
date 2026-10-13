/*
 * Copyright (C) 2024 Enflame Technologies, Inc.
 */

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
)

// EfmlInitError : Used to signal an error during initialization vs. other errors
type EfmlInitError struct{ error }

// DeviceHandle is a simple index-based handle
type DeviceHandle uint32

// DeviceInfo holds device info
type DeviceInfo struct {
	Name      string
	Vendor_Id uint32
	Device_Id uint32
	Domain_Id uint32
	Bus_Id    uint32
	Dev_Id    uint32
	Func_Id   uint32
}

// SmiDevice holds all parsed fields for a single device
type SmiDevice struct {
	Index       uint32
	DevName     string
	SKU         string
	DriverVer   string
	FwVersion   string
	VendorID    uint32
	DeviceID    uint32
	Domain      uint32
	Bus         uint32
	Dev         uint32
	Func        uint32
	MemTotalMiB uint64
	HwArchName  string
}

// Efml : Type to represent interactions with EFML
type Efml interface {
	Init() error
	Shutdown() error
	GetDriverVer() (string, error)
	CheckDevCountState() (uint, error)
	GetDeviceCount() (uint32, error)
	GetVDeviceCount() (uint32, error)
	GetDeviceHandle(id uint32) (DeviceHandle, error)

	GetDeviceSKU(h DeviceHandle) (string, error)
	GetDeviceMemoryTotalGBSize(h DeviceHandle) (uint64, error)
	GetDevInfo(h DeviceHandle) (*DeviceInfo, error)
	GetFwVersion(h DeviceHandle) (string, error)
	GetDeviceHwArchName(h DeviceHandle) (string, error)
}

// ===== JSON structures matching efsmi -q --json-format output =====
//
// These structs mirror the JSON schema produced by efsmi >= 1.7.2.25.
// Fields use pointer types (*string) where the JSON value may be null
// (e.g. FW versions on certain hardware). Unknown/new fields added by
// future efsmi versions are silently ignored by encoding/json.

type smiJsonOutput struct {
	Devices []smiJsonDevice `json:"devices"`
}

type smiJsonDevice struct {
	ID         int               `json:"id"`
	DriverInfo smiJsonDriverInfo `json:"Driver Info"`
	DeviceInfo smiJsonDevInfo    `json:"Device Info"`
	FwInfo     smiJsonFwInfo     `json:"Firmware Info"`
	PCIeInfo   smiJsonPCIe       `json:"PCIe Info"`
	DevMemInfo smiJsonMem        `json:"Device Mem Info"`
}

type smiJsonDriverInfo struct {
	Ver string `json:"Ver"`
}

type smiJsonDevInfo struct {
	DevName string `json:"Dev Name"`
	DevUUID string `json:"Dev UUID"`
	DevSN   string `json:"Dev SN"`
	Health  *bool  `json:"Health"` // pointer: may be absent
}

type smiJsonFwInfo struct {
	SSMBootFWVersion *string `json:"SSM Boot FW Version"` // pointer: may be null
	SPFWVersion      *string `json:"SP FW Version"`
	VPUFWVersion     *string `json:"VPU FW Version"`
}

type smiJsonPCIe struct {
	VendorID string `json:"Vendor ID"`
	DeviceID string `json:"Device ID"`
	Domain   string `json:"Domain"`
	Bus      string `json:"Bus"`
	Dev      string `json:"Dev"`
	Func     string `json:"Func"`
}

type smiJsonMem struct {
	TotalSize uint64 `json:"Total Size"`
}

// EfmlLib : Implementation of Efml using smi command with JSON output
type EfmlLib struct {
	devices []SmiDevice
}

// Init : Run smi tool with JSON output and parse the result (refreshes e.devices).
func (e *EfmlLib) Init() error {
	output, err := runSmi()
	if err != nil {
		return fmt.Errorf("failed to run smi tool: %v", err)
	}
	e.devices, err = parseSmiJsonOutput(output)
	if err != nil {
		return fmt.Errorf("failed to parse smi JSON output: %v", err)
	}
	log.Printf("smi: discovered %d device(s)", len(e.devices))
	return nil
}

// Shutdown : No-op for smi-based implementation
func (e *EfmlLib) Shutdown() error {
	return nil
}

func (e *EfmlLib) GetDriverVer() (string, error) {
	if len(e.devices) == 0 {
		return "", fmt.Errorf("no devices found")
	}
	return e.devices[0].DriverVer, nil
}

func (e *EfmlLib) CheckDevCountState() (uint, error) {
	return uint(len(e.devices)), nil
}

// GetDeviceCount : Returns device count from efsmi discovery result
func (e *EfmlLib) GetDeviceCount() (uint32, error) {
	return uint32(len(e.devices)), nil
}

// GetVDeviceCount : efsmi JSON does not include vGCU info;
// fall back to scanning /dev/vgcu* to preserve original behavior.
func (e *EfmlLib) GetVDeviceCount() (uint32, error) {
	count := uint32(0)
	for i := 0; i < GCU_DEV_MAX; i++ {
		devicePath := fmt.Sprintf("/dev/vgcu%d", i)
		if fileIsExist(devicePath) {
			count++
		}
	}
	return count, nil
}

func (e *EfmlLib) GetDeviceHandle(id uint32) (DeviceHandle, error) {
	if int(id) >= len(e.devices) {
		return 0, fmt.Errorf("device index %d out of range (total %d)", id, len(e.devices))
	}
	return DeviceHandle(id), nil
}

func (e *EfmlLib) GetDeviceSKU(h DeviceHandle) (string, error) {
	if int(h) >= len(e.devices) {
		return "", fmt.Errorf("invalid device handle: %d", h)
	}
	return e.devices[h].SKU, nil
}

func (e *EfmlLib) GetDeviceMemoryTotalGBSize(h DeviceHandle) (uint64, error) {
	if int(h) >= len(e.devices) {
		return 0, fmt.Errorf("invalid device handle: %d", h)
	}
	return e.devices[h].MemTotalMiB / 1024, nil
}

func (e *EfmlLib) GetDevInfo(h DeviceHandle) (*DeviceInfo, error) {
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

func (e *EfmlLib) GetFwVersion(h DeviceHandle) (string, error) {
	if int(h) >= len(e.devices) {
		return "", fmt.Errorf("invalid device handle: %d", h)
	}
	return e.devices[h].FwVersion, nil
}

func (e *EfmlLib) GetDeviceHwArchName(h DeviceHandle) (string, error) {
	if int(h) >= len(e.devices) {
		return "", fmt.Errorf("invalid device handle: %d", h)
	}
	family, ok := skuToFamily(normalizeSKU(e.devices[h].SKU))
	if !ok {
		return "UNKNOWN", nil
	}
	return family, nil
}

// ===== smi command execution =====

// runSmi executes smi tool with JSON output.
// Captures both stdout and stderr for diagnostics.
func runSmi() (string, error) {
	cmd := exec.Command("efsmi", "-q", "--json-format")
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return "", fmt.Errorf("efsmi -q --json-format failed: %v, stderr: %s", err, errMsg)
		}
		return "", fmt.Errorf("efsmi -q --json-format failed: %v", err)
	}
	return stdout.String(), nil
}

// ===== JSON parsing =====

// parseSmiJsonOutput parses JSON output from smi -q --json-format
func parseSmiJsonOutput(output string) ([]SmiDevice, error) {
	var result smiJsonOutput
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return nil, fmt.Errorf("JSON unmarshal failed: %v", err)
	}
	if len(result.Devices) == 0 {
		log.Printf("smi: no devices found in JSON output")
		return []SmiDevice{}, nil
	}

	var devices []SmiDevice
	for _, jd := range result.Devices {
		dev := SmiDevice{
			Index:       uint32(jd.ID),
			DriverVer:   jd.DriverInfo.Ver,
			DevName:     jd.DeviceInfo.DevName,
			MemTotalMiB: jd.DevMemInfo.TotalSize,
		}

		// SKU derived from Dev Name
		dev.SKU = extractSKU(dev.DevName)

		// Firmware Info (pointer fields, may be null)
		if jd.FwInfo.SSMBootFWVersion != nil {
			dev.FwVersion = *jd.FwInfo.SSMBootFWVersion
		}

		// PCIe Info (hex string fields)
		dev.VendorID = parseHexStr(jd.PCIeInfo.VendorID)
		dev.DeviceID = parseHexStr(jd.PCIeInfo.DeviceID)
		dev.Domain = parseHexStr(jd.PCIeInfo.Domain)
		dev.Bus = parseHexStr(jd.PCIeInfo.Bus)
		dev.Dev = parseHexStr(jd.PCIeInfo.Dev)
		dev.Func = parseDecStr(jd.PCIeInfo.Func)

		devices = append(devices, dev)
	}

	return devices, nil
}

// parseHexStr parses a hex string (with or without 0x prefix) to uint32.
// Logs a warning on parse failure instead of silently returning 0.
func parseHexStr(s string) uint32 {
	s = strings.TrimPrefix(strings.TrimSpace(s), "0x")
	if s == "" {
		return 0
	}
	val, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		log.Printf("Warning: failed to parse hex string %q: %v", s, err)
		return 0
	}
	return uint32(val)
}

// parseDecStr parses a decimal string to uint32.
// Logs a warning on parse failure instead of silently returning 0.
func parseDecStr(s string) uint32 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	val, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		log.Printf("Warning: failed to parse decimal string %q: %v", s, err)
		return 0
	}
	return uint32(val)
}

// ===== Helper functions =====

func extractSKU(devName string) string {
	devName = strings.TrimSpace(devName)
	if devName == "" {
		return "UNKNOWN"
	}
	parts := strings.Fields(devName)
	sku := devName
	if len(parts) >= 2 {
		sku = parts[len(parts)-1]
	}
	// Sanitize parentheses for K8s label compatibility:
	// "MODEL200(A4)" -> "MODEL200-A4"
	sku = strings.ReplaceAll(sku, "(", "-")
	sku = strings.ReplaceAll(sku, ")", "")
	return sku
}

// normalizeSKU tries to match a SKU against the configured family mapping
// by progressively stripping trailing "-suffix" or "_suffix" segments.
// e.g. "MODEL200-A4" -> "MODEL200", "MODEL300_OGX" -> "MODEL300".
func normalizeSKU(sku string) string {
	if _, ok := skuToFamily(sku); ok {
		return sku
	}
	for _, sep := range []string{"-", "_"} {
		if idx := strings.LastIndex(sku, sep); idx > 0 {
			base := sku[:idx]
			if _, ok := skuToFamily(base); ok {
				return base
			}
		}
	}
	return sku
}
