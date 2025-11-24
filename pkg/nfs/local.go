package nfs

import (
	"fmt"
	"os"

	"github.com/AnvithLobo/EvilNFSClient/pkg/ui/styles"
)

// lls lists files in local directory
func (c *NFSClient) lls(args []string) []string {
	targetPath := c.localPath
	if len(args) > 0 {
		targetPath = c.resolveLocalPath(args[0])
	}

	entries, err := os.ReadDir(targetPath)
	if err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error reading directory: %v", err))}
	}

	var output []string
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		mode := info.Mode()
		size := info.Size()
		mtime := info.ModTime()

		typeChar := "-"
		if entry.IsDir() {
			typeChar = "d"
		} else if mode&os.ModeSymlink != 0 {
			typeChar = "l"
		}

		perms := fmt.Sprintf("%s%s", typeChar, mode.Perm().String())
		dateStr := mtime.Format("Jan 02 15:04")
		sizeStr := fmt.Sprintf("%d", size)

		line := fmt.Sprintf("%-10s %8s %s ", perms, sizeStr, dateStr)

		line += formatFileEntry(entry.Name(), mode, entry.IsDir())
		output = append(output, line)
	}

	return output
}

// lcd changes local directory
func (c *NFSClient) lcd(args []string) []string {
	if len(args) < 1 {
		return []string{styles.ErrorStyle.Render("Usage: lcd <path>")}
	}

	targetPath := c.resolveLocalPath(args[0])

	info, err := os.Stat(targetPath)
	if err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error: directory not found or not accessible: %v", err))}
	}

	if !info.IsDir() {
		return []string{styles.ErrorStyle.Render("Error: not a directory")}
	}

	c.localPath = targetPath
	return []string{styles.SuccessStyle.Render(fmt.Sprintf("Changed local directory to: %s", c.localPath))}
}

// lmkdir creates directory on local system
func (c *NFSClient) lmkdir(args []string) []string {
	if len(args) < 1 {
		return []string{styles.ErrorStyle.Render("Usage: lmkdir [-p] <path>")}
	}

	createParents := false
	offset := 0
	if args[0] == "-p" {
		createParents = true
		offset = 1
		if len(args) < 2 {
			return []string{styles.ErrorStyle.Render("Usage: lmkdir [-p] <path>")}
		}
	}

	targetPath := c.resolveLocalPath(args[offset])

	if createParents {
		err := os.MkdirAll(targetPath, 0755)
		if err != nil {
			return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", err))}
		}
		return []string{styles.SuccessStyle.Render(fmt.Sprintf("Created directory (with parents): %s", targetPath))}
	}

	err := os.Mkdir(targetPath, 0755)
	if err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", err))}
	}

	return []string{styles.SuccessStyle.Render(fmt.Sprintf("Created directory: %s", targetPath))}
}
