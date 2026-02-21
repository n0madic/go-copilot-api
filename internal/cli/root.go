package cli

import (
	"fmt"
	"os"
)

func Run(args []string) int {
	if len(args) == 0 {
		printMainHelp()
		return 0
	}

	sub := args[0]
	switch sub {
	case "start":
		return runStart(args[1:])
	case "auth":
		return runAuth(args[1:])
	case "check-usage":
		return runCheckUsage(args[1:])
	case "debug":
		return runDebug(args[1:])
	case "-h", "--help", "help":
		printMainHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", sub)
		printMainHelp()
		return 1
	}
}

func printMainHelp() {
	fmt.Println("copilot-api")
	fmt.Println("\nCommands:")
	fmt.Println("  start        Start the Copilot API server")
	fmt.Println("  auth         Run GitHub auth flow")
	fmt.Println("  check-usage  Show Copilot usage")
	fmt.Println("  debug        Print debug information")
}
