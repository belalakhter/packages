package utils

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

const (
	Fatal_Error_Code = 1
	Debug_Error_Code = 2
	Log_Info         = 3
)

func LogMessage(message string, code int) {
	switch code {
	case Fatal_Error_Code:
		_, file, line, ok := runtime.Caller(1)
		if ok {
			fmt.Printf("\nFATAL ERROR: %s\nFile: %s, Line: %d\n", message, file, line)
		} else {
			fmt.Printf("\nFATAL ERROR: %s\n", message)
		}
		fmt.Println("The application will exit now.")
		os.Exit(1)
	case Debug_Error_Code:
		fmt.Printf("\nDEBUG: %s\n", message)
	case Log_Info:
		fmt.Printf("\nINFO: %s\n", message)
	default:
		fmt.Println("Unknown error code.")
	}
}
func WelComePrint(var1 string, var2 string, var3 string, var4 string) {
	border := strings.Repeat("=", len(var1)+10)
	fmt.Println(border)
	fmt.Printf(" %s \n", var1)
	fmt.Println(border)
	fmt.Printf("\n %s\n\n", var2)
	fmt.Printf(" %s\n\n", var3)
	fmt.Printf(" %s\n\n", var4)
	fmt.Println(" Testing!")
	fmt.Println(border)
}
func CalculateStopCount(count int64, pumpCount int64) int64 {
	var TotalCount int64
	var countNew int64
	TotalCount = count
	countNew = count
	for pumpCount > 0 {

		countNew = countNew * 2

		TotalCount = TotalCount + countNew
		pumpCount--
	}

	return TotalCount
}
