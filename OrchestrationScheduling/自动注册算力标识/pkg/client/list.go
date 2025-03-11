package client

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"register-power-resources/pkg/apis"
	"strings"
)

func ListResources() {
	for _, config := range serverConfigs {
		if config.Type == "local" {
			resp, err := http.Get(config.ServerURL + "/resources")
			if err != nil {
				fmt.Println("Error listing resources:", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				fmt.Println("Error listing resources: status code", resp.StatusCode)
				return
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println("Error reading response body:", err)
				return
			}

			if len(string(body)) != 0 {
				fmt.Println("All Resources:")
				fmt.Println(string(body))

				nodeInfos := strings.Split(string(body), "\n")
				for _, node := range nodeInfos {
					fmt.Println("==================================================================================" +
						"===============================================")
					fmt.Println("Resource detail for " + node + ":")
					nodeInfo := apis.ParseResourceInfo(node)
					printStructFields(nodeInfo)
				}
			}
		}
	}
}

func printStructFields(obj interface{}) {
	value := reflect.ValueOf(obj)

	// 如果传入的对象是指针，获取指针指向的结构体值
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	typ := value.Type()

	for i := 0; i < value.NumField(); i++ {
		fieldName := typ.Field(i).Name
		fieldValue := value.Field(i).Interface()
		fmt.Printf("%s: %v\n", fieldName, fieldValue)
	}
}
