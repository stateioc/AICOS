set -o errexit
set -o nounset
set -o pipefail
set -x

cd ../
go build && ./box-controller --kubeconfig=/root/.kube/config -resync-period=0 -v=4 -worker=1 -box-controller-enable=true -boxdeployment-controller-enable=false 2>&1



cd /home/axeadmin/cncos-box-controller
# 以下两个命令均可
# 仅仅更新 openAPIV3Schema
controller-gen schemapatch:manifests=./crd output:dir=./crd paths=./pkg/apis/box/...
# 生成整个 crd 文件
controller-gen crd:crdVersions=v1 output:dir=./crd paths=./pkg/apis/box/...
controller-gen crd:crdVersions=v1,maxDescLen=0 output:dir=./crd paths=./pkg/apis/box/...


# crd 定义结构体很多, 生成注释版本的 crd 然后 apply 时可能提示错误:
# The CustomResourceDefinition "boxes.cncos.io" is invalid: metadata.annotations: Too long: must have at most 262144 bytes
# 因此需要使用下面的命令生成没有注释版本的 crd 文件
controller-gen schemapatch:manifests=./crd,maxDescLen=0 output:dir=./crd paths=./pkg/apis/box/...


# 1. 先定义 crd golang struct 结构体文件
# 2. 根据结构体定义或自动生成 crd yaml 文件(通过 controller-gen ); 并 apply；
# 3. 创建 crd 对象，并尝试 apply，确保没有问题；
# 4. 利用 code-gen 生成 lister, informer, clientset 等代码;
# 5. 再编写或者更新自定义的控制器逻辑;
# 可以试一试如下命令加不加 generateEmbeddedObjectMeta 的区别
controller-gen schemapatch:manifests=./crd,maxDescLen=0 output:dir=./crd paths=./pkg/apis/boxdeployment/...
controller-gen schemapatch:manifests=./crd,maxDescLen=0,generateEmbeddedObjectMeta=true output:dir=./crd paths=./pkg/apis/boxdeployment/... 
