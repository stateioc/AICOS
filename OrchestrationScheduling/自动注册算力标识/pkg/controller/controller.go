package controller

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"register-power-resources/pkg/apis"
	"register-power-resources/pkg/client"
)

const (
	Register            = "cncos.org/register"
	City                = "cncos.org/city"
	CompanyType         = "cncos.org/company-type"
	Company             = "cncos.org/company"
	ResourceType        = "cncos.org/resource-type"
	ResourceAZ          = "cncos.org/resource-az"
	ServiceType         = "cncos.org/service-type"
	NetworkType         = "cncos.org/network-type"
	CPUChipType         = "cncos.org/cpu-chip-type"
	CPUChipModel        = "cncos.org/cpu-chip-model"
	CPUChipNumber       = "cncos.org/cpu-chip-number"
	GPUChipType         = "cncos.org/gpu-chip-type"
	GPUChipModel        = "cncos.org/gpu-chip-model"
	GPUChipNumber       = "cncos.org/gpu-chip-number"
	NetworkBandwidth    = "cncos.org/network-bandwidth"
	StorageCapacity     = "cncos.org/disk-capacity"
	NetworkAddress      = "cncos.org/network-address"
	CPUCapacity         = "cncos.org/cpu-capacity"
	MemoryCapacity      = "cncos.org/memory-capacity"
	CPUComputeCapacity  = "cncos.org/cpu-compute-capacity"
	GPUComputeCapacity  = "cncos.org/gpu-compute-capacity"
	CPUPowerConsumption = "cncos.org/cpu-power-consumption"
	GPUPowerConsumption = "cncos.org/gpu-power-consumption"
)

type NodeResourceController struct {
	clientset *kubernetes.Clientset
}

func NewNodeResourceController(clientset *kubernetes.Clientset) *NodeResourceController {
	return &NodeResourceController{clientset: clientset}
}

func (c *NodeResourceController) Run() {
	// 定期同步节点资源信息
	ticker := time.NewTicker(60 * time.Second)
	for range ticker.C {
		c.syncNodeResources()
	}
}

func (c *NodeResourceController) syncNodeResources() {
	//fmt.Println("[Debug]In syncNodeResources")
	// 获取节点列表
	nodes, err := c.clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	// 遍历节点并收集资源信息
	var registerDataMap []string
	var unRegisterDataMap []string
	var info apis.NodeResourceInfo

	for _, node := range nodes.Items {
		fmt.Println("Node [" + node.Name + "]: ")

		if len(node.Labels[City]) != 4 || len(node.Labels[CompanyType]) != 2 ||
			len(node.Labels[Company]) != 5 || len(node.Labels[ResourceType]) != 3 ||
			len(node.Labels[ResourceAZ]) != 3 || len(node.Labels[ServiceType]) != 14 {
			fmt.Println("[Info]Static Info not enough")
			continue
		}

		// 创建 NodeResourceInfo 结构
		info = apis.NodeResourceInfo{
			City:         fmt.Sprintf("%0*s", 4, node.Labels[City]),
			CompanyType:  fmt.Sprintf("%0*s", 2, node.Labels[CompanyType]),
			Company:      fmt.Sprintf("%0*s", 5, node.Labels[Company]),
			ResourceType: fmt.Sprintf("%0*s", 3, node.Labels[ResourceType]),
			ResourceAZ:   fmt.Sprintf("%0*s", 3, node.Labels[ResourceAZ]),
			ServiceType:  fmt.Sprintf("%0*s", 14, node.Labels[ServiceType]),

			// 动态采集，包含计算量、存储、网络带宽、功耗
			StorageCapacity:   fmt.Sprintf("S%0*s", 7, node.Labels[StorageCapacity]),
			NetworkBandSwitch: fmt.Sprintf("N%0*s", 6, node.Labels[NetworkBandwidth]),

			// 新增需求， 网络、芯片的属性通过动态探测的方式完成，这部分解耦通过 daemon set 来上报处理
			NetworkType:          fmt.Sprintf("%0*s", 2, node.Labels[NetworkType]),
			PowerResourceAddress: fmt.Sprintf("00" + fmt.Sprintf("%0*s", 32, node.Labels[NetworkAddress])),
		}

		//CPU 数据组装
		chipNum, err := strconv.ParseInt(node.Labels[CPUChipNumber], 10, 64)
		if err != nil {
			fmt.Println(chipNum, err)
			continue
		}

		var i int64
		for i = 1; i <= chipNum; i++ {
			info.ChipUniqNumber = fmt.Sprintf("%05d", i)
			info.ChipType = fmt.Sprintf("%0*s", 5, node.Labels[CPUChipType])
			info.ChipModel = fmt.Sprintf("%0*s", 8, node.Labels[CPUChipModel])
			info.ComputeCapacity = fmt.Sprintf("F%0*s", 4, node.Labels[CPUComputeCapacity])
			info.PowerConsumption = fmt.Sprintf("P%0*s", 5, node.Labels[CPUPowerConsumption])

			// 城市>-行业>-企业>-资源类型>-数据中心>-服务类型>-计算、存储、网络及功耗>-网络类型>-算力互联网地址>-芯片类型>-芯片型号>-芯片唯一编号。
			registerData := fmt.Sprintf("%s%s%s%s%s%s%s%s%s%s%s%s%s%s%s",
				info.City, info.CompanyType, info.Company, info.ResourceType, info.ResourceAZ, info.ServiceType,
				info.ComputeCapacity, info.StorageCapacity, info.NetworkBandSwitch, info.PowerConsumption,
				info.NetworkType, info.PowerResourceAddress, info.ChipType, info.ChipModel, info.ChipUniqNumber)

			if len(node.Labels[Register]) == 0 || node.Labels[Register] != "true" {
				fmt.Printf("[Info]Node %s need not register and unregister it first. If you want to register it, "+
					"label it with cncos.org/register=true\n", node.Name)
				unRegisterDataMap = append(unRegisterDataMap, registerData)
			} else {
				// 将节点资源信息发送到 Web 服务器
				registerDataMap = append(registerDataMap, registerData)
			}
		}

		//GPU 数据组装
		chipNum, err = strconv.ParseInt(node.Labels[GPUChipNumber], 10, 64)
		if err != nil {
			fmt.Println(chipNum, err)
			continue
		}

		for i = 1; i <= chipNum; i++ {
			info.ChipType = fmt.Sprintf("%0*s", 5, node.Labels[GPUChipType])
			info.ChipModel = fmt.Sprintf("%0*s", 8, node.Labels[GPUChipModel])
			info.ChipUniqNumber = fmt.Sprintf("%05d", i)
			info.ComputeCapacity = fmt.Sprintf("F%0*s", 4, node.Labels[GPUComputeCapacity])
			info.PowerConsumption = fmt.Sprintf("P%0*s", 5, node.Labels[GPUPowerConsumption])

			// 城市>-行业>-企业>-资源类型>-数据中心>-服务类型>-计算、存储、网络及功耗>-网络类型>-算力互联网地址>-芯片类型>-芯片型号>-芯片唯一编号。
			registerData := fmt.Sprintf("%s%s%s%s%s%s%s%s%s%s%s%s%s%s%s",
				info.City, info.CompanyType, info.Company, info.ResourceType, info.ResourceAZ, info.ServiceType,
				info.ComputeCapacity, info.StorageCapacity, info.NetworkBandSwitch, info.PowerConsumption,
				info.NetworkType, info.PowerResourceAddress, info.ChipType, info.ChipModel, info.ChipUniqNumber)

			if len(node.Labels[Register]) == 0 || node.Labels[Register] != "true" {
				fmt.Printf("[Info]Node %s need not register and unregister it first. If you want to register it, "+
					"label it with cncos.org/register=true\n", node.Name)
				unRegisterDataMap = append(unRegisterDataMap, registerData)
			} else {
				// 将节点资源信息发送到 Web 服务器
				registerDataMap = append(registerDataMap, registerData)
			}
		}
	}

	// 将节点资源信息发送到 Web 服务器
	fmt.Println(unRegisterDataMap)
	fmt.Println(registerDataMap)

	unRegisterNodeResourcesToServer(unRegisterDataMap)
	registerNodeResourcesToServer(registerDataMap)
}

func registerNodeResourcesToServer(registerData []string) {
	// 在这里实现将节点资源信息发送到 Web 服务器的逻辑，
	if os.Args[1] == "register" {
		err := client.LoadConfig()
		if err != nil {
			fmt.Println("LoadConfig error")
			return
		}
		client.RegisterResource(registerData)
	} else {
		fmt.Println(registerData)
	}
}

func unRegisterNodeResourcesToServer(unRegisterDataMap []string) {
	err := client.LoadConfig()
	if err != nil {
		fmt.Println("server can't connect because of config is invalid")
	} else {
		for _, registerData := range unRegisterDataMap {
			nodeInfo := apis.ParseResourceInfo(registerData)
			client.UnregisterResource(nodeInfo.ID)
		}
	}
}
