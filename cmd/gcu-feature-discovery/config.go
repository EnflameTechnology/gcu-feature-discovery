/*
 * Copyright (C) 2024 Enflame Technologies, Inc.
 */
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	// for k8s deploy and expose to users
	configPath string = "/etc/topscloud/"
	// for execute binary files individually
	configReleasePath string = "./config/"
	// for k8s deploy, used to init hostpath
	configInitPath string = "/tmp/"
	// config file name
	topscloudConfigFile string = "topscloud.json"
	gfdConfigFile       string = "gfd.json"
)

type config interface {
	getExtendFeature() interface{}
}

type gfdConfig struct {
	Labels map[string]string `json:"labels"`
}

func (rer *gfdConfig) getExtendFeature() interface{} {
	return rer.Labels
}

type topscloudConfig struct {
	Version          string   `json:"version"`
	RegisterResource []string `json:"registerResource"`
	MainDevice       []string `json:"mainDevice"`
	VirtualDevice    []string `json:"virtualDevice"`
	DriverDevice     []string `json:"driverDevice"`
	UtilityBins      []string `json:"utilityBins"`
	UtilityLibs      []string `json:"utilityLibs"`
}

func (rer *topscloudConfig) getExtendFeature() interface{} {
	return rer.RegisterResource
}

func (rer *topscloudConfig) domain() string {
	return strings.Split(rer.RegisterResource[0], "/")[0]
}

func getConfig(fileName string) (config, error) {
	filePath := configPath + fileName
	fileInitPath := configInitPath + fileName
	// if the config file does not exist, read from the /tmp directory, and init it to /etc/topscloud
	if !fileIsExist(filePath) {
		log.Printf("config file: %s is not exist", filePath)
		if !fileIsExist(fileInitPath) {
			fileInitPath = configReleasePath + fileName
			log.Printf("init config file: %s not exist, use: %s", configInitPath+fileName, configReleasePath+fileName)
		}
		if err := copyFile(fileInitPath, filePath); err != nil {
			return nil, err
		}
		log.Printf("init config file: %s from %s success", filePath, fileInitPath)
	} else {
		log.Printf("found already exist config file: %s", filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("error: read config file: %s failed: %v", filePath, err)
		return nil, err
	}

	if fileName == topscloudConfigFile {
		config := new(topscloudConfig)
		if err := json.Unmarshal(content, config); err != nil {
			log.Printf("error: json unmarshal content: %s to struct failed: %v", string(content), err)
			return nil, err
		}
		// if registerResource in config file is invalid, replace it by default value
		if len(config.RegisterResource) != 2 {
			defaultResource := []string{Domain + "gcu", Domain + "vgcu"}
			log.Printf("warn: registerResource: %v in config file is invalid, replace it by: %v",
				config.RegisterResource, defaultResource)
			config.RegisterResource = defaultResource
		}
		return config, nil
	}
	if fileName == gfdConfigFile {
		config := new(gfdConfig)
		if err := json.Unmarshal(content, config); err != nil {
			log.Printf("error: json unmarshal content: %s to struct failed: %v", string(content), err)
			return nil, err
		}
		return config, nil
	}
	if fileName == skuFamilyConfigFile {
		config := new(skuFamilyConfig)
		if err := json.Unmarshal(content, config); err != nil {
			log.Printf("error: json unmarshal content: %s to struct failed: %v", string(content), err)
			return nil, err
		}
		return config, nil
	}
	return nil, fmt.Errorf("config file: %s is not support", fileName)
}

func copyFile(src, dst string) error {
	// open the source file
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		log.Printf("error: os stat source file: %s failed: %v", src, err)
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("error: %s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		log.Printf("error: open the source file: %s failed: %v", src, err)
		return err
	}
	defer source.Close()

	// special scenario: consider the case where the specified YAML file is not used
	dir, _ := filepath.Split(dst)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("error: create config file parents folders: %s failed: %v", dir, err)
		return err
	}
	// create the destination file
	destination, err := os.Create(dst)
	if err != nil {
		log.Printf("error: create the destination file: %s failed: %v", dst, err)
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	if err != nil {
		log.Printf("error: copy the source file: %s to destination file: %s failed: %v", src, dst, err)
		return err
	}
	return nil
}
