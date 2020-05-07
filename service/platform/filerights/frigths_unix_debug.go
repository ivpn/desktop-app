// +build darwin linux

package filerights

import (
	"fmt"
)

func init() {
	isDebug = true
	fmt.Println("!!! WARNING !!! (filerights) File access permissions are not checking in DEBUG mode")
}
