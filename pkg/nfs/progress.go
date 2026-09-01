package nfs

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// ProgressUpdate holds all state for a single progress callback invocation.
type ProgressUpdate struct {
	Filename      string // bare display name (no [n/N] prefix — that's in FileIndex/FilesTotal)
	FileIndex     int    // 1-based; 0 = single-file transfer (no counter shown)
	FilesTotal    int    // total files in batch; 0 = single-file transfer
	FileWritten   int64  // bytes written for the current file
	FileTotal     int64  // total bytes for the current file (0 = unknown)
	GlobalWritten int64  // cumulative bytes written across all files in the batch
	GlobalTotal   int64  // total bytes across all files (0 = unknown)
}

// ProgressFunc is called periodically during a transfer.
type ProgressFunc func(update ProgressUpdate)

// progressWriterConfig carries all parameters for a progressWriter.
type progressWriterConfig struct {
	filename      string
	fileIndex     int    // 1-based; 0 = single file
	filesTotal    int    // 0 = single file
	fileTotal     int64
	globalWritten *int64 // shared pointer across files; nil = single file (use fileWritten)
	globalTotal   int64
}

// progressWriter wraps an io.Writer and fires a ProgressFunc on each chunk,
// throttled to at most one call per 50 ms.
type progressWriter struct {
	dst           io.Writer
	filename      string
	fileIndex     int
	filesTotal    int
	fileWritten   int64
	fileTotal     int64
	globalWritten *int64
	globalTotal   int64
	fn            ProgressFunc
	lastCall      time.Time
}

// ProgressLayout computes the label width and bar width for progress bars
// from the given terminal width. The caller provides the terminal width so
// that TUI and non-interactive modes can each query it appropriately.
//
// Layout breakdown (prefix "⬆ " or "  " = 2 cols):
//
//	[prefix 2] [label] ["  [" 3] [bar] ["]  " 3] ["100%" 4] ["  " 2] [size 22]
//
func ProgressLayout(termWidth int) (labelWidth, barWidth int) {
	const (
		minLabel = 12
		maxLabel = 36  // cap so wide terminals don't waste space on the label
		fixedBar = 20  // bar stays fixed; label absorbs the remaining slack
		// overhead = prefix(2) + "  ["(3) + "]  "(3) + "100%"(4) + "  "(2) + size(22)
		overhead = 2 + 3 + 3 + 4 + 2 + 22
	)
	lw := termWidth - overhead - fixedBar
	if lw < minLabel {
		lw = minLabel
	}
	if lw > maxLabel {
		lw = maxLabel
	}
	return lw, fixedBar
}

// TermWidth queries the real terminal width, falling back to 80.
func TermWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w < 20 {
		return 80
	}
	return w
}

func newProgressWriter(dst io.Writer, cfg progressWriterConfig, fn ProgressFunc) *progressWriter {
	return &progressWriter{
		dst:           dst,
		filename:      cfg.filename,
		fileIndex:     cfg.fileIndex,
		filesTotal:    cfg.filesTotal,
		fileTotal:     cfg.fileTotal,
		globalWritten: cfg.globalWritten,
		globalTotal:   cfg.globalTotal,
		fn:            fn,
		lastCall:      time.Now(),
	}
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n, err := pw.dst.Write(p)
	pw.fileWritten += int64(n)
	if pw.globalWritten != nil {
		*pw.globalWritten += int64(n)
	}
	if pw.fn != nil {
		now := time.Now()
		if now.Sub(pw.lastCall) >= 50*time.Millisecond {
			pw.fire()
			pw.lastCall = now
		}
	}
	return n, err
}

func (pw *progressWriter) fire() {
	if pw.fn == nil {
		return
	}
	gw := pw.fileWritten
	gt := pw.fileTotal
	if pw.globalWritten != nil {
		gw = *pw.globalWritten
		gt = pw.globalTotal
	}
	pw.fn(ProgressUpdate{
		Filename:      pw.filename,
		FileIndex:     pw.fileIndex,
		FilesTotal:    pw.filesTotal,
		FileWritten:   pw.fileWritten,
		FileTotal:     pw.fileTotal,
		GlobalWritten: gw,
		GlobalTotal:   gt,
	})
}

// flush fires the callback one final time (guarantees a 100% update).
func (pw *progressWriter) flush() {
	pw.fire()
}

// RenderFileProgress renders a per-file progress line using the given label
// and bar widths (from ProgressLayout). No leading spaces — the caller
// provides the arrow/indent prefix.
//
//	[3/5] testbash2                [████████░░░░░░░░░░░░]  39%  544.00 KB / 1.38 MB
func RenderFileProgress(u ProgressUpdate, labelWidth, barWidth int) string {
	var label string
	if u.FilesTotal > 1 && u.FileIndex > 0 {
		counter := fmt.Sprintf("[%d/%d] ", u.FileIndex, u.FilesTotal)
		nameWidth := labelWidth - len(counter)
		if nameWidth < 6 {
			nameWidth = 6
		}
		label = fmt.Sprintf("%s%-*s", counter, nameWidth, truncateFilename(u.Filename, nameWidth))
	} else {
		label = fmt.Sprintf("%-*s", labelWidth, truncateFilename(u.Filename, labelWidth))
	}

	sizeStr := formatBytes(u.FileWritten)
	if u.FileTotal > 0 {
		sizeStr = fmt.Sprintf("%s / %s", formatBytes(u.FileWritten), formatBytes(u.FileTotal))
	}

	if u.FileTotal <= 0 {
		dots := strings.Repeat("·", barWidth)
		return fmt.Sprintf("%s  [%s]  %s", label, dots, sizeStr)
	}

	pct := float64(u.FileWritten) / float64(u.FileTotal)
	if pct > 1 {
		pct = 1
	}
	filled := int(float64(barWidth) * pct)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	return fmt.Sprintf("%s  [%s]  %3.0f%%  %s", label, bar, pct*100, sizeStr)
}

// RenderGlobalProgress renders the overall batch progress line using the same
// label and bar widths as RenderFileProgress, guaranteeing column alignment.
// Returns "" for single-file transfers.
//
//	Total (5 files)                [██████████░░░░░░░░░░]  48%  3.29 MB / 6.90 MB
func RenderGlobalProgress(u ProgressUpdate, labelWidth, barWidth int) string {
	if u.FilesTotal <= 1 {
		return ""
	}

	raw := fmt.Sprintf("Total (%d files)", u.FilesTotal)
	label := fmt.Sprintf("%-*s", labelWidth, truncateFilename(raw, labelWidth))

	sizeStr := formatBytes(u.GlobalWritten)
	if u.GlobalTotal > 0 {
		sizeStr = fmt.Sprintf("%s / %s", formatBytes(u.GlobalWritten), formatBytes(u.GlobalTotal))
	}

	if u.GlobalTotal <= 0 {
		dots := strings.Repeat("·", barWidth)
		return fmt.Sprintf("%s  [%s]  %s", label, dots, sizeStr)
	}

	pct := float64(u.GlobalWritten) / float64(u.GlobalTotal)
	if pct > 1 {
		pct = 1
	}
	filled := int(float64(barWidth) * pct)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	return fmt.Sprintf("%s  [%s]  %3.0f%%  %s", label, bar, pct*100, sizeStr)
}

func truncateFilename(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
