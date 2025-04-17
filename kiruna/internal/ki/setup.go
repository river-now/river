package ki

import (
	"fmt"
	"os"
)

func (c *Config) SetupDistDir() error {
	// make a dist/static/internal directory
	path := c._dist.S().Static.S().Internal.FullPath()
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("error making internal directory: %w", err)
	}

	// add an empty file so that go:embed doesn't complain
	path = c._dist.S().Static.S().Keep.FullPath()
	if err := os.WriteFile(path, []byte("//go:embed directives require at least one file to compile\n"), 0644); err != nil {
		return fmt.Errorf("error making x file: %w", err)
	}

	// make an empty dist/static/assets/public/internal directory
	path = c._dist.S().Static.S().Assets.S().Public.S().PublicInternal.FullPath()
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("error making public directory: %w", err)
	}

	// make an empty dist/static/assets/private directory
	path = c._dist.S().Static.S().Assets.S().Private.FullPath()
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("error making private directory: %w", err)
	}

	return nil
}
