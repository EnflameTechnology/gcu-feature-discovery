# Enflame GCU Feature Discovery (gcu-feature-discovery)

> **Language**: English | [中文](README_zh.md)

## Table of Contents

- [Overview](#overview)
- [Project Structure](#project-structure)
- [Installation](#installation)
  - [Prerequisites](#prerequisites)
  - [Build from Source](#build-from-source)
  - [Build the Container Image](#build-the-container-image)
- [Quick Start](#quick-start)
  - [Deploy as a DaemonSet](#deploy-as-a-daemonset)
  - [Run once as a Job](#run-once-as-a-job)
  - [Inspect Node Labels](#inspect-node-labels)
- [Key Features](#key-features)
- [Node Labels](#node-labels)
- [Version Compatibility](#version-compatibility)
- [Hardware Support](#hardware-support)
- [User Manual](#user-manual)
- [Trouble Shooting](#trouble-shooting)
- [License](#license)

## Overview

`gcu-feature-discovery` (GFD) labels Kubernetes nodes with the properties and
features of the Enflame GCU (General Compute Unit) devices present on them. It is
analogous to NVIDIA GPU Feature Discovery and is built on the
[node-feature-discovery](https://github.com/kubernetes-sigs/node-feature-discovery)
pattern. The node labels it publishes enable GCU-aware scheduling through
`nodeSelector` and node affinity.

GFD inspects the GCU devices on each node — via `efsmi` and the EFML library
(`libefml.so`, i.e. the [go-eflib](../go-eflib) bindings) — and publishes node
labels under the `enflame.com` domain that describe the discovered hardware.

The project delivers:

- **Automatic node labeling** — discovers GCU model, count, driver version, memory and more, and applies them as Kubernetes node labels.
- **EFML + efsmi backend** — reads device data through `efsmi` and the EFML management library ([go-eflib](../go-eflib)).
- **Kubernetes-native** — runs as a DaemonSet for continuous labeling, or as a one-shot Job.
- **GCU-aware scheduling** — the published labels let you steer workloads with `nodeSelector` / affinity.

## Project Structure

```
gcu-feature-discovery/
├── cmd/                       # Entrypoint (main package)
├── internal/                  # Implementation (includes an embedded eflib package)
├── config/                    # Runtime configuration
├── deployments/               # Deploy scripts and manifests
│   ├── deploy.sh              # Deploy helper
│   ├── delete.sh              # Teardown helper
│   ├── dockerfiles/           # Image build files
│   └── yaml/                  # gcu-feature-discovery-daemonset.yaml
│                              # gcu-feature-discovery-job.yaml.template
├── go.mod                     # Module enflame/gcu-feature-discovery
├── Makefile                   # Builds the gcu-feature-discovery binary
├── versions.mk                # Version definitions
├── topscloud.json             # Bundled device/resource descriptor
└── build.sh                   # Build + package entry point
```

## Installation

### Prerequisites

- A Kubernetes cluster.
- Nodes with Enflame GCU devices and the Enflame GCU driver installed.
- `efsmi` and the EFML runtime library (`libefml.so`) available on the nodes.
- [Go](https://go.dev/) 1.22+ and a C toolchain (the build uses cgo, because it accesses EFML via [go-eflib](../go-eflib)) for building from source.
- Docker for building the container image.

### Build from Source

```bash
# Build the gcu-feature-discovery binary and produce a release package under dist/
./build.sh all

# Clean build artifacts
./build.sh clean
```

`./build.sh all` runs `make` to build the `gcu-feature-discovery` binary and
packages it together with `deployments/` and `config/` under
`dist/`.

### Build the Container Image

Image build files are provided under `deployments/dockerfiles/`. Build the image
with your chosen base image / registry and load or push it so the cluster can
pull it.

## Quick Start

### Deploy as a DaemonSet

GFD normally runs as a DaemonSet so that every GCU node is continuously labeled:

```bash
kubectl apply -f deployments/yaml/gcu-feature-discovery-daemonset.yaml
```

### Run once as a Job

A one-shot Job template is also provided for a single labeling pass:

```bash
# Fill in the template (image, node target, etc.) then apply it
kubectl apply -f deployments/yaml/gcu-feature-discovery-job.yaml.template
```

### Inspect Node Labels

List the labels GFD applied to a node:

```bash
kubectl get node <node> -o jsonpath='{.metadata.labels}' | tr ',' '\n' | grep enflame.com
```

## Key Features

| Feature | Description |
| --- | --- |
| Node labeling | Publishes Enflame GCU features as Kubernetes node labels under the `enflame.com` domain |
| Feature discovery | Discovers GCU model, count, driver version and memory via EFML / `efsmi` |
| DaemonSet mode | Runs continuously to keep labels in sync as nodes change |
| Job mode | One-shot Job template for a single labeling pass |
| GCU-aware scheduling | Labels enable `nodeSelector` / affinity-based scheduling of GCU workloads |
| EFML + efsmi backend | Reads device data through `efsmi` and the EFML library ([go-eflib](../go-eflib)) |

## Node Labels

GFD publishes node labels under the `enflame.com` domain describing the GCU
devices on the node — such as GCU model, device count, driver version and
memory. The exact label keys depend on the discovered hardware and driver, so
list the node labels as shown above to see the actual keys applied to your nodes.

Related device and resource identifiers (from the bundled `topscloud.json`)
include the resources `enflame.com/gcu` and `enflame.com/vgcu`; the device nodes
`/dev/gcu`, virtual `/dev/vgcu` and control `/dev/gcuctl`; the utility `efsmi`;
and the library `libefml.so`.

## Version Compatibility

`gcu-feature-discovery` reads device data through `efsmi` / EFML, so the features
it can discover track the installed Enflame GCU driver. Newer drivers expose more
device attributes. Always match the GFD release to the GCU driver deployed on
your nodes.

## Hardware Support

Supports Enflame GCU devices that provide the Enflame GCU driver together with
`efsmi` and the EFML runtime library (`libefml.so`). The same binary works across
GCU product generations; the exact features discovered depend on the driver.

## User Manual

| Document | Description |
| --- | --- |
| [deployments/README.md](deployments/README.md) | Deployment and image build guide |

## Trouble Shooting

- **Labels are not applied to nodes** — verify the Enflame GCU driver is loaded
  and that `efsmi` runs and `libefml.so` is available on the node; then check the
  DaemonSet (or Job) status and pod logs.
- **`libefml.so` not found** — ensure the Enflame GCU driver / management library
  is installed and on the library search path inside the pod.
- **Cannot see any labels** — list the node labels to confirm the applied keys,
  and check that the GFD pod is scheduled on the GCU node.

## License

`gcu-feature-discovery` is licensed under the **Apache License 2.0**. Copyright
(c) Enflame Technology. All Rights Reserved. See [LICENSE](LICENSE).
