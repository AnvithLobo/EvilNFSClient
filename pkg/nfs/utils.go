package nfs

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/AnvithLobo/EvilNFSClient/pkg/ui/styles"
)

// parseCommand parses a command string into parts, respecting quoted strings
func parseCommand(input string) []string {
	var parts []string
	var current strings.Builder
	inQuotes := false

	for _, char := range input {
		switch char {
		case '"':
			inQuotes = !inQuotes
		case ' ':
			if inQuotes {
				current.WriteRune(char)
			} else if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// formatBytes converts bytes to human readable format
func formatBytes(bytes int64) string {
	units := []string{"B", "KB", "MB", "GB"}
	size := float64(bytes)
	unitIndex := 0
	for size >= 1024 && unitIndex < len(units)-1 {
		size /= 1024
		unitIndex++
	}
	return fmt.Sprintf("%.2f %s", size, units[unitIndex])
}

// parseRecursiveFlag checks if the first argument is -r flag and returns (isRecursive, offset, error)
func parseRecursiveFlag(args []string, usage string) (bool, int, []string) {
	if len(args) < 1 {
		return false, 0, []string{styles.ErrorStyle.Render(usage)}
	}

	if args[0] == "-r" {
		if len(args) < 2 {
			return false, 0, []string{styles.ErrorStyle.Render(usage)}
		}
		return true, 1, nil
	}

	return false, 0, nil
}

// isExecutable checks if file is executable
func isExecutable(mode os.FileMode) bool {
	return mode&0111 != 0
}

// isSymlink checks if file is a symlink
func isSymlink(mode os.FileMode) bool {
	return mode&os.ModeSymlink != 0
}

// formatFileEntry colorizes a filename based on its mode
func formatFileEntry(name string, mode os.FileMode, isDir bool) string {
	if isDir {
		return styles.DirStyle.Render(name)
	} else if isExecutable(mode) {
		return styles.ExecStyle.Render(name)
	} else if isSymlink(mode) {
		return styles.LinkStyle.Render(name)
	}
	return styles.FileStyle.Render(name)
}

// parseOctalMode parses an octal mode string to uint64
func parseOctalMode(modeStr string) (uint64, error) {
	return strconv.ParseUint(modeStr, 8, 32)
}
