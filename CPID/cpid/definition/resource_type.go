package definition

import "fmt"

type ResourceType string

const (
	ResourceTypeHpc = "01" // 超级计算
	ResourceTypeIc  = "02" // 智能计算
	ResourceTypeGc  = "03" // 通用计算
)

var ResourceTypeMap = map[ResourceType]string{
	ResourceTypeHpc: "超算",
	ResourceTypeIc:  "智算",
	ResourceTypeGc:  "通用",
}

var ResourceTypeDescMap = map[string]ResourceType{}

func init() {
	for k, v := range ResourceTypeMap {
		ResourceTypeDescMap[v] = k
	}
}

func (rt ResourceType) Desc() string {
	s, ok := ResourceTypeMap[rt]
	if !ok {
		return ""
	}
	return s
}

func GetResourceType(desc string) (ResourceType, error) {
	e, ok := ResourceTypeDescMap[desc]
	if !ok {
		return "", fmt.Errorf("resource type desc:%s is not found", desc)
	}
	return e, nil
}
