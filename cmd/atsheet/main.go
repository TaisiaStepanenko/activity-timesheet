package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println("atsheet version 0.1.0")
		return
	}
	fmt.Println("atsheet - activity timesheet tool")
	fmt.Println("Usage: atsheet <command> [flags]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  build      Build timesheet from activity and calendar")
	fmt.Println("  report     Generate Markdown report")
	fmt.Println("  diff       Compare two timesheets")
	fmt.Println("  generate   Generate synthetic data triple")
	fmt.Println("  validate   Validate calendar and activity stream")
	fmt.Println("  version    Show version")
	fmt.Println()
	fmt.Println("Run 'atsheet <command> --help' for more details.")
}
