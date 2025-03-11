package cpid

import (
	"errors"
	"strings"

	"cncos.cn/cncos/open-cnc/cpid/definition"
)

type Cpid struct {
	Area         definition.Area
	Industry     definition.Industry
	Enterprise   definition.Enterprise
	ResourceType definition.ResourceType
	DataCenter   definition.DataCenter
	ServiceType  definition.ServiceType
}
type CpidDesc struct {
	AreaDesc         string
	IndustryDesc     string
	EnterpriseDesc   string
	ResourceTypeDesc string
	DataCenterDesc   string
	ServiceTypeDesc  string
}

func (id Cpid) String() string {
	s := make([]string, 0)
	s = append(s, string(id.Area))
	s = append(s, string(id.Industry))
	s = append(s, string(id.Enterprise))
	s = append(s, string(id.ResourceType))
	s = append(s, string(id.DataCenter))
	s = append(s, string(id.ServiceType))

	return strings.Join(s, "/")
}
func (id *Cpid) CpidDesc() *CpidDesc {
	desc := &CpidDesc{
		AreaDesc:         id.Area.Desc(),
		IndustryDesc:     id.Industry.Desc(),
		EnterpriseDesc:   id.Enterprise.Desc(),
		ResourceTypeDesc: id.ResourceType.Desc(),
		DataCenterDesc:   id.DataCenter.Desc(),
		ServiceTypeDesc:  id.ServiceType.Desc(),
	}
	return desc
}

func (id CpidDesc) String() string {
	s := make([]string, 0)
	s = append(s, id.AreaDesc)
	s = append(s, id.IndustryDesc)
	s = append(s, id.EnterpriseDesc)
	s = append(s, id.ResourceTypeDesc)
	s = append(s, id.DataCenterDesc)
	s = append(s, id.ServiceTypeDesc)

	return strings.Join(s, "/")
}

func Parse(cpidStr string) (cpid *Cpid, err error) {
	s := strings.Split(cpidStr, "/")
	if len(s) < 6 {
		return nil, errors.New("cpidStr is invalid")
	}

	cpid = &Cpid{
		Area:         definition.Area(s[0]),
		Industry:     definition.Industry(s[1]),
		Enterprise:   definition.Enterprise(s[2]),
		ResourceType: definition.ResourceType(s[3]),
		DataCenter:   definition.DataCenter(s[4]),
		ServiceType:  definition.ServiceType(s[5]),
	}

	return
}
