package definition

import "fmt"

type Industry string

const (
	IndustryTelecommunication = "tc" // 电讯业
	IndustryComputing         = "cp" // 计算机
	IndustryInternet          = "it" // 因特网
)

var IndustryMap = map[Industry]string{
	IndustryTelecommunication: "电讯业",
	IndustryComputing:         "计算机",
	IndustryInternet:          "因特网",
}

var IndustryDescMap = map[string]Industry{}

func init() {
	for k, v := range IndustryMap {
		IndustryDescMap[v] = k
	}
}

func (i Industry) Desc() string {
	s, ok := IndustryMap[i]
	if !ok {
		return ""
	}
	return s
}

func GetIndustry(desc string) (Industry, error) {
	e, ok := IndustryDescMap[desc]
	if !ok {
		return "", fmt.Errorf("industry desc:%s is not found", desc)
	}
	return e, nil
}
