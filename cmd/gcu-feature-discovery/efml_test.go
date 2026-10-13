/*
 * Copyright (C) 2024 Enflame Technologies, Inc.
 */
package main

import (
	"testing"
)

// ===== Sample JSON output for testing =====

const sampleJsonOutput = `{
	"devices": [{
		"id": 0,
		"Driver Info": {"Ver": "1.7.2.25"},
		"Device Info": {"Dev Name": "Enflame S60G", "Health": true},
		"Firmware Info": {"SSM Boot FW Version": "2.1.0"},
		"PCIe Info": {
			"Vendor ID": "1ed8",
			"Device ID": "0602",
			"Domain": "0000",
			"Bus": "3b",
			"Dev": "00",
			"Func": "0"
		},
		"Device Mem Info": {"Total Size": 24576}
	}, {
		"id": 1,
		"Driver Info": {"Ver": "1.7.2.25"},
		"Device Info": {"Dev Name": "Enflame L600", "Health": true},
		"Firmware Info": {"SSM Boot FW Version": "3.0.1"},
		"PCIe Info": {
			"Vendor ID": "1ed8",
			"Device ID": "0702",
			"Domain": "0000",
			"Bus": "86",
			"Dev": "00",
			"Func": "0"
		},
		"Device Mem Info": {"Total Size": 65536}
	}]
}`

// ===== Tests for parseSmiJsonOutput =====

func TestParseSmiJsonOutput_Basic(t *testing.T) {
	devices, err := parseSmiJsonOutput(sampleJsonOutput)
	if err != nil {
		t.Fatalf("parseSmiJsonOutput failed: %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}
}

func TestParseSmiJsonOutput_DeviceFields(t *testing.T) {
	devices, err := parseSmiJsonOutput(sampleJsonOutput)
	if err != nil {
		t.Fatalf("parseSmiJsonOutput failed: %v", err)
	}

	dev0 := devices[0]
	if dev0.Index != 0 {
		t.Errorf("dev0.Index: expected 0, got %d", dev0.Index)
	}
	if dev0.DevName != "Enflame S60G" {
		t.Errorf("dev0.DevName: expected 'Enflame S60G', got '%s'", dev0.DevName)
	}
	if dev0.SKU != "S60G" {
		t.Errorf("dev0.SKU: expected 'S60G', got '%s'", dev0.SKU)
	}
	if dev0.DriverVer != "1.7.2.25" {
		t.Errorf("dev0.DriverVer: expected '1.7.2.25', got '%s'", dev0.DriverVer)
	}
	if dev0.FwVersion != "2.1.0" {
		t.Errorf("dev0.FwVersion: expected '2.1.0', got '%s'", dev0.FwVersion)
	}
	if dev0.VendorID != 0x1ed8 {
		t.Errorf("dev0.VendorID: expected 0x1ed8, got 0x%x", dev0.VendorID)
	}
	if dev0.DeviceID != 0x0602 {
		t.Errorf("dev0.DeviceID: expected 0x0602, got 0x%x", dev0.DeviceID)
	}
	if dev0.Bus != 0x3b {
		t.Errorf("dev0.Bus: expected 0x3b, got 0x%x", dev0.Bus)
	}
	if dev0.MemTotalMiB != 24576 {
		t.Errorf("dev0.MemTotalMiB: expected 24576, got %d", dev0.MemTotalMiB)
	}

	dev1 := devices[1]
	if dev1.Index != 1 {
		t.Errorf("dev1.Index: expected 1, got %d", dev1.Index)
	}
	if dev1.SKU != "L600" {
		t.Errorf("dev1.SKU: expected 'L600', got '%s'", dev1.SKU)
	}
	if dev1.MemTotalMiB != 65536 {
		t.Errorf("dev1.MemTotalMiB: expected 65536, got %d", dev1.MemTotalMiB)
	}
}

func TestParseSmiJsonOutput_Empty(t *testing.T) {
	_, err := parseSmiJsonOutput("")
	if err == nil {
		t.Error("expected error for empty output, got nil")
	}
}

func TestParseSmiJsonOutput_NoDevices(t *testing.T) {
	devices, err := parseSmiJsonOutput(`{"devices": []}`)
	if err != nil {
		t.Errorf("empty devices should not error, got: %v", err)
	}
	if len(devices) != 0 {
		t.Errorf("expected 0 devices, got %d", len(devices))
	}
}

func TestParseSmiJsonOutput_InvalidJson(t *testing.T) {
	_, err := parseSmiJsonOutput(`not json at all`)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestParseSmiJsonOutput_NullFwVersion(t *testing.T) {
	jsonStr := `{
		"devices": [{
			"id": 0,
			"Driver Info": {"Ver": "1.0"},
			"Device Info": {"Dev Name": "Enflame T20"},
			"Firmware Info": {"SSM Boot FW Version": null, "AP FW Version": null},
			"PCIe Info": {"Vendor ID": "1e36", "Device ID": "c041", "Domain": "0", "Bus": "16", "Dev": "00", "Func": "0"},
			"Device Mem Info": {"Total Size": 32768}
		}]
	}`
	devices, err := parseSmiJsonOutput(jsonStr)
	if err != nil {
		t.Fatalf("parseSmiJsonOutput failed: %v", err)
	}
	if devices[0].FwVersion != "" {
		t.Errorf("expected empty FwVersion for null, got '%s'", devices[0].FwVersion)
	}
}

func TestParseSmiJsonOutput_HexParsing(t *testing.T) {
	jsonStr := `{
		"devices": [{
			"id": 0,
			"Driver Info": {"Ver": "1.0"},
			"Device Info": {"Dev Name": "Enflame L300"},
			"Firmware Info": {},
			"PCIe Info": {"Vendor ID": "1e36", "Device ID": "c041", "Domain": "0001", "Bus": "af", "Dev": "01", "Func": "2"},
			"Device Mem Info": {"Total Size": 147456}
		}]
	}`
	devices, err := parseSmiJsonOutput(jsonStr)
	if err != nil {
		t.Fatalf("parseSmiJsonOutput failed: %v", err)
	}
	dev := devices[0]
	if dev.VendorID != 0x1e36 {
		t.Errorf("VendorID: expected 0x1e36, got 0x%x", dev.VendorID)
	}
	if dev.Domain != 0x0001 {
		t.Errorf("Domain: expected 0x0001, got 0x%x", dev.Domain)
	}
	if dev.Bus != 0xaf {
		t.Errorf("Bus: expected 0xaf, got 0x%x", dev.Bus)
	}
	if dev.Dev != 0x01 {
		t.Errorf("Dev: expected 0x01, got 0x%x", dev.Dev)
	}
	if dev.Func != 2 {
		t.Errorf("Func: expected 2, got %d", dev.Func)
	}
}

// ===== Tests for helper functions =====

func TestExtractSKU(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Enflame S60G", "S60G"},
		{"Enflame L600", "L600"},
		{"Enflame GCU300", "GCU300"},
		{"S60G", "S60G"},
		{"MODEL200(A4)", "MODEL200-A4"},
		{"", "UNKNOWN"},
		{"  ", "UNKNOWN"},
	}
	for _, tt := range tests {
		result := extractSKU(tt.input)
		if result != tt.expected {
			t.Errorf("extractSKU(%q): expected %q, got %q", tt.input, tt.expected, result)
		}
	}
}

func TestParseHexStr(t *testing.T) {
	tests := []struct {
		input    string
		expected uint32
	}{
		{"1ed8", 0x1ed8},
		{"0x1ed8", 0x1ed8},
		{"3b", 0x3b},
		{"00", 0},
		{"", 0},
		{"0001", 1},
	}
	for _, tt := range tests {
		result := parseHexStr(tt.input)
		if result != tt.expected {
			t.Errorf("parseHexStr(%q): expected 0x%x, got 0x%x", tt.input, tt.expected, result)
		}
	}
}

func TestParseDecStr(t *testing.T) {
	tests := []struct {
		input    string
		expected uint32
	}{
		{"0", 0},
		{"2", 2},
		{"123", 123},
		{"", 0},
	}
	for _, tt := range tests {
		result := parseDecStr(tt.input)
		if result != tt.expected {
			t.Errorf("parseDecStr(%q): expected %d, got %d", tt.input, tt.expected, result)
		}
	}
}

// ===== Tests for normalizeSKU =====

func TestNormalizeSKU(t *testing.T) {
	old := currentSKUConfig
	currentSKUConfig = &skuFamilyConfig{SkuFamily: map[string]string{
		"MODEL200": "FAMILY_A",
		"L600":     "FAMILY_B",
		"S60G":     "FAMILY_A",
	}}
	defer func() { currentSKUConfig = old }()

	tests := []struct {
		input    string
		expected string
	}{
		{"MODEL200-A4", "MODEL200"},
		{"MODEL200", "MODEL200"},
		{"L600", "L600"},
		{"S60G", "S60G"},
		{"", ""},
	}
	for _, tt := range tests {
		result := normalizeSKU(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeSKU(%q): expected %q, got %q", tt.input, tt.expected, result)
		}
	}
}

// ===== Tests for skuToFamily mapping =====

func TestSkuToFamily(t *testing.T) {
	old := currentSKUConfig
	currentSKUConfig = &skuFamilyConfig{SkuFamily: map[string]string{
		"S60G": "FAMILY_A",
		"L600": "FAMILY_B",
		"T20":  "FAMILY_C",
	}}
	defer func() { currentSKUConfig = old }()

	tests := []struct {
		sku      string
		expected string
	}{
		{"S60G", "FAMILY_A"},
		{"L600", "FAMILY_B"},
		{"T20", "FAMILY_C"},
	}
	for _, tt := range tests {
		family, ok := skuToFamily(tt.sku)
		if !ok {
			t.Errorf("skuToFamily(%q): not found", tt.sku)
			continue
		}
		if family != tt.expected {
			t.Errorf("skuToFamily(%q): expected %q, got %q", tt.sku, tt.expected, family)
		}
	}
}

func TestSkuToFamily_Unknown(t *testing.T) {
	if _, ok := skuToFamily("NONEXISTENT"); ok {
		t.Error("skuToFamily(NONEXISTENT) should not exist")
	}
}

// ===== Tests for EfmlLib via mock =====

type mockEfml struct {
	devices []SmiDevice
}

func (m *mockEfml) Init() error                                     { return nil }
func (m *mockEfml) Shutdown() error                                 { return nil }
func (m *mockEfml) GetDriverVer() (string, error)                   { return m.devices[0].DriverVer, nil }
func (m *mockEfml) CheckDevCountState() (uint, error)               { return uint(len(m.devices)), nil }
func (m *mockEfml) GetDeviceCount() (uint32, error)                 { return uint32(len(m.devices)), nil }
func (m *mockEfml) GetVDeviceCount() (uint32, error)                { return 0, nil }
func (m *mockEfml) GetDeviceHandle(id uint32) (DeviceHandle, error) { return DeviceHandle(id), nil }
func (m *mockEfml) GetDeviceSKU(h DeviceHandle) (string, error)     { return m.devices[h].SKU, nil }
func (m *mockEfml) GetDeviceMemoryTotalGBSize(h DeviceHandle) (uint64, error) {
	return m.devices[h].MemTotalMiB / 1024, nil
}
func (m *mockEfml) GetDevInfo(h DeviceHandle) (*DeviceInfo, error) {
	d := m.devices[h]
	return &DeviceInfo{Name: d.DevName, Vendor_Id: d.VendorID, Device_Id: d.DeviceID}, nil
}
func (m *mockEfml) GetFwVersion(h DeviceHandle) (string, error) { return m.devices[h].FwVersion, nil }
func (m *mockEfml) GetDeviceHwArchName(h DeviceHandle) (string, error) {
	family, ok := skuToFamily(m.devices[h].SKU)
	if !ok {
		return "UNKNOWN", nil
	}
	return family, nil
}

func TestMigStrategyNone_GenerateLabels(t *testing.T) {
	old := currentSKUConfig
	currentSKUConfig = &skuFamilyConfig{SkuFamily: map[string]string{"L600": "FAMILY_B"}}
	defer func() { currentSKUConfig = old }()

	mock := &mockEfml{
		devices: []SmiDevice{
			{SKU: "L600", DriverVer: "1.7.2.25", MemTotalMiB: 65536, DevName: "Enflame L600", FwVersion: "3.0.1"},
		},
	}
	strategy, err := NewMigStrategy(MigStrategyNone, mock, false)
	if err != nil {
		t.Fatalf("NewMigStrategy failed: %v", err)
	}
	labels, err := strategy.GenerateLabels()
	if err != nil {
		t.Fatalf("GenerateLabels failed: %v", err)
	}
	if labels[DeviceModel] != "L600" {
		t.Errorf("DeviceModel: expected 'L600', got '%s'", labels[DeviceModel])
	}
	if labels[DeviceFamily] != "FAMILY_B" {
		t.Errorf("DeviceFamily: expected 'FAMILY_B', got '%s'", labels[DeviceFamily])
	}
	if labels[DeviceDriverVer] != "1.7.2.25" {
		t.Errorf("DeviceDriverVer: expected '1.7.2.25', got '%s'", labels[DeviceDriverVer])
	}
	if labels[DeviceMemorySize] != "65536" {
		t.Errorf("DeviceMemorySize: expected '65536', got '%s'", labels[DeviceMemorySize])
	}
}

func TestMigStrategyNone_VGCULabels(t *testing.T) {
	mock := &mockEfml{
		devices: []SmiDevice{
			{SKU: "T20", DriverVer: "1.0", MemTotalMiB: 32768, DevName: "Enflame T20"},
		},
	}
	strategy, err := NewMigStrategy(MigStrategyNone, mock, true)
	if err != nil {
		t.Fatalf("NewMigStrategy failed: %v", err)
	}
	labels, err := strategy.GenerateLabels()
	if err != nil {
		t.Fatalf("GenerateLabels failed: %v", err)
	}
	if labels[VDeviceModel] != "T20" {
		t.Errorf("VDeviceModel: expected 'T20', got '%s'", labels[VDeviceModel])
	}
	if labels[VDeviceMemorySize] != "8192" {
		t.Errorf("VDeviceMemorySize: expected '8192', got '%s'", labels[VDeviceMemorySize])
	}
}

func TestMigStrategySingle_GenerateLabels(t *testing.T) {
	mock := &mockEfml{
		devices: []SmiDevice{
			{SKU: "S60G", DriverVer: "1.7.2.25", MemTotalMiB: 24576, DevName: "Enflame S60G", FwVersion: "2.1.0", VendorID: 0x1ed8},
			{SKU: "S60G", DriverVer: "1.7.2.25", MemTotalMiB: 24576, DevName: "Enflame S60G", FwVersion: "2.1.0", VendorID: 0x1ed8},
		},
	}
	strategy, err := NewMigStrategy(MigStrategySingle, mock, false)
	if err != nil {
		t.Fatalf("NewMigStrategy failed: %v", err)
	}
	labels, err := strategy.GenerateLabels()
	if err != nil {
		t.Fatalf("GenerateLabels failed: %v", err)
	}
	if labels["enflame.com/mig.strategy"] != "single" {
		t.Errorf("mig.strategy: expected 'single', got '%s'", labels["enflame.com/mig.strategy"])
	}
	if labels["enflame.com/gcu.count"] != "2" {
		t.Errorf("gcu.count: expected '2', got '%s'", labels["enflame.com/gcu.count"])
	}
}

// ===== Tests for fallback behavior =====

func TestEfmlLib_InitIdempotent(t *testing.T) {
	// Manually populate devices to simulate a successful Init
	lib := &EfmlLib{
		devices: []SmiDevice{
			{Index: 0, SKU: "L600"},
		},
	}
	// Second Init should be a no-op (not try to run efsmi)
	err := lib.Init()
	if err != nil {
		t.Errorf("Init on already-initialized EfmlLib should succeed, got: %v", err)
	}
	if len(lib.devices) != 1 {
		t.Errorf("devices should remain unchanged, got %d", len(lib.devices))
	}
}

func TestEfmlLib_InitFailsOnBadJson(t *testing.T) {
	// parseSmiJsonOutput should fail on invalid JSON
	_, err := parseSmiJsonOutput("not valid json")
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}

	// empty devices should succeed with empty list (handled by upper layer)
	devices, err := parseSmiJsonOutput(`{"devices": []}`)
	if err != nil {
		t.Errorf("empty devices should not error, got: %v", err)
	}
	if len(devices) != 0 {
		t.Errorf("expected 0 devices, got %d", len(devices))
	}
}

func TestFallbackSelection(t *testing.T) {
	// Simulate: efsmi backend Init fails -> should use legacy
	// We test this by verifying the Efml interface contract:
	// both EfmlLib and EfmlLibLegacy implement Efml

	var _ Efml = &EfmlLib{}
	var _ Efml = &EfmlLibLegacy{}

	// EfmlLib with no devices and no efsmi available will fail Init
	smiBackend := &EfmlLib{}
	err := smiBackend.Init() // Will fail because efsmi is not available in test env
	if err == nil {
		// If efsmi happens to be available (e.g., on a GCU node), skip this test
		t.Skip("efsmi is available in test environment, skipping fallback test")
	}
	// Verify error is returned (fallback should be triggered by caller)
	t.Logf("EfmlLib.Init() correctly failed: %v", err)
}

// ===== Tests for EfmlLibText (text parsing) =====

const sampleTextOutput = `-------------------------------------------------------------------------------
--------------------- Enflame System Management Interface ---------------------
--------- Enflame Tech, All Rights Reserved. 2024-2026 Copyright (C) ----------
-------------------------------------------------------------------------------
                                                                               
DEV ID 0
    Driver Info
        Ver                     : 1.7.2.3
    Device Info
        Dev Name                : Enflame S60G
        Dev UUID                : T6R231180202
        Dev SN                  : A018K30530050
        Health                  : True
    Firmware Info
        SSM Boot FW Version     : 31.5.3
    PCIe Info
        Vendor ID               : 1e36
        Device ID               : c033
        Domain                  : 0000
        Bus                     : 01
        Dev                     : 00
        Func                    : 0
    Device Mem Info
        Total Size              : 49152 MiB
        Reserved Size           : 1129 MiB
        Used Size               : 0 MiB
        Free Size               : 48022 MiB
    BAR1 Mem Info
        Total Size              : 16 MiB
DEV ID 1
    Driver Info
        Ver                     : 1.7.2.3
    Device Info
        Dev Name                : Enflame L600
        Health                  : True
    Firmware Info
        SSM Boot FW Version     : 3.0.1
    PCIe Info
        Vendor ID               : 1ed8
        Device ID               : 0702
        Domain                  : 0000
        Bus                     : 86
        Dev                     : 00
        Func                    : 0
    Device Mem Info
        Total Size              : 65536 MiB
        Reserved Size           : 0 MiB
`

func TestParseSmiTextOutput_Basic(t *testing.T) {
	devices, err := parseSmiTextOutput(sampleTextOutput)
	if err != nil {
		t.Fatalf("parseSmiTextOutput failed: %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}
}

func TestParseSmiTextOutput_DeviceFields(t *testing.T) {
	devices, err := parseSmiTextOutput(sampleTextOutput)
	if err != nil {
		t.Fatalf("parseSmiTextOutput failed: %v", err)
	}

	dev0 := devices[0]
	if dev0.Index != 0 {
		t.Errorf("dev0.Index: expected 0, got %d", dev0.Index)
	}
	if dev0.DevName != "Enflame S60G" {
		t.Errorf("dev0.DevName: expected 'Enflame S60G', got '%s'", dev0.DevName)
	}
	if dev0.SKU != "S60G" {
		t.Errorf("dev0.SKU: expected 'S60G', got '%s'", dev0.SKU)
	}
	if dev0.DriverVer != "1.7.2.3" {
		t.Errorf("dev0.DriverVer: expected '1.7.2.3', got '%s'", dev0.DriverVer)
	}
	if dev0.FwVersion != "31.5.3" {
		t.Errorf("dev0.FwVersion: expected '31.5.3', got '%s'", dev0.FwVersion)
	}
	if dev0.VendorID != 0x1e36 {
		t.Errorf("dev0.VendorID: expected 0x1e36, got 0x%x", dev0.VendorID)
	}
	if dev0.DeviceID != 0xc033 {
		t.Errorf("dev0.DeviceID: expected 0xc033, got 0x%x", dev0.DeviceID)
	}
	if dev0.Bus != 0x01 {
		t.Errorf("dev0.Bus: expected 0x01, got 0x%x", dev0.Bus)
	}
	if dev0.MemTotalMiB != 49152 {
		t.Errorf("dev0.MemTotalMiB: expected 49152, got %d", dev0.MemTotalMiB)
	}

	dev1 := devices[1]
	if dev1.Index != 1 {
		t.Errorf("dev1.Index: expected 1, got %d", dev1.Index)
	}
	if dev1.SKU != "L600" {
		t.Errorf("dev1.SKU: expected 'L600', got '%s'", dev1.SKU)
	}
	if dev1.MemTotalMiB != 65536 {
		t.Errorf("dev1.MemTotalMiB: expected 65536, got %d", dev1.MemTotalMiB)
	}
}

func TestParseSmiTextOutput_Empty(t *testing.T) {
	devices, err := parseSmiTextOutput("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 0 {
		t.Errorf("expected 0 devices, got %d", len(devices))
	}
}

func TestParseSmiTextOutput_SkipsBAR1TotalSize(t *testing.T) {
	// BAR1 "Total Size : 16 MiB" should NOT overwrite Device Mem "Total Size"
	devices, err := parseSmiTextOutput(sampleTextOutput)
	if err != nil {
		t.Fatalf("parseSmiTextOutput failed: %v", err)
	}
	if devices[0].MemTotalMiB != 49152 {
		t.Errorf("MemTotalMiB should be 49152 (Device Mem), got %d (possibly overwritten by BAR1)", devices[0].MemTotalMiB)
	}
}

const sampleTextOutputLegacyMemSize = `
DEV ID 0
    Driver Info
        Ver                     : 1.5.0.0
    Device Info
        Dev Name                : Enflame T20
    Device Mem Info
        Mem Size                : 32768 MiB
    BAR1 Mem Info
        Total Size              : 16 MiB
`

func TestParseSmiTextOutput_LegacyMemSize(t *testing.T) {
	devices, err := parseSmiTextOutput(sampleTextOutputLegacyMemSize)
	if err != nil {
		t.Fatalf("parseSmiTextOutput failed: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}
	if devices[0].MemTotalMiB != 32768 {
		t.Errorf("MemTotalMiB: expected 32768 from legacy Mem Size, got %d", devices[0].MemTotalMiB)
	}
}

func TestParseSmiTextMemSizeMiB(t *testing.T) {
	tests := []struct {
		name   string
		val    string
		want   uint64
		wantOK bool
	}{
		{
			name:   "efsmi device mem total size",
			val:    "49152 MiB",
			want:   49152,
			wantOK: true,
		},
		{
			name:   "legacy mem size",
			val:    "32768 MiB",
			want:   32768,
			wantOK: true,
		},
		{
			name:   "leading and trailing spaces",
			val:    "  65536 MiB  ",
			want:   65536,
			wantOK: true,
		},
		{
			name:   "bar1 small size",
			val:    "16 MiB",
			want:   16,
			wantOK: true,
		},
		{
			name:   "zero mib",
			val:    "0 MiB",
			want:   0,
			wantOK: true,
		},
		{
			name:   "missing mib suffix",
			val:    "49152",
			want:   0,
			wantOK: false,
		},
		{
			name:   "empty string",
			val:    "",
			want:   0,
			wantOK: false,
		},
		{
			name:   "non numeric before mib",
			val:    "abc MiB",
			want:   0,
			wantOK: false,
		},
		{
			name:   "mib only",
			val:    "MiB",
			want:   0,
			wantOK: false,
		},
		{
			name:   "negative value",
			val:    "-1 MiB",
			want:   0,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseSmiTextMemSizeMiB(tt.val)
			if ok != tt.wantOK {
				t.Errorf("ok: got %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("value: got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestFallbackSelection_ThreeTier(t *testing.T) {
	// Verify all three backends implement Efml
	var _ Efml = &EfmlLib{}
	var _ Efml = &EfmlLibText{}
	var _ Efml = &EfmlLibLegacy{}
}
