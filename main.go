package main

import (
	"fmt"
	"os"

	"github.com/edgeflare/fabric-oidc-proxy/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Println("Error executing command:", err)
		os.Exit(1)
	}
}
