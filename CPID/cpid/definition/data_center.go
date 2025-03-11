package definition

import "fmt"

type DataCenter string

const (
	DataCenterRegion1 = "501" // 可用区1
	DataCenterRegion2 = "502" // 可用区2
	DataCenterRegion3 = "503" // 可用区3
)

var DataCenterMap = map[DataCenter]string{
	DataCenterRegion1: "可用区1",
	DataCenterRegion2: "可用区2",
	DataCenterRegion3: "可用区3",
}

var DataCenterDescMap = map[string]DataCenter{}

func init() {
	for k, v := range DataCenterMap {
		DataCenterDescMap[v] = k
	}
}

func (dc DataCenter) Desc() string {
	s, ok := DataCenterMap[dc]
	if !ok {
		return ""
	}
	return s
}

func GetDataCenter(desc string) (DataCenter, error) {
	e, ok := DataCenterDescMap[desc]
	if !ok {
		return "", fmt.Errorf("data center desc:%s is not found", desc)
	}
	return e, nil
}
