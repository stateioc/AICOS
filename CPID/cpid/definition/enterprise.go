package definition

import (
	"fmt"
)

type Enterprise string

const (
	EnterpriseChinaTelecom = "2001" // 中国电信
	EnterpriseChinaMobile  = "2002" // 中国移动
	EnterpriseChinaUnion   = "2003" // 中国联通
	EnterpriseAliCloud     = "2004" // 阿里云
	EnterpriseTencentCloud = "2005" // 腾讯云
	EnterpriseHuaWeiCloud  = "2006" // 华为云
	EnterpriseUCloud       = "2007" // UCloud
	EnterpriseQingCloud    = "2008" // 青云
)

var EnterpriseMap = map[Enterprise]string{
	EnterpriseChinaTelecom: "中国电信",
	EnterpriseChinaMobile:  "中国移动",
	EnterpriseChinaUnion:   "中国联通",
	EnterpriseAliCloud:     "阿里云",
	EnterpriseTencentCloud: "腾讯云",
	EnterpriseHuaWeiCloud:  "华为云",
	EnterpriseUCloud:       "UCloud",
	EnterpriseQingCloud:    "青云",
}

var EnterpriseDescMap = map[string]Enterprise{}

func init() {
	for k, v := range EnterpriseMap {
		EnterpriseDescMap[v] = k
	}
}

func (e Enterprise) Desc() string {
	s, ok := EnterpriseMap[e]
	if !ok {
		return ""
	}
	return s
}

func GetEnterprise(desc string) (Enterprise, error) {
	e, ok := EnterpriseDescMap[desc]
	if !ok {
		return "", fmt.Errorf("enterprise desc:%s is not found", desc)
	}
	return e, nil
}
