
## gcu-feature-discovery镜像构建

执行当前目录下的 `build-image.sh` 可以自动构建好gcu-feature-discovery 镜像。

## 自定义gcu-feature-discovery镜像名称

build-image.conf 里默认的镜像路径与名称为: 
`artifact.enflame.cn/enflame_docker_images/enflame/gcu-feature-discovery:latest`，如下：

```conf
# Currently supports ubuntu, tlinux, openeuler
OS="ubuntu"

# Currently supports docker, ctr, podman, nerdctl
CLI_NAME="docker"

# The repository name
REPO_NAME="artifact.enflame.cn/enflame_docker_images/enflame"

# The image name
IMAGE_NAME="gcu-feature-discovery"

# The image tag
TAG="latest"

# The namespace used by nerdctl, ctr
NAMESPACE="k8s.io"

```

可以根据自己的需要自定义这个镜像路径与名称。

## 更新与Apply Yaml

gcu-feature-discovery-daemonset.yaml 文件里的默认镜像为：
```
image: artifact.enflame.cn/enflame_docker_images/enflame/gcu-feature-discovery:latest
```

可以根据自己的需要自定义这个镜像路径与名称。然后执行 kubectl apply 应用这个镜像，如下：
```
kubectl apply -f yaml/gcu-feature-discovery-daemonset.yaml
```
或执行脚本：`./deploy.sh` 完成部署，删除可执行 `./delete.sh`。