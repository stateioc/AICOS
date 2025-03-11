### 重构POD实现BOX方案预研

#### 环境准备
1、纳管windwos为工作节点：配置windows为k8s的一个node节点，进行组件的调试，因为goland跑在windows,windows可以单向联通开发机;
2、kube-apiserver调试：本地goland 已经配置参数，成功运行起kube-apiserver；
3、kubelet调试： windows的goland无法直接运行kubelet,需要后续在windows配置containred环境 来调试kubelet代码；

#### 涉及组件情况
1、先实现最简单的box 创建流程 ,先配置nodeName
    1）不涉及控制器，所以不涉及kube-controller组件
    2）如果给box 设置nodeName字段，也不涉及kube-scheduler组件
    3)  如果先不考虑网络，只涉及kube-apiserver、kubelet、etcd三个组件的关系
2、再不配置nodeName，实现box的创建
    1）涉及到了kube-shcduler组件参与调度
3、再考虑网络，实现box网络，涉及kube-proxy组件等

#### 代码开发流程
1、定义box资源
    1） 根据GVR，box类似pod为无资源组资源 /<version>来定义
    2）定义内部版本，外部版本，转化函数在scheme中初始化
    3）box资源代码定义，即在k8s不同目录中添加box代码信息
2、定义 box相关函数：get ,delete 操作
3、box资源与runtime.object类型的相互转换（getobjkind,deepcopyobject）
4、根据box参数需求,注册box资源为k8s内置资源（修改kube-apiserver组件：k8s通过import和init机制触发，注册到scheme资源注册表中）
    1）初始化Scheme资源注册表
    2）注册k8s支持的资源
5、box内嵌到Informer机制中；（reflector,deltafifo,indexer）
6、在kubelet组件中，box资源对接（CRI 实现容器镜像管理,CNI实现网络配置操作）
7、box 兼容kube-proxy组件实现网络代理
8、box get，create 操作，kubectl 命令行的开发 
