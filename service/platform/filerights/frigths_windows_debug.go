// +build windows,debug

package filerights

import (
	"fmt"
)

func init() {
	isDebug = true
	fmt.Println("!!! DEBUG VERSION !!! (filerights) File access permissions are not checking in DEBUG mode")
}
