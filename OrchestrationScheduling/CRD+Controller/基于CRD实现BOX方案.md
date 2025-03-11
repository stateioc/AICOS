### 基于CRD实现BOX方案

#### 定义 Box 类型的 CRD 对象. 
需要注意的是，在低版本的 k8s 集群中，CRD schema 无需严格定义，创建/修改 CRD object时，可以设置 spec 字段为任意合法的 yaml 内容均可。
而在高版本 k8s 集群中，CRD 中需通过 openAPIV3Schema 字段，对 CRD 的属性、字段等均做了严格限定，创建时如果出现未定义字段或者类型不正确，创建 CRD object 都会失败。(这里为了描述方便，openAPIV3Schema 只定义了有限的几个字段，后面可以根据需要进行添加调整)

```
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: boxs.cncos.cncos.io
  annotations:
    "api-approved.kubernetes.io": "unapproved, experimental-only; please get an approval from Kubernetes API reviewers if you're trying to develop a CRD in the *.k8s.io or *.kubernetes.io groups"
spec:
  group: box.cncos.cncos.io
  versions:
    - name: v1alpha1
      served: true
      storage: true
      schema:
        # schema used for validation
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                boxName:
                  type: string
                replicas:
                  type: integer
                  minimum: 1
                  maximum: 10
            status:
              type: object
              properties:
                availableReplicas:
                  type: integer
      # subresources for the custom resource
      subresources:
        # enables the status subresource
        status: {}
  names:
    kind: Box
    plural: boxes
  scope: Namespaced
```


#### CRD管理
创建 Box 类型的资源。可以正确 get，edit， update；这一步是确保定义的 CRD 的 schema 无问题。
经过上面两步且正确无误的话，Box crd schema 定义，box 类型对象，都可以正确存储到 etcd 上；

#### 控制器开发
但是仅仅把对象数据保存在 etcd 是没有什么意义的。因此需要自定义 controller，通过监听 Box 资源的新增、修改、删除等操作，针对这些操作做出相应的响应。而对于通过 CRD 实现 box 的需求，我们需要在 controller 中监听 Box 资源变化，进而注册回调函数，在回调函数里调用 k8s kube-api 接口（通过 client-go）实现对 k8s 工作负载资源 （如 POD）的创建，从而在 worker 节点启动 container。大致步骤如下：
- 创建声明 api 对象相关代码，主要包括 doc.go, types.go, register.go 等相关文件，文件中的 groupVersion 声明等信息要和 CRD 中定义保持一致。
- 利用 codegen 代码生成工具生成自定义对象的 deepcopy,client,informer,lister 等代码；
- 编写 controller 代码，监听 CRD 对象变化，如果对象未达到声明状态，则对对象进行处理调谐到声明状态。

#### 部署
- CRD 控制器一般以 deployment 的方式进行部署；
- CRD 控制器调谐对象状态的过程中，涉及到对 kube-api 对资源的操作，因此需要创建 service account，clusterrole，clusterrolebindings 等。授权对资源的操作访问权限；
- 线下controller 开发阶段，可以通过命令行直接kubeconfig 的方式和 k8s 集群进行交互，测试程序的功能是否正常；
