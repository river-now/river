package main

import (
	"log"

	"github.com/river-now/river/kit/scripts/repoconcat"
)

func main() {
	cfg := repoconcat.Config{
		Root:       "./internal/framework",
		Output:     "LLM__INTERNAL_FRAMEWORK.local.txt",
		IgnoreDirs: []string{},
		IgnoreFiles: []string{
			"**/*_test.go",
			"**/*.test.*",
			"bench.txt",
			"**/*.bench.*",
			"**/*.local.*",
			"**/*.bench.txt",
		},
		Verbose: true,
	}

	if err := repoconcat.Concat(cfg); err != nil {
		log.Fatal(err)
	}

	cfg = repoconcat.Config{
		Root:       "./wave",
		Output:     "LLM__WAVE.local.txt",
		IgnoreDirs: []string{},
		IgnoreFiles: []string{
			"**/*_test.go",
			"**/*.test.*",
			"bench.txt",
			"**/*.bench.*",
			"**/*.local.*",
			"**/*.bench.txt",
		},
		Verbose: true,
	}

	if err := repoconcat.Concat(cfg); err != nil {
		log.Fatal(err)
	}

	cfg = repoconcat.Config{
		Root:       "./kit/matcher",
		Output:     "LLM__KIT_MATCHER.local.txt",
		IgnoreDirs: []string{},
		IgnoreFiles: []string{
			// "**/*_test.go",
			// "**/*.test.*",
			"bench.txt",
			"**/*.bench.*",
			"**/*.local.*",
			"**/*.bench.txt",
		},
		Verbose: true,
	}

	if err := repoconcat.Concat(cfg); err != nil {
		log.Fatal(err)
	}

	cfg = repoconcat.Config{
		Root:       "./kit/validate",
		Output:     "LLM__KIT_VALIDATE.local.txt",
		IgnoreDirs: []string{},
		IgnoreFiles: []string{
			"**/*_test.go",
			"**/*.test.*",
			"bench.txt",
			"**/*.bench.*",
			"**/*.local.*",
			"**/*.bench.txt",
		},
		Verbose: true,
	}

	if err := repoconcat.Concat(cfg); err != nil {
		log.Fatal(err)
	}

	cfg = repoconcat.Config{
		Root:       "./kit/tasks",
		Output:     "LLM__KIT_TASKS.local.txt",
		IgnoreDirs: []string{},
		IgnoreFiles: []string{
			"**/*_test.go",
			"**/*.test.*",
			"bench.txt",
			"**/*.bench.*",
			"**/*.local.*",
			"**/*.bench.txt",
		},
		Verbose: true,
	}

	if err := repoconcat.Concat(cfg); err != nil {
		log.Fatal(err)
	}

	cfg = repoconcat.Config{
		Root:       "./kit/mux",
		Output:     "LLM__KIT_MUX.local.txt",
		IgnoreDirs: []string{},
		IgnoreFiles: []string{
			"**/*_test.go",
			"**/*.test.*",
			"bench.txt",
			"**/*.bench.*",
			"**/*.local.*",
			"**/*.bench.txt",
		},
		Verbose: true,
	}

	if err := repoconcat.Concat(cfg); err != nil {
		log.Fatal(err)
	}

	cfg = repoconcat.Config{
		Root:       "./kit/response",
		Output:     "LLM__KIT_RESPONSE.local.txt",
		IgnoreDirs: []string{},
		IgnoreFiles: []string{
			"**/*_test.go",
			"**/*.test.*",
			"bench.txt",
			"**/*.bench.*",
			"**/*.local.*",
			"**/*.bench.txt",
		},
		Verbose: true,
	}

	if err := repoconcat.Concat(cfg); err != nil {
		log.Fatal(err)
	}

	cfg = repoconcat.Config{
		Root:       "./kit/viteutil",
		Output:     "LLM__KIT_VITEUTIL.local.txt",
		IgnoreDirs: []string{},
		IgnoreFiles: []string{
			"**/*_test.go",
			"**/*.test.*",
			"bench.txt",
			"**/*.bench.*",
			"**/*.local.*",
			"**/*.bench.txt",
		},
		Verbose: true,
	}

	if err := repoconcat.Concat(cfg); err != nil {
		log.Fatal(err)
	}
}
