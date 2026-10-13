/*
 * Copyright (C) 2024 Enflame Technologies, Inc.
 */

package main

import (
	"fmt"
	"enflame/gcu-feature-discovery/internal/eflib"
	"enflame/gcu-feature-discovery/internal/eflib/efml"
	"log"
)

// EfmlLibLegacy : Implementation of Efml using the libefml.so library (CGO)
type EfmlLibLegacy struct {
	handles     []efml.Handle
	initialized bool
}

// Init : Init EFML lib (idempotent: safe to call multiple times)
func (e *EfmlLibLegacy) Init() error {
	if e.initialized {
		return nil
	}
	if err := efml.InitV2(false); err != nil {
		return err
	}
	e.initialized = true
	log.Printf("efml(legacy): initialized via libefml.so")
	return nil
}

// Shutdown : Shutdown EFML lib
func (e *EfmlLibLegacy) Shutdown() error {
	return efml.Shutdown()
}

func (e *EfmlLibLegacy) GetDriverVer() (string, error) {
	return eflib.GetDriverVer()
}

func (e *EfmlLibLegacy) CheckDevCountState() (uint, error) {
	return eflib.CheckDevCountState()
}

func (e *EfmlLibLegacy) GetDeviceCount() (uint32, error) {
	count := 0
	for i := 0; i < GCU_DEV_MAX; i++ {
		devicePath := fmt.Sprintf("%s%d", GCU_DEV_NAME, i)
		if fileIsExist(devicePath) {
			count++
		}
	}
	return uint32(count), nil
}

func (e *EfmlLibLegacy) GetVDeviceCount() (uint32, error) {
	return eflib.GetVDeviceCount()
}

func (e *EfmlLibLegacy) GetDeviceHandle(id uint32) (DeviceHandle, error) {
	h, err := eflib.GetDeviceHandle(id)
	if err != nil {
		return 0, err
	}
	// Store handle for later use
	for int(id) >= len(e.handles) {
		e.handles = append(e.handles, efml.Handle{})
	}
	e.handles[id] = h
	return DeviceHandle(id), nil
}

func (e *EfmlLibLegacy) getHandle(h DeviceHandle) (efml.Handle, error) {
	if int(h) < len(e.handles) {
		return e.handles[h], nil
	}
	return efml.Handle{}, fmt.Errorf("invalid device handle: %d (total %d)", h, len(e.handles))
}

func (e *EfmlLibLegacy) GetDeviceSKU(h DeviceHandle) (string, error) {
	handle, err := e.getHandle(h)
	if err != nil {
		return "", err
	}
	return eflib.GetDeviceSKU(handle)
}

func (e *EfmlLibLegacy) GetDeviceHwArchName(h DeviceHandle) (string, error) {
	handle, err := e.getHandle(h)
	if err != nil {
		return "", err
	}
	return eflib.GetDeviceHwArchName(handle)
}

func (e *EfmlLibLegacy) GetDeviceMemoryTotalGBSize(h DeviceHandle) (uint64, error) {
	handle, err := e.getHandle(h)
	if err != nil {
		return 0, err
	}
	return eflib.GetDeviceMemoryTotalGBSize(handle)
}

func (e *EfmlLibLegacy) GetDevInfo(h DeviceHandle) (*DeviceInfo, error) {
	handle, err := e.getHandle(h)
	if err != nil {
		return nil, err
	}
	devInfo, err := handle.GetDevInfo()
	if err != nil {
		log.Printf("Failed to get device info")
		return nil, err
	}
	return &DeviceInfo{
		Name:      devInfo.Name,
		Vendor_Id: uint32(devInfo.Vendor_Id),
		Device_Id: uint32(devInfo.Device_Id),
		Domain_Id: uint32(devInfo.Domain_Id),
		Bus_Id:    uint32(devInfo.Bus_Id),
		Dev_Id:    uint32(devInfo.Dev_Id),
		Func_Id:   uint32(devInfo.Func_Id),
	}, nil
}

func (e *EfmlLibLegacy) GetFwVersion(h DeviceHandle) (string, error) {
	handle, err := e.getHandle(h)
	if err != nil {
		return "", err
	}
	fwVersion, err := handle.GetFwVersion()
	if err != nil {
		log.Printf("Failed to get firmware version of device")
		return "INVALID", err
	}
	return fwVersion, nil
}
