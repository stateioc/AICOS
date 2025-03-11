# 部署文档

## 算力标识注册
算力标识注册共分为3部分：
- 算力资源动态扫描服务
- 算力资源展示服务
- 算力资源上报服务

## 解压安装包
```shell
## 解压
tar -zxvf cncos-resource-register.tar.gz

## 输出结果
./cncos-resource-register
```


### 重新进入安装目录
```shell
cd ./cncos-resource-register
```
### 服务部署

#### 创建cncos命名空间
```shell
kubectl create ns cncos-system
```

#### 动态资源扫描服务部署
```shell
cd ./node-reporter
kubectl apply -f ./rbac.yaml -n cncos-system
kubectl apply -f ./daemonset.yaml -n cncos-system
kubectl apply -f ./configmap.yaml -n cncos-system

通过以下命令查看启动情况
kubectl get po -n cncos-system -o wide | grep node-reporter

通过以下命令查看动态扫描情况
kubectl get node --show-labels | grep cncos
```
#### 资源展示服务部署
```shell
cd -
cd ./server
kubectl apply -f ./deployment.yaml -n cncos-system

通过以下命令查看启动情况
kubectl get po -n cncos-system -o wide | grep server
```

#### 资源上报服务部署
```shell
cd -

## 填写上报集群的kubeconfig信息至configmap中
cd cluster-lists/
kubectl create configmap cluster-kubeconfig --namespace=kube-system --from-file=kubeconfig=$HOME/.kube/config --dry-run=client -o yaml | sed '/^metadata:/a\ \ labels:\n\ \ \ \ cluster_resources_register_kubeconfig: "true"' > cluster-kubeconfig.yaml
kubectl apply -f cluster-kubeconfig.yaml -n kube-system

## 部署资源上报服务
cd -
cd ./controller

## 替换master的IP
KUBE_IP=$(kubectl get endpoints kubernetes -n default -o jsonpath='{.subsets[0].addresses[0].ip}') && sed -i "s/0.0.0.0/$KUBE_IP/" deployment.yaml

kubectl apply -f ./rbac.yaml -n cncos-system
kubectl apply -f ./configmap.yaml -n cncos-system
kubectl apply -f ./deployment.yaml -n cncos-system

## 通过以下命令查看启动情况
kubectl get po -n cncos-system -o wide | grep resource-controller
```

#### 资源上报
```shell
## 节点名称，每个节点都需执行


for node in $(kubectl get node --no-headers | awk '{print $1}'); do
	echo ${node}

	export NODE_NAME=${node}
	## len=4
	export RESOURCE_CITY=xxxx
	## len=2
	export COMPANY_TYPE=xx 
	## len=5
	export COMPANY=xxxxx
	## len=3
	export RESOURCE_TYPE=xxx
	## len=3
	export RESOURCE_AZ=xxx
	## len=14
	export SERVICE_TYPE=xxxxxxxxxxxxxx

	kubectl label node ${NODE_NAME} cncos.org/register=true cncos.org/city=${RESOURCE_CITY} cncos.org/company-type=${COMPANY_TYPE} cncos.org/company=${COMPANY} cncos.org/resource-type=${RESOURCE_TYPE} cncos.org/resource-az=${RESOURCE_AZ} cncos.org/service-type=${SERVICE_TYPE} --overwrite
done

```

#### 查看资源上报情况
```shell
export SERVER_POD=$(kubectl get po -n cncos-system -o wide --no-headers | grep resource-server | head -n 1 | awk '{print $1}')
kubectl exec -it -n cncos-system ${SERVER_POD} -- bash
cd /root/client/
./client list
```

