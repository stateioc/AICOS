package cpid

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {

	cpidStr := "1101/tc/2004/01/502/01"
	cpid, err := Parse(cpidStr)
	assert.Nil(t, err)
	assert.Equal(t, "北京", cpid.Area.Desc())
	assert.Equal(t, "电讯业", cpid.Industry.Desc())
	assert.Equal(t, "阿里云", cpid.Enterprise.Desc())
	assert.Equal(t, "超算", cpid.ResourceType.Desc())
	assert.Equal(t, "可用区2", cpid.DataCenter.Desc())
	assert.Equal(t, "云主机", cpid.ServiceType.Desc())

	fmt.Printf("cpidStr: %s \nresult: %s\n", cpidStr, cpid.CpidDesc().String())
}
