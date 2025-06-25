package pool

import (
	"fmt"
	"os"
	"runtime"
)

const (
	fatal_Error_Code = 1
	debug_Error_Code = 2
	log_Info         = 3
)

func LogMessage(message string, code int) {
	switch code {
	case fatal_Error_Code:
		_, file, line, ok := runtime.Caller(1)
		if ok {
			fmt.Printf("\nFATAL ERROR: %s\nFile: %s, Line: %d\n", message, file, line)
		} else {
			fmt.Printf("\nFATAL ERROR: %s\n", message)
		}
		fmt.Println("The application will exit now.")
		os.Exit(1)
	case debug_Error_Code:
		fmt.Printf("\nDEBUG: %s\n", message)
	case log_Info:
		fmt.Printf("\nINFO: %s\n", message)
	default:
		fmt.Println("Unknown error code.")
	}
}
