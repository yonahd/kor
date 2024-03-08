package utils

import (
	"fmt"

	"github.com/fatih/color"
)

var Version = "dev"

func PrintVersion() {
	fmt.Printf("kor version: v%s\n", Version)
}

func PrintLogo(outputFormat string) {
	boldBlue := color.New(color.FgHiBlue, color.Bold)
	asciiLogo := `
  _  _____  ____
 | |/ / _ \|  _ \
 | ' / | | | |_) |
 | . \ |_| |  _ <
 |_|\_\___/|_| \_\
`
	// processing of the `outputFormat` happens inside of the rootCmd so this requires a pretty large change
	// to keep the banner. Instead just loop through os args and find if the format was set and handle it there
	if outputFormat != "yaml" && outputFormat != "json" {
		PrintVersion()
		boldBlue.Println(asciiLogo)
	}
}
