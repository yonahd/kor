package utils

import (
	"os"

	"github.com/fatih/color"
)

func PrintLogo() {
	format := "table" // match the default format
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
	for index, arg := range os.Args {
		if arg == "-o" || arg == "--output" {
			format = os.Args[index+1]
		}
	}

	if format != "yaml" && format != "json" {
		boldBlue.Println(asciiLogo)
	}
}
