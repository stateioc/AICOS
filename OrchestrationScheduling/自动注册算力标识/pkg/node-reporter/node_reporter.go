package node_reporter

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"log"
	"math"
	"strconv"
)

const (
	CPUChipType         = "cncos.org/cpu-chip-type"
	CPUChipModel        = "cncos.org/cpu-chip-model"
	CPUChipNumber       = "cncos.org/cpu-chip-number"
	GPUChipType         = "cncos.org/gpu-chip-type"
	GPUChipModel        = "cncos.org/gpu-chip-model"
	GPUChipNumber       = "cncos.org/gpu-chip-number"
	NetworkBandwidth    = "cncos.org/network-bandwidth"
	CPUComputeCapacity  = "cncos.org/cpu-compute-capacity"
	GPUComputeCapacity  = "cncos.org/gpu-compute-capacity"
	CPUPowerConsumption = "cncos.org/cpu-power-consumption"
	GPUPowerConsumption = "cncos.org/gpu-power-consumption"
	StorageCapacity     = "cncos.org/disk-capacity"
	NetworkAddress      = "cncos.org/network-address"
	CPUCapacity         = "cncos.org/cpu-capacity"
	MemoryCapacity      = "cncos.org/memory-capacity"
)

type NodeReporter struct {
	clientset *kubernetes.Clientset
	nodeName  string
}

func NewNodeReporter(clientset *kubernetes.Clientset, nodeName string) *NodeReporter {
	return &NodeReporter{clientset: clientset, nodeName: nodeName}
}

func (r *NodeReporter) UpdateNodeLabels() error {
	node, err := r.clientset.CoreV1().Nodes().Get(context.Background(), r.nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting node %s: %v", r.nodeName, err)
	}

	configMap, err := r.clientset.CoreV1().ConfigMaps("cncos-system").Get(context.Background(), "chip-codes", metav1.GetOptions{})
	if err != nil {
		log.Fatalf("Error getting ConfigMap: %v\n", err)
	}

	// 存储量
	nodeDiskCapacity, err := getNodeDiskCapacity()
	if err != nil {
		return fmt.Errorf("error getting nodeDiskCapacity resources: %v", err)
	}
	node.Labels[StorageCapacity] = fmt.Sprintf("%06d", nodeDiskCapacity)

	//网络带宽
	maxBandwidth, err := getMaxConfiguredBandwidth()
	if err != nil {
		return fmt.Errorf("error getting max configured bandwidth: %v", err)
	}
	node.Labels[NetworkBandwidth] = fmt.Sprintf("%06d", maxBandwidth)

	//CPU芯片型号、芯片类型、芯片数量
	cpuChipType, cpuChipModel, cpuChipNum, err := getCPUChipInfo(configMap.Data)
	if err != nil {
		return fmt.Errorf("error getting chip info: %v", err)
	}

	node.Labels[CPUChipType] = cpuChipType
	node.Labels[CPUChipModel] = cpuChipModel
	node.Labels[CPUChipNumber] = strconv.Itoa(cpuChipNum)

	//CPU芯片型号、芯片类型、芯片数量
	gpuChipType, gpuChipModel, gpuChipNum, err := getGPUChipInfo(configMap.Data)
	if err != nil {
		return fmt.Errorf("error getting chip info: %v", err)
	}

	node.Labels[GPUChipType] = gpuChipType
	node.Labels[GPUChipModel] = gpuChipModel
	node.Labels[GPUChipNumber] = strconv.Itoa(gpuChipNum)

	// 计算量、能耗
	node.Labels[CPUComputeCapacity] = fmt.Sprintf("%04d", int64(getCPUChipFlops()/10))
	node.Labels[CPUPowerConsumption] = fmt.Sprintf("%05d", int64(getCPUChipFlops()/10/220))

	node.Labels[GPUComputeCapacity] = fmt.Sprintf("%04d", int64(getGPUChipFlops()/10)*12)
	node.Labels[GPUPowerConsumption] = fmt.Sprintf("%05d", int64(getGPUChipFlops()*12/10/220))

	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeInternalIP {
			node.Labels[NetworkAddress] = ipv4ToBinary(addr.Address)
		}
	}

	//CPU/内存容量（不上报至互联互通平台，保留使用）
	availableCPU, availableMemory, err := getAllocatableResources(r.clientset, node)
	if err != nil {
		return fmt.Errorf("error getting available resources: %v", err)
	}
	node.Labels[CPUCapacity] = fmt.Sprintf("%04d", int64(math.Ceil(availableCPU)))
	node.Labels[MemoryCapacity] = fmt.Sprintf("%07d", int64(math.Ceil(availableMemory)))

	_, err = r.clientset.CoreV1().Nodes().Update(context.Background(), node, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error updating node %s: %v", node.Name, err)
	}

	return nil
}
