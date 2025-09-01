package repoconcat

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/bmatcuk/doublestar/v4"
)

type Config struct {
	Root        string   // Directory to scan
	Output      string   // Output file path
	IgnoreDirs  []string // Directory patterns to skip entirely
	IgnoreFiles []string // File patterns to skip
	Verbose     bool     // Log included files and folders
}

// isTextFile checks if a file is valid UTF-8 text
func isTextFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	// Read first 8KB to check encoding
	buf := make([]byte, 8192)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return false
	}

	// Check if it's valid UTF-8
	return utf8.Valid(buf[:n])
}

// matchesPattern checks if a path matches a glob pattern
func matchesPattern(pattern, path string) bool {
	matched, _ := doublestar.Match(pattern, path)
	return matched
}

// Concat concatenates all files according to config
func Concat(cfg Config) error {
	// Create output file
	outFile, err := os.Create(cfg.Output)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer outFile.Close()

	writer := bufio.NewWriter(outFile)
	defer writer.Flush()

	// Get absolute paths for comparison
	absOutput, _ := filepath.Abs(cfg.Output)

	// Default patterns that should match anywhere in the tree
	defaultIgnoreDirs := []string{
		"**/node_modules",
		"**/.git",
		"**/.vscode",
	}

	defaultIgnoreFiles := []string{
		"**/.DS_Store",
		"**/.gitignore",
		"**/.gitignore.local",
		"**/*.svg",
		"**/go.sum",
		"**/package-lock.json",
		"**/yarn.lock",
		"**/pnpm-lock.yaml",
		"**/bun.lockb",
	}

	// Cache for gitignore patterns by directory
	gitignoreCache := make(map[string][]string)

	// Statistics
	var includedFiles, skippedBinary int

	err = filepath.Walk(cfg.Root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the output file itself
		absPath, _ := filepath.Abs(path)
		if absPath == absOutput {
			return nil
		}

		// Get relative path and normalize to forward slashes
		relPath, _ := filepath.Rel(cfg.Root, path)
		relPath = filepath.ToSlash(relPath)

		// Handle directories
		if info.IsDir() {
			// Check user-provided directory patterns
			for _, pattern := range cfg.IgnoreDirs {
				if matchesPattern(filepath.ToSlash(pattern), relPath) {
					return filepath.SkipDir
				}
			}

			// Check default directory patterns
			for _, pattern := range defaultIgnoreDirs {
				if matchesPattern(pattern, relPath) {
					return filepath.SkipDir
				}
			}

			// Check gitignore for directory patterns
			patterns := getGitignorePatterns(path, cfg.Root, gitignoreCache)
			for _, pattern := range patterns {
				// Directory patterns from gitignore
				if strings.HasSuffix(pattern, "/**") {
					dirPattern := strings.TrimSuffix(pattern, "/**")
					if matchesPattern(dirPattern, relPath) {
						return filepath.SkipDir
					}
				}
			}

			return nil
		}

		// Handle files

		// Check user-provided file patterns
		for _, pattern := range cfg.IgnoreFiles {
			if matchesPattern(filepath.ToSlash(pattern), relPath) {
				return nil
			}
		}

		// Check default file patterns
		for _, pattern := range defaultIgnoreFiles {
			if matchesPattern(pattern, relPath) {
				return nil
			}
		}

		// Check gitignore for file patterns
		patterns := getGitignorePatterns(path, cfg.Root, gitignoreCache)
		for _, pattern := range patterns {
			if matchesPattern(pattern, relPath) {
				return nil
			}
		}

		// Skip binary files
		if !isTextFile(path) {
			skippedBinary++
			return nil
		}

		// Include this file
		if cfg.Verbose {
			includedFiles++
			fmt.Printf("[FILE] %s (%.2f KB)\n", relPath, float64(info.Size())/1024)
		}

		// Write file header
		fmt.Fprintf(writer, "\n%s\n", strings.Repeat("=", 80))
		fmt.Fprintf(writer, "FILE: %s\n", relPath)
		fmt.Fprintf(writer, "%s\n\n", strings.Repeat("=", 80))

		// Copy file contents
		file, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(writer, "[ERROR READING FILE: %v]\n", err)
			if cfg.Verbose {
				fmt.Printf("  ERROR: Could not read file: %v\n", err)
			}
			return nil
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		if err != nil {
			fmt.Fprintf(writer, "\n[ERROR COPYING FILE: %v]\n", err)
			if cfg.Verbose {
				fmt.Printf("  ERROR: Could not copy file: %v\n", err)
			}
		}

		fmt.Fprintln(writer)
		return nil
	})

	if cfg.Verbose && err == nil {
		fmt.Printf("\nSummary: %d files included, %d binary files skipped\n",
			includedFiles, skippedBinary)
	}

	return err
}

// getGitignorePatterns returns all applicable gitignore patterns for a path
func getGitignorePatterns(path, root string, cache map[string][]string) []string {
	var allPatterns []string

	// Walk up from path to root, collecting patterns
	dir := filepath.Dir(path)
	for {
		// Check cache first
		if patterns, exists := cache[dir]; exists {
			allPatterns = append(allPatterns, patterns...)
		} else {
			var dirPatterns []string

			// Load .gitignore
			gitignorePath := filepath.Join(dir, ".gitignore")
			if patterns := loadGitignoreFile(gitignorePath, dir, root); patterns != nil {
				dirPatterns = append(dirPatterns, patterns...)
			}

			// Load .gitignore.local
			gitignoreLocalPath := filepath.Join(dir, ".gitignore.local")
			if patterns := loadGitignoreFile(gitignoreLocalPath, dir, root); patterns != nil {
				dirPatterns = append(dirPatterns, patterns...)
			}

			// Cache the patterns for this directory
			if len(dirPatterns) > 0 {
				cache[dir] = dirPatterns
				allPatterns = append(allPatterns, dirPatterns...)
			}
		}

		// Stop at root
		if dir == root || dir == filepath.Dir(dir) {
			break
		}
		dir = filepath.Dir(dir)
	}

	return allPatterns
}

// loadGitignoreFile loads and converts patterns from a gitignore file
func loadGitignoreFile(path, gitignoreDir, root string) []string {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	var patterns []string
	scanner := bufio.NewScanner(file)

	// Get the relative path from root to the gitignore directory
	relDir, _ := filepath.Rel(root, gitignoreDir)
	relDir = filepath.ToSlash(relDir)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Skip negation patterns (not supported)
		if strings.HasPrefix(line, "!") {
			continue
		}

		pattern := line

		// Handle directory-only patterns (ending with /)
		isDir := strings.HasSuffix(pattern, "/")
		if isDir {
			pattern = strings.TrimSuffix(pattern, "/")
		}

		// Handle absolute patterns (starting with /)
		if strings.HasPrefix(pattern, "/") {
			// Absolute to the gitignore location
			pattern = strings.TrimPrefix(pattern, "/")
			if relDir != "" && relDir != "." {
				pattern = relDir + "/" + pattern
			}
		} else {
			// Relative patterns can match anywhere below gitignore location
			if relDir != "" && relDir != "." {
				pattern = relDir + "/**/" + pattern
			} else {
				pattern = "**/" + pattern
			}
		}

		// For directory patterns, append /** to match everything inside
		if isDir {
			patterns = append(patterns, pattern+"/**")
		}

		// Add the pattern itself (matches both files and dirs with that name)
		patterns = append(patterns, pattern)
	}

	return patterns
}
