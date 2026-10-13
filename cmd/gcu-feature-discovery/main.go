/*
 * Copyright (c) 2019, NVIDIA CORPORATION. All rights reserved.
 * Modifications Copyright (C) 2024 Enflame Technologies, Inc.
 */
package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Version : Version of the binary
// This will be set using ldflags at compile time
var Version = ""

func main() {
	log.SetPrefix(ComponentName + ": ")

	if Version == "" {
		Version = "0.0.0"
	}

	log.Printf("Running %s in version %s", ComponentName, Version)

	efml := initEfmlWithFallback()

	conf := Conf{}
	conf.getConfFromArgv(os.Args)
	conf.getConfFromEnv()
	log.Print("Loaded configuration:")
	log.Print("Oneshot: ", conf.Oneshot)
	log.Print("FailOnInitError: ", conf.FailOnInitError)
	log.Print("SleepInterval: ", conf.SleepInterval)
	log.Print("DeviceWatchInterval: ", conf.DeviceWatchInterval)
	log.Print("MigStrategy: ", conf.MigStrategy)
	log.Print("NoTimestamp: ", conf.NoTimestamp)
	log.Print("OutputFilePath: ", conf.OutputFilePath)

	log.Print("Start running")
	err := run(efml, conf)
	if err != nil {
		log.Printf("Unexpected error: %v", err)
		os.Exit(1)
	}
	log.Print("Exiting")
}

// initEfmlWithFallback tries three backends in order:
//  1. efsmi -q --json-format (EfmlLib)
//  2. efsmi -q text parsing  (EfmlLibText)
//  3. libefml.so CGO          (EfmlLibLegacy)
func initEfmlWithFallback() Efml {
	// Try 1: efsmi JSON
	smiJson := &EfmlLib{}
	if err := smiJson.Init(); err == nil {
		log.Printf("Using backend: efsmi JSON")
		return smiJson
	} else {
		log.Printf("efsmi JSON backend failed: %v, trying text parsing...", err)
	}

	// Try 2: efsmi text
	smiText := &EfmlLibText{}
	if err := smiText.Init(); err == nil {
		log.Printf("Using backend: efsmi text parsing")
		return smiText
	} else {
		log.Printf("efsmi text backend failed: %v, falling back to libefml...", err)
	}

	// Try 3: libefml (legacy)
	legacy := &EfmlLibLegacy{}
	if err := legacy.Init(); err == nil {
		log.Printf("Using backend: libefml (legacy)")
		return legacy
	} else {
		log.Printf("libefml backend init failed: %v", err)
	}

	// All backends failed, return uninitialized legacy (run() will retry Init and report error)
	log.Printf("Warning: all backends failed, will retry initialization in run loop")
	return &EfmlLibLegacy{}
}

func run(efml Efml, conf Conf) error {
	defer checkRemoveOutputFile(conf.OutputFilePath, conf.Oneshot)
	exitChan := make(chan bool)
	getExitSignal(exitChan)
	defer func() {
		err := efml.Shutdown()
		if err != nil {
			log.Printf("Warning: Shutdown of EFML returned: %v", err)
		}
	}()

	if err := efml.Init(); err != nil {
		return fmt.Errorf("failed to initialize EFML: %v", err)
	}
	log.Println("efml init success")

	stopWatch := make(chan struct{})
	var watchWG sync.WaitGroup
	var efmlMu sync.Mutex
	var watchInitFail chan error
	if !conf.Oneshot && conf.DeviceWatchInterval > 0 {
		watchInitFail = make(chan error, 1)
		watchWG.Add(1)
		go func() {
			defer watchWG.Done()
			runDeviceWatchLoop(efml, conf.DeviceWatchInterval, &efmlMu, stopWatch, watchInitFail)
		}()
		log.Printf("device watch: enabled, interval=%v", conf.DeviceWatchInterval)
	} else if !conf.Oneshot {
		log.Print("device watch: disabled (interval<=0), EFML Init runs every label cycle")
	}
	defer func() {
		close(stopWatch)
		watchWG.Wait()
	}()

	gfdLabels := make(map[string]string)
	if !conf.NoTimestamp {
		getDeployedTime(gfdLabels)
	}

L:
	for {
		err := func() error {
			if !conf.Oneshot && conf.DeviceWatchInterval > 0 {
				efmlMu.Lock()
				defer efmlMu.Unlock()
			} else if err := efml.Init(); err != nil {
				return fmt.Errorf("failed to initialize EFML: %v", err)
			}

			loadSKUFamilyConfig()
			efmlLabels, err := getEFMLLabels(efml, conf.MigStrategy)
			if err != nil {
				_, isInitError := err.(EfmlInitError)
				if !isInitError || (isInitError && conf.FailOnInitError) {
					return fmt.Errorf("error generating EFML labels: %v", err)
				}
				log.Printf("Warning: Error generating EFML labels: %v", err)
			}

			if len(efmlLabels) == 0 {
				log.Printf("Warning: no labels generated from any source")
			}
			gfdLabels[LabeledTimestamp] = time.Now().Format("2006-01-02-15-04-05")

			allLabels := []map[string]string{
				gfdLabels,
				efmlLabels,
			}
			applyMirrorDomains(allLabels)
			if err := replaceLabelsByConfig(efml, &allLabels); err != nil {
				return fmt.Errorf("error replace labels by config file")
			}
			log.Printf("Writing labels to output file, labels list: %v", allLabels)
			if err := writeLabelsToFile(conf.OutputFilePath, allLabels...); err != nil {
				return fmt.Errorf("error writing file '%s': %v", conf.OutputFilePath, err)
			}
			return nil
		}()
		if err != nil {
			return err
		}

		if conf.Oneshot {
			break
		}

		log.Print("Sleeping for ", conf.SleepInterval)

		select {
		case <-exitChan:
			break L
		case err := <-watchInitFail:
			return fmt.Errorf("failed to initialize EFML: %v", err)
		case <-time.After(conf.SleepInterval):
			break
		}
	}

	return nil
}

func replaceLabelsByConfig(efml Efml, allLabels *[]map[string]string) error {
	// replace by topscloud.json
	vcount, err := efml.GetVDeviceCount()
	if err != nil {
		return fmt.Errorf("error getting device vcount: %v", err)
	}
	config, err := getConfig(topscloudConfigFile)
	if err != nil {
		log.Printf("get topscloud config filed: %v", err)
		return err
	}
	oldStr, newStr := Domain+"gcu", config.getExtendFeature().([]string)[0]
	if vcount > 0 {
		oldStr, newStr = Domain+"vgcu", config.getExtendFeature().([]string)[1]
	}
	for index, labelsMap := range *allLabels {
		newLabelsMap := make(map[string]string)
		for key, value := range labelsMap {
			newKey := key
			if strings.HasPrefix(key, oldStr) {
				// replace enflame.com/gcu or enflame.com/vgcu
				newKey = newStr + key[len(oldStr):]
			} else if strings.HasPrefix(key, Domain) {
				// replace enflame.com
				newKey = config.(*topscloudConfig).domain() + "/" + key[len(Domain):]
			}
			if newKey != key && !strings.HasPrefix(newKey, Domain) {
				log.Printf("Warn: label: '%s' domain replaced to: '%s' by topscloud.json, make sure nfd can recognize new domain",
					key, config.(*topscloudConfig).domain())
			}
			newLabelsMap[newKey] = value
		}
		(*allLabels)[index] = newLabelsMap
	}

	// replace by gfd.json
	gfdConfig, err := getConfig(gfdConfigFile)
	if err != nil {
		log.Printf("get gfd config filed: %v", err)
		return err
	}
	labelsFromFile := gfdConfig.getExtendFeature().(map[string]string)
	nonEmptyLabels := make(map[string]string)
	for key, value := range labelsFromFile {
		if strings.TrimSpace(value) != "" {
			nonEmptyLabels[key] = value
			log.Printf("Update labels from file: gfd.json, Key: %s, Value: %s", key, value)
		}
	}
	if len(nonEmptyLabels) > 0 {
		*allLabels = append(*allLabels, nonEmptyLabels)
	}
	return nil
}

func getEFMLLabels(efml Efml, MigStrategy string) (map[string]string, error) {
	_, err := efml.CheckDevCountState()
	if err != nil {
		return nil, EfmlInitError{fmt.Errorf("failed to check device count: %v", err)}
	}

	vcount, err := efml.GetVDeviceCount()
	if err != nil {
		return nil, fmt.Errorf("error getting device vcount: %v", err)
	}
	if vcount > 0 {
		return makeLabels(efml, MigStrategy, true)
	}

	count, err := efml.GetDeviceCount()
	if err != nil {
		return nil, fmt.Errorf("error getting device count: %v", err)
	}
	if count > 0 {
		return makeLabels(efml, MigStrategy, false)
	}
	err = fmt.Errorf("error: gcu or vgcu device is not found in this node")
	log.Println(err.Error())
	return nil, err
}

func makeLabels(efml Efml, MigStrategy string, vgcuEnable bool) (map[string]string, error) {
	commonLabels, err := generateCommonLabels(efml, vgcuEnable)
	if err != nil {
		return nil, fmt.Errorf("error generating common labels: %v", err)
	}

	migStrategy, err := NewMigStrategy(MigStrategy, efml, vgcuEnable)
	if err != nil {
		return nil, fmt.Errorf("error creating MIG strategy: %v", err)
	}

	migStrategyLabels, err := migStrategy.GenerateLabels()
	if err != nil {
		return nil, fmt.Errorf("error generating labels from MIG strategy: %v", err)
	}

	allLabels := make(map[string]string)
	for k, v := range commonLabels {
		allLabels[k] = v
	}
	for k, v := range migStrategyLabels {
		allLabels[k] = v
	}
	if vgcuEnable {
		allLabels["enflame.com/vgcu.present"] = "true"
	}
	return allLabels, nil
}

func generateCommonLabels(efml Efml, vgcuEnable bool) (map[string]string, error) {
	machineType, err := getMachineType(MachineTypePath)
	if err != nil {
		return nil, fmt.Errorf("error getting machine type: %v", err)
	}

	labels := make(map[string]string)
	if vgcuEnable {
		labels[VDeviceMachineType] = strings.Replace(machineType, " ", "-", -1)
	} else {
		labels[DeviceMachineType] = strings.Replace(machineType, " ", "-", -1)
	}
	return labels, nil
}

func getMachineType(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}


// writeLabelsToFile writes a set of labels to the specified path. The file is written atomically
func writeLabelsToFile(path string, labelSets ...map[string]string) error {
	output := new(bytes.Buffer)
	for _, labels := range labelSets {
		for k, v := range labels {
			fmt.Fprintf(output, "%s=%s\n", k, v)
		}
	}
	err := writeFileAtomically(path, output.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("error atomically writing file '%s': %v", path, err)
	}
	return nil
}

func writeFileAtomically(path string, contents []byte, perm os.FileMode) error {
	fileName := strings.Split(path, "/")[len(strings.Split(path, "/"))-1]
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to retrieve absolute path of output file: %v", err)
	}

	absDir := filepath.Dir(absPath)
	tmpDir := filepath.Join(absDir, fmt.Sprintf("%s-tmp", fileName))

	err = os.Mkdir(tmpDir, os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create temporary directory: %v", err)
	}
	defer func() {
		if err != nil {
			os.RemoveAll(tmpDir)
		}
	}()

	tmpFile, err := os.CreateTemp(tmpDir, fmt.Sprintf("%s-", fileName))
	if err != nil {
		return fmt.Errorf("fail to create temporary output file: %v", err)
	}
	defer func() {
		if err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
		}
	}()

	err = os.WriteFile(tmpFile.Name(), contents, perm)
	if err != nil {
		return fmt.Errorf("error writing temporary file '%v': %v", tmpFile.Name(), err)
	}

	err = os.Rename(tmpFile.Name(), path)
	if err != nil {
		return fmt.Errorf("error moving temporary file to '%v': %v", path, err)
	}

	err = os.Chmod(path, perm)
	if err != nil {
		return fmt.Errorf("error setting permissions on '%v': %v", path, err)
	}

	return nil
}

func checkRemoveOutputFile(path string, oneShot bool) {
	fileName := strings.Split(path, "/")[len(strings.Split(path, "/"))-1]
	if oneShot {
		return
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		log.Printf("failed to retrieve absolute path of output file: %v", err)
		return
	}

	absDir := filepath.Dir(absPath)
	tmpDir := filepath.Join(absDir, fmt.Sprintf("%s-tmp", fileName))

	err = os.RemoveAll(tmpDir)
	if err != nil {
		log.Printf("failed to remove temporary output directory: %v", err)
		return
	}

	err = os.Remove(absPath)
	if err != nil {
		log.Printf("failed to remove output file: %v", err)
		return
	}
}
