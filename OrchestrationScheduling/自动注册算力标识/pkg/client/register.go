package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type RequestBody struct {
	ComputeIDs []string `json:"compute_ids"`
}

func RegisterResource(computeIDs []string) {
	//准备请求 body

	body := struct {
		ComputeIDs []string `json:"compute_ids"`
	}{
		ComputeIDs: computeIDs,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		fmt.Printf("Error preparing request body: %v\n", err)
		return
	}

	// 向每个 server 发送 POST 请求
	for _, config := range serverConfigs {
		serverURL := config.ServerURL

		if config.Type == "local" {
			serverURL = config.ServerURL + "/resources"
		}

		fmt.Println(serverURL)

		req, err := http.NewRequest("POST", serverURL, bytes.NewBuffer(bodyBytes))
		if err != nil {
			fmt.Printf("Error creating request for %s: %v\n", config.ServerURL, err)
			continue
		}

		if config.Type == "local" {
			req.Header.Set("Content-Type", "application/json")
		}

		for key, value := range config.Headers {
			req.Header.Set(key, value)
		}

		fmt.Println("request messages begin")
		fmt.Println(req)
		fmt.Println("request messages end")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error sending request to %s: %v\n", config.ServerURL, err)
			continue
		}
		defer resp.Body.Close()

		fmt.Printf("Response from %s: %s\n", config.ServerURL, resp.Status)
		fmt.Println(resp)
	}
}
