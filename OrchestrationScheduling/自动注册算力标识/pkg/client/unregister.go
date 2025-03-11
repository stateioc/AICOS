package client

import (
	"fmt"
	"net/http"
)

func UnregisterResource(id string) {
	for _, config := range serverConfigs {
		if config.Type == "local" {
			req, err := http.NewRequest("DELETE", config.ServerURL+"/resources/"+id, nil)
			if err != nil {
				fmt.Println("Error unregistering resource:", err)
				return
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Println("Error unregistering resource:", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusNoContent {
				fmt.Println("Error unregistering resource: status code", resp.StatusCode)
				return
			}

			fmt.Println("[Debug]Resource unregistered successfully")
		}
	}
}
