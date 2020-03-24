package main

import (
	"fmt"
	"io/ioutil"
	"path"
)

func printServStartInstructions() {
	fmt.Printf("Please, restart 'ivpn-service'\n")
	tmpDir := "/opt/ivpn/mutable"
	// print service install instructions (if exists)
	content, err := ioutil.ReadFile(path.Join(tmpDir, "service_install.txt"))
	if err == nil {
		fmt.Println(string(content))
	}
}
