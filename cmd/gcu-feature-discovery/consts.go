/*
 * Copyright (C) 2024 Enflame Technologies, Inc.
 */

package main

// component info
const (
	ComponentName = "gcu-feature-discovery"
)

// labels path
const (
	// MachineTypePath : Path to the file describing the machine type
	// This will be override during unit testing
	MachineTypePath = "/sys/class/dmi/id/product_name"
)

// labels for gcu, Domain = enflame.com
const (
	DeployedTimestamp = Domain + "gfd.timestamp"
	LabeledTimestamp  = Domain + "gfd.latestLabeledTimestamp"
	DeviceDriverVer   = Domain + "gcu.driverVer"
	DeviceMachineType = Domain + "gcu.machine"
	DeviceMemorySize  = Domain + "gcu.memory"
	DeviceModel       = Domain + "gcu.model"
	DeviceCount       = Domain + "gcu.count"
	DeviceFamily      = Domain + "gcu.family"
	DeviceProduct     = Domain + "gcu.product"
)

// labels for vgcu, Domain = enflame.com
const (
	VDeviceDriverVer   = Domain + "vgcu.driverVer"
	VDeviceMachineType = Domain + "vgcu.machine"
	VDeviceMemorySize  = Domain + "vgcu.memory"
	VDeviceModel       = Domain + "vgcu.model"
	VDeviceCount       = Domain + "vgcu.count"
	VDeviceFamily      = Domain + "vgcu.family"
	VDeviceProduct     = Domain + "vgcu.product"
)

// other
const (
	GCU_DEV_MAX  = 64
	GCU_DEV_NAME = "/dev/gcu"
	TwoVGCU      = "twoVGCU"
	FourVGCU     = "fourVGCU"
	Domain       = "enflame.com/"
)