package nfs

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/AnvithLobo/EvilNFSClient/pkg/ui/styles"
) // NFS Operations - List

func (c *NFSClient) ls(args []string) []string {
	targetPath := c.CurrentPath
	if len(args) > 0 {
		targetPath = c.resolvePath(args[0])
	}

	entries, err := c.mount.ReadDirPlus(targetPath)
	if err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", err))}
	}

	type formattedEntry struct {
		perms string
		uid   uint32
		gid   uint32
		size  string
		date  string
		mode  os.FileMode
		isDir bool
		name  string
	}

	var formattedEntries []formattedEntry

	for _, entry := range entries {
		name := entry.Name()
		if name == "." || name == ".." {
			continue
		}

		mode := entry.Mode()
		size := entry.Size()
		mtime := entry.ModTime()

		uid := uint32(0)
		gid := uint32(0)
		if entry.Attr.IsSet {
			uid = entry.Attr.Attr.UID
			gid = entry.Attr.Attr.GID
		}

		typeChar := "-"
		if entry.IsDir() {
			typeChar = "d"
		}

		perms := fmt.Sprintf("%s%s", typeChar, mode.Perm().String())
		dateStr := mtime.Format("Jan 02 15:04")
		sizeStr := fmt.Sprintf("%d", size)

		formattedEntries = append(formattedEntries, formattedEntry{
			perms: perms,
			uid:   uid,
			gid:   gid,
			size:  sizeStr,
			date:  dateStr,
			mode:  mode,
			isDir: entry.IsDir(),
			name:  name,
		})
	}

	maxUIDWidth := 5
	maxGIDWidth := 5
	for _, entry := range formattedEntries {
		uidWidth := len(fmt.Sprintf("%d", entry.uid))
		gidWidth := len(fmt.Sprintf("%d", entry.gid))
		if uidWidth > maxUIDWidth {
			maxUIDWidth = uidWidth
		}
		if gidWidth > maxGIDWidth {
			maxGIDWidth = gidWidth
		}
	}

	var output []string
	for _, entry := range formattedEntries {
		line := fmt.Sprintf("%-10s %*d %*d %8s %s ",
			entry.perms,
			maxUIDWidth,
			entry.uid,
			maxGIDWidth,
			entry.gid,
			entry.size,
			entry.date)

		line += formatFileEntry(entry.name, entry.mode, entry.isDir)
		output = append(output, line)
	}

	return output
}

// NFS Operations - Navigation

func (c *NFSClient) cd(args []string) []string {
	if len(args) < 1 {
		return []string{styles.ErrorStyle.Render("Usage: cd <path>")}
	}

	targetPath := args[0]

	if targetPath == ".." {
		if c.CurrentPath == "/" {
			c.CurrentPath = "/"
		} else {
			c.CurrentPath = path.Dir(c.CurrentPath)
		}
	} else if targetPath == "." {
		// Stay in current directory
	} else {
		if !strings.HasPrefix(targetPath, "/") {
			if c.CurrentPath == "/" {
				targetPath = "/" + targetPath
			} else {
				targetPath = path.Join(c.CurrentPath, targetPath)
			}
		}
		c.CurrentPath = path.Clean(targetPath)
	}

	entries, err := c.mount.ReadDirPlus(c.CurrentPath)
	if err != nil {
		c.CurrentPath = "/"
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error: directory not found or not accessible: %v", err))}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			return []string{styles.SuccessStyle.Render(fmt.Sprintf("Changed directory to: %s", c.CurrentPath))}
		}
	}

	return []string{styles.SuccessStyle.Render(fmt.Sprintf("Changed directory to: %s", c.CurrentPath))}
}

func (c *NFSClient) tree(args []string) []string {
	targetPath := c.CurrentPath
	if len(args) > 0 {
		targetPath = c.resolvePath(args[0])
	}

	var output []string
	output = append(output, targetPath)
	c.treeRecursive(targetPath, "", &output, true)
	return output
}

func (c *NFSClient) treeRecursive(dir, prefix string, output *[]string, isLast bool) {
	entries, err := c.mount.ReadDirPlus(dir)
	if err != nil {
		*output = append(*output, prefix+styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", err)))
		return
	}

	for i, entry := range entries {
		name := entry.Name()
		if name == "." || name == ".." {
			continue
		}

		isLastEntry := i == len(entries)-1
		connector := "├── "
		if isLastEntry {
			connector = "└── "
		}

		if entry.IsDir() {
			*output = append(*output, prefix+connector+"📁 "+name)

			newPrefix := prefix
			if isLastEntry {
				newPrefix += "    "
			} else {
				newPrefix += "│   "
			}

			newPath := path.Join(dir, name)
			c.treeRecursive(newPath, newPrefix, output, isLastEntry)
		} else {
			*output = append(*output, prefix+connector+name)
		}
	}
}

// NFS Operations - File Permissions

func (c *NFSClient) chmod(args []string) []string {
	if len(args) < 2 {
		return []string{styles.ErrorStyle.Render("Usage: chmod <mode> <file>")}
	}

	modeStr := args[0]
	filePath := args[1]

	resolvedPath := c.resolvePath(filePath)

	mode, err := strconv.ParseUint(modeStr, 8, 32)
	if err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Invalid mode: %v", err))}
	}

	err = c.mount.Chmod(resolvedPath, uint32(mode))
	if err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", err))}
	}

	return []string{styles.SuccessStyle.Render(fmt.Sprintf("Changed permissions of %s to %o", resolvedPath, mode))}
}

// NFS Operations - Download/Get

func (c *NFSClient) get(args []string) []string {
	recursive, offset, errMsg := parseRecursiveFlag(args, "Usage: get [-r] <remote_path> [<local_path>]")
	if errMsg != nil {
		return errMsg
	}

	remotePath := c.resolvePath(args[offset])

	localPath := filepath.Join(c.localPath, filepath.Base(remotePath))

	if len(args) > offset+1 {
		localPath = args[offset+1]
		if !filepath.IsAbs(localPath) {
			localPath = filepath.Join(c.localPath, localPath)
		}
	}

	if recursive {
		return c.downloadRecursive(remotePath, localPath)
	}
	return c.downloadFile(remotePath, localPath)
}

func (c *NFSClient) downloadFile(remotePath, localPath string) []string {
	file, err := c.mount.Open(remotePath)
	if err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error opening remote file: %v", err))}
	}
	defer file.Close()

	finalLocalPath := localPath

	if strings.HasSuffix(localPath, "/") {
		finalLocalPath = filepath.Join(localPath, filepath.Base(remotePath))
		err := os.MkdirAll(localPath, 0755)
		if err != nil {
			return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error creating local directory: %v", err))}
		}
	} else {
		info, err := os.Stat(localPath)
		if err == nil && info.IsDir() {
			finalLocalPath = filepath.Join(localPath, filepath.Base(remotePath))
		}
	}

	localFile, err := os.Create(finalLocalPath)
	if err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error creating local file: %v", err))}
	}
	defer localFile.Close()

	written, err := io.Copy(localFile, file)
	if err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error downloading: %v", err))}
	}

	sizeStr := formatBytes(written)
	return []string{styles.SuccessStyle.Render(fmt.Sprintf("Downloaded: %s -> %s (%s)", remotePath, finalLocalPath, sizeStr))}
}

func (c *NFSClient) downloadRecursive(remoteDir, localDir string) []string {
	var output []string

	err := os.MkdirAll(localDir, 0755)
	if err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error creating local dir: %v", err))}
	}

	entries, err := c.mount.ReadDirPlus(remoteDir)
	if err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error reading remote dir: %v", err))}
	}

	for _, entry := range entries {
		name := entry.Name()
		if name == "." || name == ".." {
			continue
		}

		remotePath := path.Join(remoteDir, name)
		localPath := filepath.Join(localDir, name)

		if entry.IsDir() {
			result := c.downloadRecursive(remotePath, localPath)
			output = append(output, result...)
		} else {
			result := c.downloadFile(remotePath, localPath)
			output = append(output, result...)
		}
	}

	return output
}

func (c *NFSClient) mget(args []string) []string {
	if len(args) < 1 {
		return []string{styles.ErrorStyle.Render("Usage: mget <pattern> [<dest_dir>]")}
	}

	pattern := args[0]
	destDir := "."
	dir := c.CurrentPath

	if strings.Contains(pattern, "/") {
		dir = c.resolvePath(path.Dir(pattern))
		pattern = path.Base(pattern)
	} else {
		dir = c.resolvePath(dir)
	}

	if len(args) > 1 {
		destDir = args[1]
	} else {
		destDir = c.localPath
	}

	err := os.MkdirAll(destDir, 0755)
	if err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error creating destination directory: %v", err))}
	}

	entries, err := c.mount.ReadDirPlus(dir)
	if err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", err))}
	}

	var output []string
	matchCount := 0

	for _, entry := range entries {
		name := entry.Name()
		matched, err := filepath.Match(pattern, name)
		if err != nil {
			continue
		}

		if matched && !entry.IsDir() {
			remotePath := path.Join(dir, name)
			localPath := filepath.Join(destDir, name)
			result := c.downloadFile(remotePath, localPath)
			output = append(output, result...)
			matchCount++
		}
	}

	if matchCount == 0 {
		output = append(output, styles.ErrorStyle.Render("No files matched pattern"))
	} else {
		output = append(output, styles.SuccessStyle.Render(fmt.Sprintf("Downloaded %d file(s)", matchCount)))
	}

	return output
}

// NFS Operations - Upload/Put

func (c *NFSClient) put(args []string) []string {
	recursive, offset, errMsg := parseRecursiveFlag(args, "Usage: put [-r] <local_path> [<remote_path>]")
	if errMsg != nil {
		return errMsg
	}

	localPath := args[offset]

	if !filepath.IsAbs(localPath) {
		localPath = filepath.Join(c.localPath, localPath)
	}

	remotePath := filepath.Base(localPath)

	if len(args) > offset+1 {
		remotePath = args[offset+1]
	}

	if recursive {
		return c.uploadRecursive(localPath, remotePath)
	}

	return c.uploadFile(localPath, remotePath)
}

func (c *NFSClient) uploadFile(localPath, remotePath string) []string {
	localFile, err := os.Open(localPath)
	if err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error opening local file: %v", err))}
	}
	defer localFile.Close()

	finalRemotePath := c.resolvePath(remotePath)

	if strings.HasSuffix(remotePath, "/") {
		finalRemotePath = path.Join(finalRemotePath, filepath.Base(localPath))
	} else {
		entries, err := c.mount.ReadDirPlus(finalRemotePath)
		if err == nil && len(entries) >= 0 {
			finalRemotePath = path.Join(finalRemotePath, filepath.Base(localPath))
		}
	}

	remoteFile, err := c.mount.OpenFile(finalRemotePath, 0644)
	if err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error creating remote file: %v", err))}
	}
	defer remoteFile.Close()

	written, err := io.Copy(remoteFile, localFile)
	if err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error uploading: %v", err))}
	}

	sizeStr := formatBytes(written)
	return []string{styles.SuccessStyle.Render(fmt.Sprintf("Uploaded: %s -> %s (%s)", localPath, finalRemotePath, sizeStr))}
}

func (c *NFSClient) uploadRecursive(localDir, remoteDir string) []string {
	var output []string

	resolvedRemoteDir := c.resolvePath(remoteDir)

	_, err := c.mount.Mkdir(resolvedRemoteDir, 0755)
	if err != nil {
		// Directory might already exist, continue
	}

	entries, err := os.ReadDir(localDir)
	if err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error reading local dir: %v", err))}
	}

	for _, entry := range entries {
		localPath := filepath.Join(localDir, entry.Name())
		remotePath := path.Join(resolvedRemoteDir, entry.Name())

		if entry.IsDir() {
			result := c.uploadRecursive(localPath, remotePath)
			output = append(output, result...)
		} else {
			result := c.uploadFile(localPath, remotePath)
			output = append(output, result...)
		}
	}

	return output
}

func (c *NFSClient) mput(args []string) []string {
	if len(args) < 1 {
		return []string{styles.ErrorStyle.Render("Usage: mput <pattern> [<dest_path>]")}
	}

	pattern := args[0]
	remotePath := c.CurrentPath

	if len(args) > 1 {
		remotePath = c.resolvePath(args[1])
	}

	var searchPattern string
	if filepath.IsAbs(pattern) {
		searchPattern = pattern
	} else {
		searchPattern = filepath.Join(c.localPath, pattern)
	}

	matches, err := filepath.Glob(searchPattern)
	if err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Invalid pattern: %v", err))}
	}

	if len(matches) == 0 {
		return []string{styles.ErrorStyle.Render("No files matched pattern")}
	}

	var output []string
	uploadCount := 0
	for _, localPath := range matches {
		info, err := os.Stat(localPath)
		if err != nil || info.IsDir() {
			continue
		}

		finalRemotePath := remotePath
		if strings.HasSuffix(remotePath, "/") {
			finalRemotePath = path.Join(remotePath, filepath.Base(localPath))
		} else {
			finalRemotePath = path.Join(remotePath, filepath.Base(localPath))
		}

		result := c.uploadFile(localPath, finalRemotePath)
		output = append(output, result...)
		uploadCount++
	}

	output = append(output, styles.SuccessStyle.Render(fmt.Sprintf("Uploaded %d file(s)", uploadCount)))
	return output
}

// NFS Operations - File/Directory Management

func (c *NFSClient) rm(args []string) []string {
	recursive, offset, errMsg := parseRecursiveFlag(args, "Usage: rm [-r] <path>")
	if errMsg != nil {
		return errMsg
	}

	targetPath := c.resolvePath(args[offset])

	if recursive {
		return c.removeRecursive(targetPath)
	}
	return c.removeSingle(targetPath)
}

func (c *NFSClient) removeSingle(targetPath string) []string {
	err := c.mount.Remove(targetPath)
	if err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", err))}
	}
	return []string{styles.SuccessStyle.Render(fmt.Sprintf("Removed: %s", targetPath))}
}

func (c *NFSClient) removeRecursive(targetPath string) []string {
	err := c.mount.RemoveAll(targetPath)
	if err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", err))}
	}
	return []string{styles.SuccessStyle.Render(fmt.Sprintf("Removed: %s", targetPath))}
}

func (c *NFSClient) mkdir(args []string) []string {
	if len(args) < 1 {
		return []string{styles.ErrorStyle.Render("Usage: mkdir [-p] <path>")}
	}

	createParents := false
	offset := 0
	if args[0] == "-p" {
		createParents = true
		offset = 1
		if len(args) < 2 {
			return []string{styles.ErrorStyle.Render("Usage: mkdir [-p] <path>")}
		}
	}

	targetPath := c.resolvePath(args[offset])

	if createParents {
		return c.createDirRecursive(targetPath)
	}

	_, err := c.mount.Mkdir(targetPath, 0755)
	if err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", err))}
	}

	return []string{styles.SuccessStyle.Render(fmt.Sprintf("Created directory: %s", targetPath))}
}

func (c *NFSClient) createDirRecursive(targetPath string) []string {
	parts := strings.Split(strings.Trim(targetPath, "/"), "/")
	currentPath := ""
	if strings.HasPrefix(targetPath, "/") {
		currentPath = "/"
	}

	for _, part := range parts {
		if part == "" {
			continue
		}
		if currentPath == "/" {
			currentPath = "/" + part
		} else {
			currentPath = path.Join(currentPath, part)
		}

		entries, err := c.mount.ReadDirPlus(currentPath)
		if err == nil && len(entries) > 0 {
			continue
		}

		_, err = c.mount.Mkdir(currentPath, 0755)
		if err != nil {
			entries, checkErr := c.mount.ReadDirPlus(currentPath)
			if checkErr == nil && len(entries) > 0 {
				continue
			}
			return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error creating %s: %v", currentPath, err))}
		}
	}
	return []string{styles.SuccessStyle.Render(fmt.Sprintf("Created directory (with parents): %s", targetPath))}
}

// resolvePath resolves a relative path against the current working directory
func (c *NFSClient) resolvePath(p string) string {
	if strings.HasPrefix(p, "/") {
		// Absolute path
		return path.Clean(p)
	}
	// Relative path - join with currentPath
	if c.CurrentPath == "/" {
		return "/" + path.Clean(p)
	}
	return path.Clean(path.Join(c.CurrentPath, p))
}

// resolveLocalPath handles ~ expansion and relative path resolution for local filesystem paths
func (c *NFSClient) resolveLocalPath(p string) string {
	// Handle ~ expansion
	if strings.HasPrefix(p, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			p = filepath.Join(home, p[1:])
		}
	}

	// Handle relative paths
	if !filepath.IsAbs(p) {
		p = filepath.Join(c.localPath, p)
	}

	return p
}
