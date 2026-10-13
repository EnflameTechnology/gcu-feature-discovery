/*
 * Copyright (C) 2026 Enflame Technologies, Inc.
 */
package main

import "testing"

func TestApplyMirrorDomains(t *testing.T) {
	old := currentSKUConfig
	currentSKUConfig = &skuFamilyConfig{
		MirrorDomains: []mirrorDomain{{
			Domain: "mirror.example.com/",
			KeyMap: map[string]string{"gcu.": "gpu.", "gfd.": "gfd."},
			ValueMap: map[string]map[string]string{
				"gpu.product": {"family_a": "family_a-x"},
			},
		}},
	}
	defer func() { currentSKUConfig = old }()

	labels := map[string]string{
		Domain + "gcu.count":     "8",
		Domain + "gcu.product":   "family_a",
		Domain + "gcu.family":    "FAMILY_A",
		Domain + "gfd.timestamp": "2026-01-01-00-00-00",
		Domain + "vgcu.count":    "2",
		Domain + "mig.strategy":  "single",
		"other.domain/key":       "v",
	}
	applyMirrorDomains([]map[string]string{labels})

	expect := map[string]string{
		"mirror.example.com/gpu.count":     "8",
		"mirror.example.com/gpu.product":   "family_a-x",
		"mirror.example.com/gpu.family":    "FAMILY_A",
		"mirror.example.com/gfd.timestamp": "2026-01-01-00-00-00",
	}
	for k, v := range expect {
		if labels[k] != v {
			t.Errorf("%s: expected %q, got %q", k, v, labels[k])
		}
	}
	for _, absent := range []string{
		"mirror.example.com/vgcu.count",
		"mirror.example.com/gpu.strategy",
		"mirror.example.com/mig.strategy",
	} {
		if _, ok := labels[absent]; ok {
			t.Errorf("%s should not be mirrored", absent)
		}
	}
	if labels[Domain+"gcu.product"] != "family_a" {
		t.Errorf("primary product must stay untouched, got %q", labels[Domain+"gcu.product"])
	}
}

func TestApplyMirrorDomains_EmptyConfig(t *testing.T) {
	old := currentSKUConfig
	currentSKUConfig = &skuFamilyConfig{}
	defer func() { currentSKUConfig = old }()

	labels := map[string]string{Domain + "gcu.count": "8"}
	applyMirrorDomains([]map[string]string{labels})
	if len(labels) != 1 {
		t.Errorf("no mirror domains configured: labels must be unchanged, got %v", labels)
	}
}
