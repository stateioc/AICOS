package client

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

func GetResource(id string) {
	for _, config := range serverConfigs {
		if config.Type == "local" {
			resp, err := http.Get(config.ServerURL + "/resources/" + id)
			if err != nil {
				fmt.Println("Error getting resource:", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				fmt.Println("Error getting resource: status code", resp.StatusCode)
				return
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println("Error reading response body:", err)
				return
			}

			fmt.Println("Resource:", string(body))
		}
	}

}
