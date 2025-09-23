package ki

import (
	"io/fs"
	"os"
	"testing"
)

func TestGetPublicFS(t *testing.T) {
	env := setupTestEnv(t)
	defer teardownTestEnv(t)

	env.createTestFile(t, "dist/static/assets/public/test.txt", "public fs content")

	publicFS, err := env.config.GetPublicFS()
	if err != nil {
		t.Fatalf("GetPublicFS() error = %v", err)
	}

	content, err := fs.ReadFile(publicFS, "test.txt")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(content) != "public fs content" {
		t.Errorf("ReadFile() content = %v, want %v", string(content), "public fs content")
	}
}

func TestGetPrivateFS(t *testing.T) {
	env := setupTestEnv(t)
	defer teardownTestEnv(t)

	env.createTestFile(t, "dist/static/assets/private/test.txt", "private fs content")

	privateFS, err := env.config.GetPrivateFS()
	if err != nil {
		t.Fatalf("GetPrivateFS() error = %v", err)
	}

	content, err := fs.ReadFile(privateFS, "test.txt")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(content) != "private fs content" {
		t.Errorf("ReadFile() content = %v, want %v", string(content), "private fs content")
	}
}

func TestGetBaseFS(t *testing.T) {
	env := setupTestEnv(t)
	defer teardownTestEnv(t)

	env.createTestFile(t, "dist/static/assets/public/test.txt", "base fs content")

	baseFS, err := env.config.GetBaseFS()
	if err != nil {
		t.Fatalf("GetBaseFS() error = %v", err)
	}

	content, err := fs.ReadFile(baseFS, "assets/public/test.txt")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(content) != "base fs content" {
		t.Errorf("ReadFile() content = %v, want %v", string(content), "base fs content")
	}

	// Test Sub
	subFS, err := fs.Sub(baseFS, "assets/public")
	if err != nil {
		t.Fatalf("Sub() error = %v", err)
	}

	content, err = fs.ReadFile(subFS, "test.txt")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(content) != "base fs content" {
		t.Errorf("ReadFile() content = %v, want %v", string(content), "base fs content")
	}

	// Test ReadDir
	entries, err := fs.ReadDir(baseFS, "assets")
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}

	expectedDirs := map[string]bool{PUBLIC: true, PRIVATE: true}
	for _, entry := range entries {
		if entry.IsDir() {
			if !expectedDirs[entry.Name()] {
				t.Errorf("Unexpected directory: %s", entry.Name())
			}
			delete(expectedDirs, entry.Name())
		}
	}

	if len(expectedDirs) > 0 {
		t.Errorf("Missing expected directories: %v", expectedDirs)
	}
}

func TestFSEdgeCases(t *testing.T) {
	env := setupTestEnv(t)
	defer teardownTestEnv(t)

	t.Run("NonexistentFile", func(t *testing.T) {
		baseFS, _ := env.config.GetBaseFS()
		_, err := fs.ReadFile(baseFS, "nonexistent.txt")
		if !os.IsNotExist(err) {
			t.Errorf("Expected os.IsNotExist(err) to be true for nonexistent file, got %v", err)
		}
	})

	t.Run("EmptyFile", func(t *testing.T) {
		env.createTestFile(t, "dist/static/assets/public/empty.txt", "")
		baseFS, _ := env.config.GetBaseFS()
		content, err := fs.ReadFile(baseFS, "assets/public/empty.txt")
		if err != nil {
			t.Errorf("Unexpected error reading empty file: %v", err)
		}
		if len(content) != 0 {
			t.Errorf("Expected empty content, got %d bytes", len(content))
		}
	})
}

func TestGetIsUsingEmbeddedFS(t *testing.T) {
	env := setupTestEnv(t)
	defer teardownTestEnv(t)

	// Test when DistFS is set (default in setupTestEnv)
	if !env.config.get_is_using_embedded_fs() {
		t.Errorf("getIsUsingEmbeddedFS() = false, want true when DistFS is set")
	}

	// Test when DistFS is nil
	env.config.DistStaticFS = nil
	if env.config.get_is_using_embedded_fs() {
		t.Errorf("getIsUsingEmbeddedFS() = true, want false when DistFS is nil")
	}
}
