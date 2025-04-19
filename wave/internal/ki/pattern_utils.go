package ki

import (
	"fmt"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
)

type potentialMatch struct {
	pattern string
	path    string
}

func (c *Config) match_results_key_maker(k potentialMatch) string {
	return k.pattern + k.path
}

func (c *Config) get_initial_match_results(k potentialMatch) (bool, error) {
	normalizedPath := filepath.ToSlash(k.path)

	matches, err := doublestar.Match(k.pattern, normalizedPath)
	if err != nil {
		c.Logger.Error(fmt.Sprintf("error: failed to match file: %v", err))
		return false, err
	}

	return matches, nil
}

func (c *Config) get_is_match(k potentialMatch) bool {
	isMatch, _ := c.matchResults.Get(k)
	return isMatch
}

func (c *Config) get_is_ignored(path string, ignoredPatterns []string) bool {
	for _, pattern := range ignoredPatterns {
		if c.get_is_match(potentialMatch{pattern: pattern, path: path}) {
			return true
		}
	}
	return false
}
