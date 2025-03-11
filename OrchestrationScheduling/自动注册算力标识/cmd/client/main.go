package main

import (
	"flag"
	"fmt"
	"os"

	"register-power-resources/pkg/client"
)

func main() {
	client.LoadConfig()

	//registerCmd := flag.NewFlagSet("register", flag.ExitOnError)
	//registerData := registerCmd.String("data", "", "Resource data string")
	//
	//unregisterCmd := flag.NewFlagSet("unregister", flag.ExitOnError)
	//unregisterID := unregisterCmd.String("id", "", "Resource ID")

	getCmd := flag.NewFlagSet("get", flag.ExitOnError)
	getID := getCmd.String("id", "", "Resource ID")

	if len(os.Args) < 2 {
		fmt.Println("Usage: client <command> [<args>]")
		fmt.Println("Commands: register, unregister, get, list")
		return
	}

	switch os.Args[1] {
	//case "register":
	//	registerCmd.Parse(os.Args[2:])
	//	client.RegisterResource(*registerData)
	//case "unregister":
	//	unregisterCmd.Parse(os.Args[2:])
	//	client.UnregisterResource(*unregisterID)
	case "get":
		getCmd.Parse(os.Args[2:])
		client.GetResource(*getID)
	case "list":
		client.ListResources()
	default:
		fmt.Println("Unknown command:", os.Args[1])
	}
}
