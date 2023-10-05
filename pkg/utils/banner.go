package utils

import (
	"github.com/fatih/color"
)

func PrintLogo() {
	boldBlue := color.New(color.FgHiBlue, color.Bold)
	asciiLogo := `
  _  _____  ____  
 | |/ / _ \|  _ \ 
 | ' / | | | |_) |
 | . \ |_| |  _ < 
 |_|\_\___/|_| \_\
`

	boldBlue.Println(asciiLogo)
}
