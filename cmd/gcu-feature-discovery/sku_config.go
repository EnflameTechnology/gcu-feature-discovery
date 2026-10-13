/*
 * Copyright (C) 2026 Enflame Technologies, Inc.
 */
package main

import (
	"log"
	"strings"
)

const skuFamilyConfigFile string = "sku_family.json"

// variantRule rewrites product label values for hardware variants whose
// SKU contains a given marker (e.g. a board revision suffix). The rewrite
// maps are keyed by the current label value.
type variantRule struct {
	SkuContains string            `json:"skuContains"`
	ProductMap  map[string]string `json:"productMap"`
	VProductMap map[string]string `json:"vproductMap"`
}

// skuFamilyConfig externalizes the hardware naming knowledge of gfd:
// which SKU belongs to which hardware family, how a family name maps to
// the product label value of the secondary label domain, and variant
// specific product rewrites. It is loaded from sku_family.json via the
// same /etc/topscloud config chain as gfd.json / topscloud.json, so the
// mapping can be maintained per deployment without code changes.
// mirrorDomain declares a secondary label domain into which the primary
// domain labels are mirrored: keyMap rewrites the key prefix after the
// domain (e.g. "gcu." -> "gpu.") and valueMap optionally rewrites the
// value of specific mirrored keys.
type mirrorDomain struct {
	Domain   string                       `json:"domain"`
	KeyMap   map[string]string            `json:"keyMap"`
	ValueMap map[string]map[string]string `json:"valueMap"`
}

type skuFamilyConfig struct {
	SkuFamily     map[string]string `json:"skuFamily"`
	Variants      []variantRule     `json:"productVariants"`
	MirrorDomains []mirrorDomain    `json:"mirrorDomains"`
}

func (rer *skuFamilyConfig) getExtendFeature() interface{} {
	return rer
}

var currentSKUConfig = &skuFamilyConfig{
	SkuFamily: map[string]string{},
}

// loadSKUFamilyConfig refreshes the SKU/family mapping from the config
// chain. On failure the previous mapping is kept so that a transient
// config problem does not wipe existing labels.
func loadSKUFamilyConfig() {
	cfg, err := getConfig(skuFamilyConfigFile)
	if err != nil {
		log.Printf("warn: load %s failed: %v, keep current mapping (%d skus)",
			skuFamilyConfigFile, err, len(currentSKUConfig.SkuFamily))
		return
	}
	sc := cfg.(*skuFamilyConfig)
	if sc.SkuFamily == nil {
		sc.SkuFamily = map[string]string{}
	}
	currentSKUConfig = sc
}

// skuToFamily resolves a (normalized) SKU to its hardware family.
func skuToFamily(sku string) (string, bool) {
	family, ok := currentSKUConfig.SkuFamily[sku]
	return family, ok
}

// applyProductVariants applies variant specific product label rewrites.
func applyProductVariants(labels map[string]string, devSKU string, vgcuEnable bool) {
	upperSKU := strings.ToUpper(devSKU)
	for _, rule := range currentSKUConfig.Variants {
		if rule.SkuContains == "" || !strings.Contains(upperSKU, strings.ToUpper(rule.SkuContains)) {
			continue
		}
		if vgcuEnable {
			if v, ok := rule.VProductMap[labels[VDeviceProduct]]; ok {
				labels[VDeviceProduct] = v
			}
			continue
		}
		if v, ok := rule.ProductMap[labels[DeviceProduct]]; ok {
			labels[DeviceProduct] = v
		}
	}
}

// applyMirrorDomains mirrors primary-domain labels into the configured
// secondary label domains. A key is mirrored only when its path after
// the primary domain matches a keyMap prefix, so e.g. "vgcu.*" and
// "mig.*" keys stay primary-domain only unless explicitly configured.
func applyMirrorDomains(labelSets []map[string]string) {
	for _, labels := range labelSets {
		for _, md := range currentSKUConfig.MirrorDomains {
			if md.Domain == "" {
				continue
			}
			mirrored := map[string]string{}
			for key, val := range labels {
				if !strings.HasPrefix(key, Domain) {
					continue
				}
				rest := key[len(Domain):]
				for src, dst := range md.KeyMap {
					if !strings.HasPrefix(rest, src) {
						continue
					}
					newRest := dst + rest[len(src):]
					newVal := val
					if vm, ok := md.ValueMap[newRest]; ok {
						if mv, ok := vm[val]; ok {
							newVal = mv
						}
					}
					mirrored[md.Domain+newRest] = newVal
				}
			}
			for k, v := range mirrored {
				labels[k] = v
			}
		}
	}
}
