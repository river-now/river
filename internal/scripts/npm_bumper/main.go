package main

import (
	"os"
	"strings"

	t "github.com/river-now/river/kit/cliutil"
	"github.com/river-now/river/kit/parseutil"
)

func main() {
	lines, versionLine, currentVersion := parseutil.PackageJSONFromFile("./package.json")

	// Show current tag
	t.Plain("current version: ")
	t.Green(currentVersion)
	t.NewLine()

	// Ask for new version
	t.Blue("what is the new version? ")
	version, err := t.NewReader().ReadString('\n')
	if err != nil {
		t.Exit("failed to read version", err)
	}

	trimmedVersion := strings.TrimSpace(version)
	if trimmedVersion == "" {
		t.Exit("version is empty", nil)
	}

	// Show new tag
	t.Plain("Result: ")
	t.Red(currentVersion)
	t.Plain("  -->  ")
	t.Green(trimmedVersion)
	t.NewLine()

	// Ask for confirmation
	t.Blue("is this correct? ")
	t.RequireYes("aborted")

	lines[versionLine] = strings.Replace(lines[versionLine], currentVersion, trimmedVersion, 1)

	// Ask for write confirmation
	t.Blue("write new version ")
	t.Green(trimmedVersion)
	t.Blue(" to package.json? ")
	t.RequireYes("aborted")

	// Write the new version to the file
	if err = os.WriteFile("./package.json", []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		t.Exit("failed to write file", err)
	}

	// Sanity check
	_, _, newCurrentVersion := parseutil.PackageJSONFromFile("./package.json")
	if newCurrentVersion != trimmedVersion {
		t.Exit("failed to update version", nil)
	}

	isPre := strings.Contains(newCurrentVersion, "pre")

	if isPre {
		t.Plain("pre-release version detected")
		t.NewLine()
	}

	// Ask whether to initiate a new build?
	t.Blue("emit a new build to ./npm_dist?  ")
	t.RequireYes("aborted")

	cmd := t.Cmd("make", "npmbuild")
	t.MustRun(cmd, "npm dist build failed")

	// Ask for publish confirmation
	t.Blue("do you want to publish ")
	if isPre {
		t.Red("PRE release ")
	} else {
		t.Red("FINAL release ")
	}
	t.Green(trimmedVersion)
	t.Blue(" npm? ")
	t.RequireYes("aborted")

	cmd = t.Cmd("make", "tspublishpre")
	if !isPre {
		cmd = t.Cmd("make", "tspublishnonpre")
	}

	t.MustRun(cmd, "npm publish failed")

	t.Plain("npm publish done")
	t.NewLine()
}
