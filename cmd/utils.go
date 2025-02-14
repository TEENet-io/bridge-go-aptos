package cmd

import (
	"os"
)

// fileExists checks if a file exists and is readable
func FileExists(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()
	return true
}
