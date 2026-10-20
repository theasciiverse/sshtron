package main

import (
	"os"

	"golang.org/x/sys/windows"
)

// enableVT lets the console draw the escape codes the game is made of
func enableVT() {
	h := windows.Handle(os.Stdout.Fd())
	var mode uint32
	if windows.GetConsoleMode(h, &mode) == nil {
		windows.SetConsoleMode(h, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING|
			windows.DISABLE_NEWLINE_AUTO_RETURN)
	}
	windows.SetConsoleOutputCP(65001)
}
