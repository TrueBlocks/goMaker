package main

import (
	"embed"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/TrueBlocks/goMaker/v6/types"
	"github.com/TrueBlocks/trueblocks-chifra/v6/pkg/logger"
)

//go:embed help.txt help_verbose.txt VERSION
var helpFS embed.FS

// Build-time variables
var (
	compileTime = "unknown"
)

func showHelp() {
	helpFileName := "help.txt"
	if types.IsVerbose() {
		helpFileName = "help_verbose.txt"
	}

	helpFile, err := helpFS.Open(helpFileName)
	if err != nil {
		fmt.Println("Error: Could not load help text.")
		return
	}
	defer helpFile.Close()

	helpContent, err := io.ReadAll(helpFile)
	if err != nil {
		fmt.Println("Error: Could not read help text.")
		return
	}

	fmt.Println(string(helpContent))
}

func showVersion() {
	versionFile, err := helpFS.Open("VERSION")
	if err != nil {
		fmt.Println("Error: Could not load version information.")
		return
	}
	defer versionFile.Close()

	versionContent, err := io.ReadAll(versionFile)
	if err != nil {
		fmt.Println("Error: Could not read version information.")
		return
	}

	version := strings.TrimSpace(string(versionContent))
	fmt.Println("Version:  v" + version)
	fmt.Println("Compiled:", compileTime)
}

func main() {
	showHelpFlag := false
	showVersionFlag := false

	// Validate all arguments first
	for i, arg := range os.Args {
		if i == 0 { // Skip program name
			continue
		}

		switch arg {
		case "--help", "-h", "-help", "help":
			showHelpFlag = true
		case "--verbose", "-v", "-verbose":
			types.SetVerbose(true)
		case "--version":
			showVersionFlag = true
		default:
			fmt.Printf("Error: Unknown option '%s'\n\n", arg)
			fmt.Println("Valid options:")
			fmt.Println("  --help, -h     Show help information")
			fmt.Println("  --verbose, -v  Show verbose help information")
			fmt.Println("  --version      Show version information")
			os.Exit(1)
		}
	}

	if showVersionFlag {
		showVersion()
		return
	}

	if showHelpFlag {
		showHelp()
		return
	}

	// Normal execution
	pwd, _ := os.Getwd()
	logger.InfoBY("Current folder:", pwd)

	// First check if templates folder exists and isn't empty
	if err := types.ValidateTemplatesFolder(); err != nil {
		fmt.Println("Error:", err)
		fmt.Println("\nHere are the requirements to run goMaker:")
		types.SetVerbose(false)
		showHelp()
		os.Exit(1)
	}

	codeBase, err := types.LoadCodebase()
	if err != nil {
		if strings.Contains(err.Error(), "could not find the templates directory") {
			fmt.Println("Error:", err)
			fmt.Println("\nHere are the requirements to run goMaker:")
			types.SetVerbose(false)
			showHelp()
			os.Exit(1)
		}
		logger.Fatal(err)
	}
	codeBase.Generate()
}
