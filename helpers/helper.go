// +build debug

package helpers

import (
	"fmt"
	"runtime"
)

func GoroutinesCount() int {
	stackRecords := make([]runtime.StackRecord, 100)
	count, _ := runtime.GoroutineProfile(stackRecords)
	return count
}

// PrintGoroutinesInfo - prints to the console count of active goroutines and their stak traces
func PrintGoroutinesInfo() {
	stackRecords := make([]runtime.StackRecord, 100)
	count, ok := runtime.GoroutineProfile(stackRecords)
	if !ok {
		return
	}

	fmt.Println("----------------------")
	fmt.Println(fmt.Sprintf("Goroutines COUNT: %d", runtime.NumGoroutine()))
	for i := 0; i < count; i++ {
		fmt.Println("*** #", i+1)
		frames := stackRecords[i].Stack()

		for x := range frames {
			f := runtime.FuncForPC(frames[x])
			file, line := f.FileLine(frames[x])
			fmt.Println("\t", f.Name(), file, line)
		}
	}
	fmt.Println("----------------------")
}
