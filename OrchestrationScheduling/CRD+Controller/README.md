# Box-Controller


# 简介

本代码库实现了基于 CRD 方案的 box, 具有如下特点和功能.
- 基于 k8s v1.28 版本;
- 和 k8s 原生 pod 完全兼容的 Box 资源(数据结构完全兼容), 并支持原生 pod 的所有功能(功能兼容)如: 挂载数据卷、加载环境变量、设置调度亲和性、容忍度、联网策略等;
- 类比 k8s 原生的 Deployment, 基于 Box 实现的 BoxDeployment 高阶资源 (数据结构完全兼容), 目前支持创建、更新、删除、扩缩容副本数等原生 Deployment 功能;
- 实现了基于Box的有状态应用BoxStatefulSet，兼容原生有状态应用StatefulSet的数据结构，同时支持创建、滚动升级、创建PVC、更新、删除、扩缩容等功能。
- 实现了 Box 和 BoxDeployment 控制器, 每个控制器可以独立运行, 也可以一起运行;


# 部署

## 安装 crd 资源

安装 crd 资源并检查是否安装成功

```
git clone git@182.42.135.89:cncos/open-cnc.git

cd open-cnc/重构kubernetes/CRD+Controller
kubectl apply -f crd/box-crd.yaml
kubectl apply -f crd/boxdeployment-crd.yaml
kubectl apply -f crd/cncos.io_boxstatefulsets.yaml
```

```
# kubectl get crd
NAME                      CREATED AT
boxdeployments.cncos.io   2023-11-20T02:49:51Z
boxes.cncos.io            2023-11-14T07:55:50Z
boxstatefulsets.cncos.io  2023-12-10T03:26:22Z
```

安装 crd 资源成功后, 可以启动 crd 资源控制器, 并创建对应的Box、BoxDeployment、BoxStatefulSet资源, 控制器会监听对应事件并进行响应,如果对象未达到声明状态, 则将对象调谐到声明状态。

## 控制器启动参数介绍

- `--kubeconfig` 集群 kubeconfig 配置文件路径, 仅当控制器在集群外运行时, 需要该参数
- `-resync-period` resync 拉取资源周期时间
- `-v` 控制器显示的日志曾经
- `-worker` 每个控制器启动几个实际工作的 goroutine
- `-box-controller-enable` 开启 box 资源控制器
- `-boxdeployment-controller-enable` 开启 boxdeploymnent 资源控制器
- `-boxstatefulset-controller-enable` 开启BoxStatefulSet控制器
- `-kube-config-qps` 设置k8s请求客户端的最大QPS，默认设置为100
- `-kube-config-burst` 设置k8s请求客户端的最大爆发，默认设置为200

crd 控制器可以集群内和集群外两种方式运行, 下面分开介绍，推荐使用集群内运行.

## 集群外部署

控制器在集群外运行需要指定 kubeconfig 参数, 控制器通过该参数连接 k8s 集群并进行交互

```
go build 
# Box、BoxDeployment、BoxStatefulSet控制器在一个进程中共同运行
./box-controller --kubeconfig=/root/.kube/config -resync-period=0 -v=4 -worker=1 -box-controller-enable=true -boxdeployment-controller-enable=false -boxstatefulset-controller-enable=false 2>&1

# 也可以开两个终端, box 和 boxDeployment 控制器分开独立运行
./box-controller --kubeconfig=/root/.kube/config -resync-period=0 -v=4 -worker=1 -box-controller-enable=true
./box-controller --kubeconfig=/root/.kube/config -resync-period=0 -v=4 -worker=1 -boxdeployment-controller-enable=true
./box-controller --kubeconfig=/root/.kube/config -resync-period=0 -v=4 -worker=1 -boxstatefulset-controller-enable=true
```

## 集群内部署

### Helm包部署
```
cd open-cnc/重构kubernetes/CRD+Controller/charts
# 当你想修改部署包以适配你的集群时，可以这行代码前编辑values.yaml
helm package box-controller/
# 当你想修改部署包以适配你的集群时，例如修改镜像可以使用--set image=“镜像地址” 
helm install box-controller -n boxns ./box-controller-1.0.0.tgz -i --create-namespace
```


### 部署脚本部署
CRD 控制器调谐对象状态的过程中，涉及通过 kube-api 对资源的操作，因此需要创建 service account，clusterrole，clusterrolebindings 等, 授权对对应资源的操作访问权限.

```
apiVersion: v1
kind: Namespace
metadata:
  name: boxns
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: boxsa
  namespace: boxns
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: box-clusterrole
rules:
- apiGroups:
  - ""
  resources:
  - pods
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - ""
  resources:
  - events
  verbs:
  - create
  - patch
- apiGroups:
  # 这里不知道为什么不能写成 cncos.io, 集群内运行时会提示没有 box/boxdeployment list 权限
  # - 'cncos.io'
  - '*'
  resources:
  - boxes
  - boxes/status
  - boxdeployments
  - boxdeployments/status
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: boxsa-clusterrolebinding
subjects:
- kind: ServiceAccount
  name: boxsa
  namespace: boxns
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: box-clusterrole
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: box-controller
  namespace: boxns
spec:
  progressDeadlineSeconds: 600
  replicas: 1
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      app: box-controller
  strategy:
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 1
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: box-controller
    spec:
      containers:
      - args:
        - ./box-controller
        - -resync-period=0
        - -v=4
        - -worker=1
        - -box-controller-enable=true
        - -boxdeployment-controller-enable=true
        image: harbor.ctyuncdn.cn/devops/box-controller:v0.6
        imagePullPolicy: IfNotPresent
        name: box-controller
        resources: {}
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      serviceAccount: boxsa
      serviceAccountName: boxsa
      terminationGracePeriodSeconds: 30

```

kubectl apply -f crd/deploy.yaml

然后通过 Deployment 部署控制器即可。

## 创建Box相关资源

由于box、boxdeployment、boxstatefulset 和 k8s 原生 pod、deployment、statefulset 完全兼容, 创建资源时只需要更改编排 kind 和 apiVersion 字段即可:
- apiVersion 字段改成 *cncos.io/v1alpha1*
- kind 字段改成 *Box* 或者 *BoxDeployment* 或者 *BoxStatefulSet* 即可


# 基于Box的无状态应用BoxDeployment的演示

首先在 k8s v1.28 版本中安装 nginx ingress controller, 并创建如下 boxdeployment, svc, ingress 资源

```
apiVersion: cncos.io/v1alpha1
kind: BoxDeployment
metadata:
  annotations:
    componentName: container-middleground
  labels:
    cn.ctcdn.walrus.eks: deployment
  name: new-boxdp1
  namespace: default
spec:
  progressDeadlineSeconds: 600
  replicas: 2
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      run: hello-app
  strategy:
    type: Recreate
  template:
    metadata:
      annotations:
        componentName: container-middleground
      labels:
        run: hello-app
    spec:
      affinity:
        nodeAffinity: {}
      containers:
      - image: hello-app:v1.0
        imagePullPolicy: IfNotPresent
        lifecycle: {}
        name: container01
        resources: {}
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsConfig: {}
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 30
---
apiVersion: v1
kind: Service
metadata:
  name: new-boxdp1-svc
spec:
  selector:
    run: hello-app
  ports:
  - name: name-of-service-port
    protocol: TCP
    port: 80
    #targetPort: http-web-svc
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ing2
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  rules:
  - host: cncos-test.io
    http:
      paths:
      - backend:
          service:
            name: new-boxdp1-svc
            port:
              number: 80
        path: /
        pathType: ImplementationSpecific
```

编辑 /etc/hosts 增加如下内容：

```
192.168.49.2 cncos-test.io
```

查看新创建的 ingress 并且尝试通过 ingress + svc 访问相关资源
```
# kubectl get ing
NAME   CLASS   HOSTS                  ADDRESS        PORTS   AGE
ing1   nginx   jupiter-dev.ctcdn.cn   192.168.49.2   80      38h
ing2   nginx   cncos-test.io          192.168.49.2   80      38h

# curl cncos-test.io/abc
Hello, world!
Version: 1.0.0
Hostname: new-boxdp1-j7ztn8
```

可以看到使用 ingress + svc 使用访问 boxdeployment 创建的工作负载资源

# 基于Box的有状态应用BoxStatefulSet的演示


## 创建基于Box的有状态应用BoxStatefulSet

创建有状态应用BoxStatefulSet
```yaml
apiVersion: cncos.io/v1alpha1
kind: BoxStatefulSet
metadata:
  name: boxsts-test
spec:
  podManagementPolicy: OrderedReady
  replicas: 2
  selector:
    matchLabels:
      app: nginx
  serviceName: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - image: lowyard/nginx-slim:0.8
        name: nginx
        ports:
        - containerPort: 80
          name: web
          protocol: TCP
        volumeMounts:
        - mountPath: /usr/share/nginx/html
          name: www
  volumeClaimTemplates:
  - metadata:
      name: www
    spec:
      accessModes:
      - ReadWriteOnce
      resources:
        requests:
          storage: 1Gi
      storageClassName: caas-lvm
```
查看有状态应用BoxStatefulSet以及相关资源的运行状态
```
# kubectl get boxstatefulset,box,pod,pvc
NAME                                  AGE
boxstatefulset.cncos.io/boxsts-test   40s

NAME                         AGE
box.cncos.io/boxsts-test-0   39s
box.cncos.io/boxsts-test-1   10s

NAME                READY   STATUS    RESTARTS   AGE
pod/boxsts-test-0   1/1     Running   0          38s
pod/boxsts-test-1   1/1     Running   0          10s

NAME                                      STATUS   VOLUME                                      CAPACITY   ACCESS MODES   STORAGECLASS   AGE
persistentvolumeclaim/www-boxsts-test-0   Bound    disk-0d15ed9d-a62a-46f8-a426-e52b8a6160ad   1Gi        RWO            caas-lvm       37s
persistentvolumeclaim/www-boxsts-test-1   Bound    disk-e46ac8e2-c513-42d5-b523-1421c2f839bc   1Gi        RWO            caas-lvm       10s
```

## 有状态应用BoxStatefulSet的扩缩容
可以直接使用kubectl scale针对BoxStatefulSet进行扩缩容
```
# kubectl scale boxstatefulset -n chenkun boxsts-test --replicas=3
boxstatefulset.cncos.io/boxsts-test scaled
```

查看BoxStatefulSet扩容后相关资源的运行状态
```
# kubectl get boxstatefulset,box,pod,pvc -n chenkun
NAME                                  AGE
boxstatefulset.cncos.io/boxsts-test   11m

NAME                         AGE
box.cncos.io/boxsts-test-0   11m
box.cncos.io/boxsts-test-1   11m
box.cncos.io/boxsts-test-2   4s

NAME                READY   STATUS    RESTARTS   AGE
pod/boxsts-test-0   1/1     Running   0          11m
pod/boxsts-test-1   1/1     Running   0          11m
pod/boxsts-test-2   1/1     Running   0          4s

NAME                                      STATUS   VOLUME                                      CAPACITY   ACCESS MODES   STORAGECLASS   AGE
persistentvolumeclaim/www-boxsts-test-0   Bound    disk-0d15ed9d-a62a-46f8-a426-e52b8a6160ad   1Gi        RWO            caas-lvm       11m
persistentvolumeclaim/www-boxsts-test-1   Bound    disk-e46ac8e2-c513-42d5-b523-1421c2f839bc   1Gi        RWO            caas-lvm       11m
persistentvolumeclaim/www-boxsts-test-2   Bound    disk-a841341f-4c2b-49c9-a0cd-68c6f5ae6c95   1Gi        RWO            caas-lvm       4s
```

## 有状态应用BoxStatefulSet的数据持久化

Nginx Web服务器默认会加载位于`/usr/share/nginx/html/index.html`文件。
BoxStatefulSet的spec字段中的`volumeMounts`字段保证了`/usr/share/nginx/html`文件夹由一个`PersistentVolume`卷支持。
将 Pod 的主机名写入它们的`index.html`文件并验证 Nginx Web服务器使用该主机名提供服务：

```
# for i in 0 1 2; do kubectl exec "boxsts-test-$i" -- sh -c 'echo "$(hostname)" > /usr/share/nginx/html/index.html'; done

# for i in 0 1 2; do kubectl exec -i -t "boxsts-test-$i" -- curl http://localhost/; done
boxsts-test-0
boxsts-test-1
boxsts-test-2
```

此时将BoxStatefulSet的副本数缩容为0再扩容为3，查看相关资源的状态，发现pvc其实并没有跟随Pod删除而删除。
```
# kubectl get boxstatefulset,box,pod,pvc -n chenkun
NAME                                  AGE
boxstatefulset.cncos.io/boxsts-test   20m

NAME                         AGE
box.cncos.io/boxsts-test-0   11s
box.cncos.io/boxsts-test-1   8s
box.cncos.io/boxsts-test-2   5s

NAME                READY   STATUS    RESTARTS   AGE
pod/boxsts-test-0   1/1     Running   0          11s
pod/boxsts-test-1   1/1     Running   0          8s
pod/boxsts-test-2   1/1     Running   0          5s

NAME                                      STATUS   VOLUME                                      CAPACITY   ACCESS MODES   STORAGECLASS   AGE
persistentvolumeclaim/www-boxsts-test-0   Bound    disk-0d15ed9d-a62a-46f8-a426-e52b8a6160ad   1Gi        RWO            caas-lvm       20m
persistentvolumeclaim/www-boxsts-test-1   Bound    disk-e46ac8e2-c513-42d5-b523-1421c2f839bc   1Gi        RWO            caas-lvm       20m
persistentvolumeclaim/www-boxsts-test-2   Bound    disk-a841341f-4c2b-49c9-a0cd-68c6f5ae6c95   1Gi        RWO            caas-lvm       9m
```

再次进入到Pod容器进行请求，发现返回的数据没有变化
```
# for i in 0 1 2; do kubectl exec -i -t "boxsts-test-$i" -- curl http://localhost/; done
boxsts-test-0
boxsts-test-1
boxsts-test-2
```