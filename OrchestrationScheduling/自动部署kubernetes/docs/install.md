# 部署文档（概括版）

## 解压安装包
```shell
## 解压
tar -zxvf cncos-ops.tar.gz

## 输出结果
./cncos-ops
```

## 部署操作系统

### 预置检查
```shell
cd ./cncos-ops

## 【预置检查】您应当注意检查结果为 `[FATAL]` 项目，并在准备环境的过程中进行调整。
./cncos-ops --check all
```

### 配置系统变量
```shell
## 请在第一台主机上执行以下命令
### 设置操作系统版本与基础环境变量
set -a
K8S_VER="1.24.15"
CRI_TYPE="containerd"
BK_PUBLIC_REPO=hub.bktencent.com
INSECURE_REGISTRY=""
set +a
```

### 安装操作系统
#### 部署第一个Master节点
```shell
### 安装master节点
./cncos-ops -i master

执行完成后，将输出类似以下内容，请将内容拷贝出来，备用：
======================
# Expand Control Plane, run the following command on new machine
set -a
CLUSTER_ENV=xxxx
MASTER_JOIN_CMD=xxxx
set +a
./cncos-ops -i master
======================
# Expand Worker Plane, run the following command on new machine
set -a
CLUSTER_ENV=xxxx
JOIN_CMD=xxxx
set +a
```
#### 部署其他Master节点或Node节点
```shell
## 请根据在master节点上的输出内容，在新的节点上执行类似的命令(可在第一台主机上执行./cncos-ops --render joincmd 重新查看)

## 比如再加入一个master节点：
set -a
CLUSTER_ENV=xxxx
JOIN_CMD=xxxx
set +a
./cncos-ops -i master

## 加入一个node节点：
set -a
CLUSTER_ENV=xxxx
JOIN_CMD=xxxx
set +a
./cncos-ops -i node
```

### 【检查】
```shell
执行以下命令，将看到节点已安装完成
kubectl get node -o wide 
```
