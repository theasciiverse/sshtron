package main

import (
	"os"
	"path/filepath"
	"runtime"
)

// dataDir is where the host key and the known servers are kept:
// $XDG_DATA_HOME/sshtron, ~/Library/Application Support/sshtron on macOS,
// %APPDATA%\sshtron on Windows.
func dataDir() (string, error) {
	switch runtime.GOOS {
	case "windows", "darwin":
		dir, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(dir, "sshtron"), nil
	}
	if dir := os.Getenv("XDG_DATA_HOME"); filepath.IsAbs(dir) {
		return filepath.Join(dir, "sshtron"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "sshtron"), nil
}
