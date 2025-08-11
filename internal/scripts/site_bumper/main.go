package main

import (
	"fmt"
	"strings"

	t "github.com/river-now/river/kit/cliutil"
	"github.com/river-now/river/kit/parseutil"
)

func main() {
	// Read current version from package.json
	_, _, version := parseutil.PackageJSONFromFile("./package.json")

	t.Plain("Updating site packages to version ")
	t.Green(version)
	t.NewLine()

	// Determine if it's a pre-release
	isPre := strings.Contains(version, "pre")

	// Update Go module
	goVersion := fmt.Sprintf("v%s", version)
	t.Plain("Updating Go module to ")
	t.Green(goVersion)
	t.NewLine()

	cmd := t.Cmd("go", "get", fmt.Sprintf("github.com/river-now/river@%s", goVersion))
	cmd.Dir = "internal/site"
	t.MustRun(cmd, "failed to update Go module")

	cmd = t.Cmd("go", "mod", "tidy")
	cmd.Dir = "internal/site"
	t.MustRun(cmd, "failed to tidy Go module")

	// Update npm package
	npmTag := "latest"
	if isPre {
		npmTag = "pre"
	}

	t.Plain("Updating npm package with tag ")
	t.Green(npmTag)
	t.NewLine()

	cmd = t.Cmd("pnpm", "update", fmt.Sprintf("river.now@%s", npmTag))
	cmd.Dir = "internal/site"
	t.MustRun(cmd, "failed to update npm package")

	t.Green("✓ Site packages updated successfully")
	t.NewLine()
}
