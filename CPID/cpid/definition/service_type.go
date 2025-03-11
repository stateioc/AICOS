package definition

import "fmt"

type ServiceType string

const (
	ServiceTypeVirtualMachine  = "01" // 云主机
	ServiceTypeBlockStorage    = "02" // 块存储
	ServiceTypeCloudBackup     = "03" // 云备份
	ServiceTypePhysicalMachine = "04" // 物理机
	ServiceTypeCloudCache      = "05" // 云缓存
	ServiceTypeCloudDistribute = "06" // 云分发
)

var ServiceTypeMap = map[ServiceType]string{
	ServiceTypeVirtualMachine:  "云主机",
	ServiceTypeBlockStorage:    "块存储",
	ServiceTypeCloudBackup:     "云备份",
	ServiceTypePhysicalMachine: "物理机",
	ServiceTypeCloudCache:      "云缓存",
	ServiceTypeCloudDistribute: "云分发",
}

var ServiceTypeDescMap = map[string]ServiceType{}

func init() {
	for k, v := range ServiceTypeMap {
		ServiceTypeDescMap[v] = k
	}
}

func (st ServiceType) Desc() string {
	s, ok := ServiceTypeMap[st]
	if !ok {
		return ""
	}
	return s
}

func GetServiceType(desc string) (ServiceType, error) {
	e, ok := ServiceTypeDescMap[desc]
	if !ok {
		return "", fmt.Errorf("service type desc:%s is not found", desc)
	}
	return e, nil
}
