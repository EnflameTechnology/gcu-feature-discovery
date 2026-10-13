# 燧原 GCU Feature Discovery（gcu-feature-discovery）

> **语言**: [English](README.md) | 中文

## 目录

- [概述](#概述)
- [项目结构](#项目结构)
- [安装](#安装)
  - [前置条件](#前置条件)
  - [从源码构建](#从源码构建)
  - [构建容器镜像](#构建容器镜像)
- [快速开始](#快速开始)
  - [以 DaemonSet 方式部署](#以-daemonset-方式部署)
  - [以 Job 方式运行一次](#以-job-方式运行一次)
  - [查看节点标签](#查看节点标签)
- [核心功能](#核心功能)
- [节点标签](#节点标签)
- [版本兼容性](#版本兼容性)
- [硬件支持](#硬件支持)
- [用户手册](#用户手册)
- [故障排查](#故障排查)
- [许可证](#许可证)

## 概述

`gcu-feature-discovery`（GFD）为 Kubernetes 节点打上其上燧原 GCU（General
Compute Unit，通用计算单元）设备的属性与特性标签。它与 NVIDIA GPU Feature
Discovery 类似，构建于
[node-feature-discovery](https://github.com/kubernetes-sigs/node-feature-discovery)
模式之上。它所发布的节点标签可支撑基于 `nodeSelector` 与节点亲和性的 GCU 感知调度。

GFD 通过 `efsmi` 与 EFML 库（`libefml.so`，即 [go-eflib](../go-eflib) 绑定）检查每个
节点上的 GCU 设备，并在 `enflame.com` 域下发布描述所发现硬件的节点标签。

本项目提供：

- **自动节点打标** —— 发现 GCU 型号、数量、驱动版本、显存等信息，并将其作为 Kubernetes 节点标签应用。
- **EFML + efsmi 后端** —— 通过 `efsmi` 与 EFML 管理库（[go-eflib](../go-eflib)）读取设备数据。
- **Kubernetes 原生** —— 既可作为 DaemonSet 持续打标，也可作为一次性 Job 运行。
- **GCU 感知调度** —— 所发布的标签可配合 `nodeSelector` / 亲和性引导工作负载调度。

## 项目结构

```
gcu-feature-discovery/
├── cmd/                       # 入口（main 包）
├── internal/                  # 实现（包含内嵌的 eflib 包）
├── config/                    # 运行时配置
├── deployments/               # 部署脚本与清单
│   ├── deploy.sh              # 部署脚本
│   ├── delete.sh              # 卸载脚本
│   ├── dockerfiles/           # 镜像构建文件
│   └── yaml/                  # gcu-feature-discovery-daemonset.yaml
│                              # gcu-feature-discovery-job.yaml.template
├── go.mod                     # 模块 enflame/gcu-feature-discovery
├── Makefile                   # 构建 gcu-feature-discovery 二进制
├── versions.mk                # 版本定义
├── topscloud.json             # 内置设备/资源描述文件
└── build.sh                   # 构建 + 打包入口
```

## 安装

### 前置条件

- 一个 Kubernetes 集群。
- 装有燧原 GCU 设备且已安装燧原 GCU 驱动的节点。
- 节点上具备 `efsmi` 与 EFML 运行时库（`libefml.so`）。
- 从源码构建需 [Go](https://go.dev/) 1.22+ 以及 C 工具链（构建使用 cgo，因为它通过 [go-eflib](../go-eflib) 访问 EFML）。
- 构建容器镜像需 Docker。

### 从源码构建

```bash
# 构建 gcu-feature-discovery 二进制，并在 dist/ 下生成发布包
./build.sh all

# 清理构建产物
./build.sh clean
```

`./build.sh all` 会执行 `make` 构建 `gcu-feature-discovery` 二进制，并将其与
`deployments/`、`config/` 一起打包到 `dist/` 下。

### 构建容器镜像

镜像构建文件位于 `deployments/dockerfiles/`。使用所选的基础镜像 / 镜像仓库构建镜像，
并将其加载或推送到集群可拉取的位置。

## 快速开始

### 以 DaemonSet 方式部署

GFD 通常以 DaemonSet 方式运行，从而对每个 GCU 节点持续打标：

```bash
kubectl apply -f deployments/yaml/gcu-feature-discovery-daemonset.yaml
```

### 以 Job 方式运行一次

项目也提供了一次性 Job 模板，用于执行单次打标：

```bash
# 填写模板（镜像、目标节点等）后再应用
kubectl apply -f deployments/yaml/gcu-feature-discovery-job.yaml.template
```

### 查看节点标签

查看 GFD 应用到节点上的标签：

```bash
kubectl get node <node> -o jsonpath='{.metadata.labels}' | tr ',' '\n' | grep enflame.com
```

## 核心功能

| 功能 | 说明 |
| --- | --- |
| 节点打标 | 在 `enflame.com` 域下将燧原 GCU 特性发布为 Kubernetes 节点标签 |
| 特性发现 | 通过 EFML / `efsmi` 发现 GCU 型号、数量、驱动版本与显存 |
| DaemonSet 模式 | 持续运行，随节点变化保持标签同步 |
| Job 模式 | 提供一次性 Job 模板执行单次打标 |
| GCU 感知调度 | 标签支撑基于 `nodeSelector` / 亲和性的 GCU 工作负载调度 |
| EFML + efsmi 后端 | 通过 `efsmi` 与 EFML 库（[go-eflib](../go-eflib)）读取设备数据 |

## 节点标签

GFD 在 `enflame.com` 域下发布描述节点上 GCU 设备的节点标签 —— 例如 GCU 型号、设备数量、
驱动版本与显存。具体的标签键取决于所发现的硬件与驱动，因此请按上文方式列出节点标签，
查看实际应用到节点上的标签键。

相关的设备与资源标识（来自内置的 `topscloud.json`）包括：资源 `enflame.com/gcu`
与 `enflame.com/vgcu`；设备节点 `/dev/gcu`、虚拟 `/dev/vgcu` 与控制节点 `/dev/gcuctl`；
工具 `efsmi`；以及库 `libefml.so`。

## 版本兼容性

`gcu-feature-discovery` 通过 `efsmi` / EFML 读取设备数据，因此其可发现的特性会随所安装的
燧原 GCU 驱动而变化。较新的驱动会暴露更多设备属性。请始终使 GFD 版本与节点上部署的 GCU
驱动相匹配。

## 硬件支持

支持提供燧原 GCU 驱动以及 `efsmi` 与 EFML 运行时库（`libefml.so`）的燧原 GCU 设备。同一份
二进制可跨 GCU 产品代际工作；可发现的具体特性取决于驱动。

## 用户手册

| 文档 | 说明 |
| --- | --- |
| [deployments/README.md](deployments/README.md) | 部署与镜像构建指南 |

## 故障排查

- **标签未应用到节点** —— 确认燧原 GCU 驱动已加载，且节点上 `efsmi` 可运行、`libefml.so`
  可用；随后检查 DaemonSet（或 Job）状态与 Pod 日志。
- **找不到 `libefml.so`** —— 确保燧原 GCU 驱动 / 管理库已安装，且位于 Pod 内的库搜索路径上。
- **看不到任何标签** —— 列出节点标签确认已应用的标签键，并检查 GFD Pod
  是否已调度到 GCU 节点上。

## 许可证

`gcu-feature-discovery` 采用 **Apache License 2.0** 许可证。版权所有 (c) Enflame
Technology，保留所有权利。详见 [LICENSE](LICENSE)。
