package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("atsheet - activity timesheet tool")
        fmt.Println("Usage: atsheet <command> [flags]")
        fmt.Println("Commands: build, report, diff, generate, validate, version")
        os.Exit(0)
	}

	switch os.Args[1] {
	case "version":
		fmt.Println("atsheet version 0.1.0")
	case "validate":
		os.Exit(RunValidate(os.Args[2:]))
	default:
		fmt.Printf("unknown command: %s\n", os.Args[1])
        os.Exit(2)
	}
	
}
