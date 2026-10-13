/*
 * Copyright (c) 2020, NVIDIA CORPORATION.  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright (C) 2024 Enflame Technologies, Inc.
 */

package main

import (
	"fmt"
	"log"
	"strings"
)

// Constants representing different MIG strategies.
const (
	MigStrategyNone   = "none"
	MigStrategySingle = "single"
	MigStrategyMixed  = "mixed"
)

// MigStrategy defines the strategy to use for setting labels on MIG devices.
type MigStrategy interface {
	GenerateLabels() (map[string]string, error)
}

// MigDeviceCounts maintains a count of unique MIG device types across all GPUs on a node
type MigDeviceCounts map[string]int

// NewMigStrategy creates a new MIG strategy to generate labels with.
func NewMigStrategy(strategy string, efml Efml, vgcuEnable bool) (MigStrategy, error) {
	switch strategy {
	case MigStrategyNone:
		return &migStrategyNone{efml, vgcuEnable}, nil
	case MigStrategySingle:
		return &migStrategySingle{efml, vgcuEnable}, nil
	case MigStrategyMixed:
		return &migStrategyMixed{efml, vgcuEnable}, nil
	}
	return nil, fmt.Errorf("unknown strategy: %v", strategy)
}

type migStrategyNone struct {
	efml       Efml
	vgcuEnable bool
}
type migStrategySingle struct {
	efml       Efml
	vgcuEnable bool
}
type migStrategyMixed struct {
	efml       Efml
	vgcuEnable bool
}

// migStrategyNone
func (s *migStrategyNone) GenerateLabels() (map[string]string, error) {
	labels := make(map[string]string)

	if err := s.getLabelCount(labels); err != nil {
		return nil, err
	}

	if err := s.getLabelDriverVer(labels); err != nil {
		return nil, err
	}

	h, err := s.efml.GetDeviceHandle(0)
	if err != nil {
		return nil, fmt.Errorf("error getting device: %v", err)
	}

	devSKU, err := s.efml.GetDeviceSKU(h)
	if err != nil {
		return nil, fmt.Errorf("error getting device sku: %v", err)
	}
	if s.vgcuEnable {
		labels[VDeviceModel] = devSKU
	} else {
		labels[DeviceModel] = devSKU
	}
	family, err := s.efml.GetDeviceHwArchName(h)
	if err != nil {
		log.Printf("get hw arch name error: %v", err)
		family = "UNKNOWN"
	}
	s.getLabelFamily(family, labels, devSKU)
	s.getLabelProduct(labels, devSKU)

	memory, err := s.efml.GetDeviceMemoryTotalGBSize(h)
	if err != nil {
		return nil, fmt.Errorf("error getting device memory total size: %v", err)
	}
	if !s.vgcuEnable {
		labels[DeviceMemorySize] = fmt.Sprintf("%d", memory*1024)
	} else {
		s.getVGCULabelMemory(labels, devSKU, memory*1024)
		if labels[VDeviceMemorySize] == "0" {
			return nil, fmt.Errorf("gcu sku: %s is not support vdevice", devSKU)
		}
	}
	return labels, nil
}

func (s *migStrategyNone) getLabelDriverVer(labels map[string]string) error {
	driverVer, err := s.efml.GetDriverVer()
	if err != nil {
		log.Printf("error getting driver version: %v", err)
		return err
	}

	if s.vgcuEnable {
		labels[VDeviceDriverVer] = driverVer
	} else {
		labels[DeviceDriverVer] = driverVer
	}
	return nil
}

func (s *migStrategyNone) getVGCULabelMemory(labels map[string]string, devSKU string, totalMem uint64) {
	deviceMap := map[string]string{
		FourVGCU: "T20, T21, T30, T31, t20, t21, t30, t31, T10, T11, t10, t11",
		TwoVGCU:  "I20, i20, I21, i21",
	}
	var vgcuMem uint64
	if strings.Contains(deviceMap[FourVGCU], devSKU) {
		vgcuMem = totalMem / 4
	} else if strings.Contains(deviceMap[TwoVGCU], devSKU) {
		vgcuMem = totalMem / 2
	} else {
		log.Printf("error: device model %s is not support VGCU", devSKU)
	}
	labels[VDeviceMemorySize] = fmt.Sprintf("%d", vgcuMem)
}

func (s *migStrategyNone) getLabelFamily(family string, labels map[string]string, devSKU string) {
	if family == "UNKNOWN" {
		mapFamily, ok := skuToFamily(normalizeSKU(devSKU))
		if ok {
			family = mapFamily
		}
	}
	if s.vgcuEnable {
		labels[VDeviceFamily] = family
	} else {
		labels[DeviceFamily] = family
	}
}

func (s *migStrategyNone) getLabelCount(labels map[string]string) error {
	if s.vgcuEnable {
		vcount, err := s.efml.GetVDeviceCount()
		if err != nil {
			log.Printf("error getting vdevice count: %v", err)
			return err
		}
		labels[VDeviceCount] = fmt.Sprintf("%d", vcount)
		return nil
	}
	count, err := s.efml.GetDeviceCount()
	if err != nil {
		log.Printf("error getting device count: %v", err)
		return err
	}
	labels[DeviceCount] = fmt.Sprintf("%d", count)
	return nil
}

func (s *migStrategyNone) getLabelProduct(labels map[string]string, devSKU string) {
	family, ok := skuToFamily(normalizeSKU(devSKU))
	if !ok {
		if s.vgcuEnable {
			labels[VDeviceProduct] = "UNKNOWN"
		} else {
			labels[DeviceProduct] = "UNKNOWN"
		}
		return
	}
	lowerFamily := strings.ToLower(family)
	if s.vgcuEnable {
		labels[VDeviceProduct] = lowerFamily
	} else {
		labels[DeviceProduct] = lowerFamily
	}

	// Variant specific product rewrites (e.g. board revision suffix in
	// the SKU) are driven by the productVariants section of sku_family.json.
	applyProductVariants(labels, devSKU, s.vgcuEnable)
}

// migStrategySingle
func (s *migStrategySingle) GenerateLabels() (map[string]string, error) {
	// Generate the same "base" labels as the none strategy
	none, _ := NewMigStrategy(MigStrategyNone, s.efml, s.vgcuEnable)
	labels, err := none.GenerateLabels()
	if err != nil {
		return nil, fmt.Errorf("unable to generate base labels: %v", err)
	}

	// Add a new label specifying the MIG strategy
	labels["enflame.com/mig.strategy"] = "single"

	// Verify that all MIG devices on this node are the same type
	count, err := s.efml.GetDeviceCount()
	if err != nil {
		return nil, fmt.Errorf("error getting device count: %v", err)
	}

	devSKU := ""
	counts := make(MigDeviceCounts)
	for index := uint32(0); index < uint32(count); index++ {
		h, err := s.efml.GetDeviceHandle(index)
		if err != nil {
			return nil, fmt.Errorf("error getting device: %v", err)
		}

		devSKU, err = s.efml.GetDeviceSKU(h)
		if err != nil {
			return nil, fmt.Errorf("unable to parse MIG device name: %v", err)
		}
		counts[devSKU]++
	}

	if len(counts) != 1 {
		s.setInvalidMigStrategyLabels(labels, "more than one MIG device type present on node")
		return labels, nil
	}

	// Get the attributes of only the first MIG device (since they are all the same)
	h, err := s.efml.GetDeviceHandle(0)
	if err != nil {
		return nil, fmt.Errorf("error getting device: %v", err)
	}
	devInfo, err := s.efml.GetDevInfo(h)
	if err != nil {
		return nil, fmt.Errorf("unable to get attributes of MIG device: %v", err)
	}
	fwVersion, err := s.efml.GetFwVersion(h)
	if err != nil {
		return nil, fmt.Errorf("unable to get firmware version of MIG device: %v", err)
	}

	// Override some top-level GPU labels set by the 'none' strategy with MIG specific values
	labels["enflame.com/gcu.count"] = fmt.Sprintf("%d", counts[devSKU])
	labels["enflame.com/gcu.model"] = fmt.Sprintf("%s-MIG-%s", labels["enflame.com/gcu.model"], devInfo.Name)

	// Add new MIG specific labels on the top-level GPU type
	//labels["enflame.com/gcu.devinfo.name"] = fmt.Sprintf("%s", devInfo.Name)
	labels["enflame.com/gcu.devinfo.vendorID"] = fmt.Sprintf("0x%x", devInfo.Vendor_Id)
	//labels["enflame.com/gcu.devinfo.deviceID"] = fmt.Sprintf("%d", devInfo.Device_Id)
	//labels["enflame.com/gcu.devinfo.domainID"] = fmt.Sprintf("%d", devInfo.Domain_Id)
	//labels["enflame.com/gcu.devinfo.busID"] = fmt.Sprintf("%d", devInfo.Bus_Id)
	//labels["enflame.com/gcu.devinfo.devID"] = fmt.Sprintf("%d", devInfo.Dev_Id)
	//labels["enflame.com/gcu.devinfo.funcID"] = fmt.Sprintf("%d", devInfo.Func_Id)
	labels["enflame.com/gcu.firmware"] = fwVersion

	return labels, nil
}

// setInvalidMigStrategyLabels modifies the labels in-place; indicating that the device configuration is invalid for
// the 'single' MIG strategy
func (s *migStrategySingle) setInvalidMigStrategyLabels(labels map[string]string, reason string) {
	log.Printf("WARNING: Invalid configuration detected for mig-strategy=single: %v", reason)

	labels["enflame.com/gcu.count"] = "0"
	labels["enflame.com/gcu.memory"] = "0"
	labels["enflame.com/gcu.model"] = fmt.Sprintf("%s-MIG-INVALID", labels["enflame.com/gcu.model"])
}

// migStrategyMixed
func (s *migStrategyMixed) GenerateLabels() (map[string]string, error) {
	// Generate the same "base" labels as the none strategy
	none, _ := NewMigStrategy(MigStrategyNone, s.efml, s.vgcuEnable)
	labels, err := none.GenerateLabels()
	if err != nil {
		return nil, fmt.Errorf("unable to generate base labels: %v", err)
	}

	// Add a new label specifying the MIG strategy
	labels["enflame.com/mig.strategy"] = "mixed"

	count, err := s.efml.GetDeviceCount()
	if err != nil {
		return nil, fmt.Errorf("error getting device count: %v", err)
	}

	// Add new MIG related labels on each individual MIG type
	devSKU := ""
	counts := make(MigDeviceCounts)
	for index := uint32(0); index < uint32(count); index++ {
		h, err := s.efml.GetDeviceHandle(index)
		if err != nil {
			return nil, fmt.Errorf("error getting device: %v", err)
		}

		devSKU, err = s.efml.GetDeviceSKU(h)
		if err != nil {
			return nil, fmt.Errorf("unable to parse MIG device name: %v", err)
		}

		memory, err := s.efml.GetDeviceMemoryTotalGBSize(h)
		if err != nil {
			return nil, fmt.Errorf("unable to get MIG device memory: %v", err)
		}

		// Only set labels for a MIG device type the first time we encounter it
		if counts[devSKU] == 0 {
			devInfo, err := s.efml.GetDevInfo(h)
			if err != nil {
				return nil, fmt.Errorf("unable to get attributes of MIG device: %v", err)
			}
			fwVersion, err := s.efml.GetFwVersion(h)
			if err != nil {
				return nil, fmt.Errorf("unable to get firmware version of MIG device: %v", err)
			}

			prefix := fmt.Sprintf("enflame.com/mig-%s", devSKU)
			labels[prefix+".memory"] = fmt.Sprintf("%d", memory*1024)
			//labels[prefix+".devinfo.name"] = fmt.Sprintf("%s", devInfo.Name)
			labels[prefix+".devinfo.vendorID"] = fmt.Sprintf("0x%x", devInfo.Vendor_Id)
			//labels[prefix+"devinfo.deviceID"] = fmt.Sprintf("%d", devInfo.Device_Id)
			//labels[prefix+"devinfo.domainID"] = fmt.Sprintf("%d", devInfo.Domain_Id)
			//labels[prefix+"devinfo.busID"] = fmt.Sprintf("%d", devInfo.Bus_Id)
			//labels[prefix+"devinfo.devID"] = fmt.Sprintf("%d", devInfo.Dev_Id)
			//labels[prefix+"devinfo.funcID"] = fmt.Sprintf("%d", devInfo.Func_Id)
			labels[prefix+".firmware"] = fwVersion
		}

		// Maintain the total count of this MIG device type for setting later
		counts[devSKU]++
	}

	// Set the total count on each new MIG type
	for name, count := range counts {
		prefix := fmt.Sprintf("enflame.com/mig-%s", name)
		labels[prefix+".count"] = fmt.Sprintf("%d", count)
	}

	return labels, nil
}
