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

// batchState tracks shared counters across a multi-file transfer so that
// every progressWriter in the batch can report global progress.
type batchState struct {
	fileIndex     *int   // current file number (1-based), incremented before each file
	filesTotal    int    // total file count in the batch
	globalWritten *int64 // cumulative bytes written across all files
	globalTotal   int64  // pre-calculated total bytes (0 = unknown)
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
	return c.doDownloadFile(remotePath, localPath, filepath.Base(remotePath), nil)
}

// doDownloadFile is the shared implementation used by single-file and batch downloads.
// batch == nil means a standalone single-file transfer.
func (c *NFSClient) doDownloadFile(remotePath, localPath, name string, batch *batchState) []string {
	// Best-effort file size for the progress bar
	var fileTotal int64
	if info, _, err := c.mount.Lookup(remotePath); err == nil {
		fileTotal = info.Size()
	}

	file, err := c.mount.Open(remotePath)
	if err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error opening remote file: %v", err))}
	}
	defer file.Close()

	finalLocalPath := localPath
	if strings.HasSuffix(localPath, "/") {
		finalLocalPath = filepath.Join(localPath, filepath.Base(remotePath))
		if err := os.MkdirAll(localPath, 0755); err != nil {
			return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error creating local directory: %v", err))}
		}
	} else {
		if info, err := os.Stat(localPath); err == nil && info.IsDir() {
			finalLocalPath = filepath.Join(localPath, filepath.Base(remotePath))
		}
	}

	localFile, err := os.Create(finalLocalPath)
	if err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error creating local file: %v", err))}
	}
	defer localFile.Close()

	cfg := progressWriterConfig{
		filename:  name,
		fileTotal: fileTotal,
	}
	if batch != nil {
		*batch.fileIndex++
		cfg.fileIndex     = *batch.fileIndex
		cfg.filesTotal    = batch.filesTotal
		cfg.globalWritten = batch.globalWritten
		cfg.globalTotal   = batch.globalTotal
	}

	pw := newProgressWriter(localFile, cfg, c.progressFn)
	written, err := io.Copy(pw, file)
	pw.flush()
	if err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error downloading: %v", err))}
	}

	sizeStr := formatBytes(written)
	return []string{styles.SuccessStyle.Render(fmt.Sprintf("Downloaded: %s -> %s (%s)", remotePath, finalLocalPath, sizeStr))}
}

func (c *NFSClient) downloadRecursive(remoteDir, localDir string) []string {
	// Pre-scan to get file count and total bytes for global progress
	total, totalBytes := countRemoteFilesAndSize(c, remoteDir)
	fileIndex := 0
	globalWritten := int64(0)
	batch := &batchState{
		fileIndex:     &fileIndex,
		filesTotal:    total,
		globalWritten: &globalWritten,
		globalTotal:   totalBytes,
	}
	var output []string
	c.downloadRecursiveHelper(remoteDir, localDir, batch, &output)
	return output
}

func (c *NFSClient) downloadRecursiveHelper(remoteDir, localDir string, batch *batchState, output *[]string) {
	if err := os.MkdirAll(localDir, 0755); err != nil {
		*output = append(*output, styles.ErrorStyle.Render(fmt.Sprintf("Error creating local dir: %v", err)))
		return
	}

	entries, err := c.mount.ReadDirPlus(remoteDir)
	if err != nil {
		*output = append(*output, styles.ErrorStyle.Render(fmt.Sprintf("Error reading remote dir: %v", err)))
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		if name == "." || name == ".." {
			continue
		}
		remotePath := path.Join(remoteDir, name)
		localPath := filepath.Join(localDir, name)
		if entry.IsDir() {
			c.downloadRecursiveHelper(remotePath, localPath, batch, output)
		} else {
			result := c.doDownloadFile(remotePath, localPath, name, batch)
			*output = append(*output, result...)
		}
	}
}

// countRemoteFilesAndSize recursively counts files and sums their sizes under remoteDir.
func countRemoteFilesAndSize(c *NFSClient, remoteDir string) (int, int64) {
	entries, err := c.mount.ReadDirPlus(remoteDir)
	if err != nil {
		return 0, 0
	}
	count, total := 0, int64(0)
	for _, entry := range entries {
		name := entry.Name()
		if name == "." || name == ".." {
			continue
		}
		if entry.IsDir() {
			subCount, subTotal := countRemoteFilesAndSize(c, path.Join(remoteDir, name))
			count += subCount
			total += subTotal
		} else {
			count++
			total += entry.Size()
		}
	}
	return count, total
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

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error creating destination directory: %v", err))}
	}

	entries, err := c.mount.ReadDirPlus(dir)
	if err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", err))}
	}

	// First pass: collect matched files and their sizes for global progress
	type matchedEntry struct{ remote, local, name string }
	var matched []matchedEntry
	var totalBytes int64
	for _, entry := range entries {
		name := entry.Name()
		ok, err := filepath.Match(pattern, name)
		if err != nil || !ok || entry.IsDir() {
			continue
		}
		matched = append(matched, matchedEntry{
			remote: path.Join(dir, name),
			local:  filepath.Join(destDir, name),
			name:   name,
		})
		totalBytes += entry.Size()
	}

	if len(matched) == 0 {
		return []string{styles.ErrorStyle.Render("No files matched pattern")}
	}

	// Build batch state for global progress
	fileIndex := 0
	globalWritten := int64(0)
	batch := &batchState{
		fileIndex:     &fileIndex,
		filesTotal:    len(matched),
		globalWritten: &globalWritten,
		globalTotal:   totalBytes,
	}

	var output []string
	for _, f := range matched {
		result := c.doDownloadFile(f.remote, f.local, f.name, batch)
		output = append(output, result...)
	}
	output = append(output, styles.SuccessStyle.Render(fmt.Sprintf("Downloaded %d file(s)", len(matched))))
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
	return c.doUploadFile(localPath, remotePath, filepath.Base(localPath), nil)
}

// doUploadFile is the shared implementation used by single-file and batch uploads.
// batch == nil means a standalone single-file transfer.
func (c *NFSClient) doUploadFile(localPath, remotePath, name string, batch *batchState) []string {
	var fileTotal int64
	if info, err := os.Stat(localPath); err == nil {
		fileTotal = info.Size()
	}

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

	cfg := progressWriterConfig{
		filename:  name,
		fileTotal: fileTotal,
	}
	if batch != nil {
		*batch.fileIndex++
		cfg.fileIndex     = *batch.fileIndex
		cfg.filesTotal    = batch.filesTotal
		cfg.globalWritten = batch.globalWritten
		cfg.globalTotal   = batch.globalTotal
	}

	pw := newProgressWriter(remoteFile, cfg, c.progressFn)
	written, err := io.Copy(pw, localFile)
	pw.flush()
	if err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Error uploading: %v", err))}
	}

	sizeStr := formatBytes(written)
	return []string{styles.SuccessStyle.Render(fmt.Sprintf("Uploaded: %s -> %s (%s)", localPath, finalRemotePath, sizeStr))}
}

func (c *NFSClient) uploadRecursive(localDir, remoteDir string) []string {
	// Pre-scan to get file count and total bytes for global progress
	total, totalBytes := countLocalFilesAndSize(localDir)
	fileIndex := 0
	globalWritten := int64(0)
	batch := &batchState{
		fileIndex:     &fileIndex,
		filesTotal:    total,
		globalWritten: &globalWritten,
		globalTotal:   totalBytes,
	}
	var output []string
	c.uploadRecursiveHelper(localDir, remoteDir, batch, &output)
	return output
}

func (c *NFSClient) uploadRecursiveHelper(localDir, remoteDir string, batch *batchState, output *[]string) {
	resolvedRemoteDir := c.resolvePath(remoteDir)
	_, _ = c.mount.Mkdir(resolvedRemoteDir, 0755) // ignore error; dir may already exist

	entries, err := os.ReadDir(localDir)
	if err != nil {
		*output = append(*output, styles.ErrorStyle.Render(fmt.Sprintf("Error reading local dir: %v", err)))
		return
	}

	for _, entry := range entries {
		localPath := filepath.Join(localDir, entry.Name())
		remotePath := path.Join(resolvedRemoteDir, entry.Name())
		if entry.IsDir() {
			c.uploadRecursiveHelper(localPath, remotePath, batch, output)
		} else {
			result := c.doUploadFile(localPath, remotePath, entry.Name(), batch)
			*output = append(*output, result...)
		}
	}
}

// countLocalFilesAndSize recursively counts files and sums their sizes under dir.
func countLocalFilesAndSize(dir string) (int, int64) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, 0
	}
	count, total := 0, int64(0)
	for _, e := range entries {
		if e.IsDir() {
			subCount, subTotal := countLocalFilesAndSize(filepath.Join(dir, e.Name()))
			count += subCount
			total += subTotal
		} else {
			count++
			if info, err := e.Info(); err == nil {
				total += info.Size()
			}
		}
	}
	return count, total
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

	globs, err := filepath.Glob(searchPattern)
	if err != nil {
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Invalid pattern: %v", err))}
	}
	if len(globs) == 0 {
		return []string{styles.ErrorStyle.Render("No files matched pattern")}
	}

	// First pass: filter to regular files and sum sizes
	type localFile struct{ path, name string }
	var files []localFile
	var totalBytes int64
	for _, lp := range globs {
		info, err := os.Stat(lp)
		if err != nil || info.IsDir() {
			continue
		}
		files = append(files, localFile{path: lp, name: filepath.Base(lp)})
		totalBytes += info.Size()
	}
	if len(files) == 0 {
		return []string{styles.ErrorStyle.Render("No files matched pattern")}
	}

	// Build batch state for global progress
	fileIndex := 0
	globalWritten := int64(0)
	batch := &batchState{
		fileIndex:     &fileIndex,
		filesTotal:    len(files),
		globalWritten: &globalWritten,
		globalTotal:   totalBytes,
	}

	var output []string
	for _, f := range files {
		finalRemote := path.Join(remotePath, f.name)
		result := c.doUploadFile(f.path, finalRemote, f.name, batch)
		output = append(output, result...)
	}
	output = append(output, styles.SuccessStyle.Render(fmt.Sprintf("Uploaded %d file(s)", len(files))))
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
