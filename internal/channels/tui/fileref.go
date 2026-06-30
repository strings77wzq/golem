package tui

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var fileRefPattern = regexp.MustCompile(`(?:^|\s)@(\S+)`)

const maxFileSize = 50 * 1024 // 50KB

// resolveFileRefs replaces @path references with file contents using the working directory.
func (m Model) resolveFileRefs(text string) string {
	wd, _ := os.Getwd()
	return m.resolveFileRefsWithDir(text, wd)
}

// resolveFileRefsWithDir replaces @path references with file contents using the given base directory.
func (m Model) resolveFileRefsWithDir(text string, baseDir string) string {
	if !strings.Contains(text, "@") {
		return text
	}

	matches := fileRefPattern.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return text
	}

	var result strings.Builder
	lastEnd := 0

	for _, match := range matches {
		// match[0] = full match start, match[1] = full match end
		// match[2] = capture group start, match[3] = capture group end
		fullStart := match[0]
		fullEnd := match[1]
		captureStart := match[2]
		captureEnd := match[3]

		path := text[captureStart:captureEnd]

		// Append text before this match (preserving the leading space if any)
		result.WriteString(text[lastEnd:fullStart])

		content, isBinary, err := readFileContent(path, baseDir)
		if err != nil || isBinary {
			// File not readable or binary — preserve original @path
			result.WriteString(text[fullStart:fullEnd])
		} else {
			result.WriteString(content)
		}

		lastEnd = fullEnd
	}

	result.WriteString(text[lastEnd:])
	return result.String()
}

// readFileContent reads a file and returns a formatted attachment string.
// Returns ("", nil) for binary files to signal "skip but no error".
// Returns ("", err) for read failures to signal "file not found".
func readFileContent(path, baseDir string) (string, bool, error) {
	// Expand tilde
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", false, err
		}
		path = filepath.Join(home, path[1:])
	} else if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}

	data, err := os.ReadFile(path) // #nosec G304 -- CLI tool, file path from user input
	if err != nil {
		return "", false, err
	}

	// Check for binary content
	if isBinaryFile(data) {
		return "", true, nil // binary, skip
	}

	// Truncate large files
	content := string(data)
	fileSize := len(data)
	if fileSize > maxFileSize {
		content = content[:maxFileSize] + "\n[truncated — file too large]"
	}

	// Get display path (relative if possible)
	displayPath := path
	if rel, err := filepath.Rel(baseDir, path); err == nil && len(rel) < len(path) {
		displayPath = rel
	}

	result := "\n<file path=\"" + displayPath + "\" size=\"" + itoa(fileSize) + "\">\n" + content + "\n</file>\n"
	return result, false, nil
}

// isBinaryFile checks if data contains null bytes (simple binary detection).
func isBinaryFile(data []byte) bool {
	checkLen := len(data)
	if checkLen > 512 {
		checkLen = 512
	}
	for i := 0; i < checkLen; i++ {
		if data[i] == 0 {
			return true
		}
	}
	return false
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
