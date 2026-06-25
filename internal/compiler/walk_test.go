package compiler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xoai/sage-wiki/internal/config"
	"github.com/xoai/sage-wiki/internal/manifest"
	"github.com/xoai/sage-wiki/internal/wiki"
)

func TestWalkSourceDir_SymlinkedDirectory(t *testing.T) {
	dir := t.TempDir()

	// Real source directory outside the project
	realDir := filepath.Join(dir, "realsrc")
	os.MkdirAll(realDir, 0755)
	os.WriteFile(filepath.Join(realDir, "page1.md"), []byte("# Page 1"), 0644)
	os.WriteFile(filepath.Join(realDir, "page2.md"), []byte("# Page 2"), 0644)

	// Symlink inside a project
	projectDir := filepath.Join(dir, "project")
	os.MkdirAll(projectDir, 0755)
	linkPath := filepath.Join(projectDir, "raw")
	if err := os.Symlink(realDir, linkPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	var paths []string
	err := WalkSourceDir(linkPath, func(absPath, relPath string, _ os.DirEntry) error {
		paths = append(paths, relPath)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkSourceDir: %v", err)
	}

	if len(paths) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(paths), paths)
	}
}

func TestWalkSourceDir_PlainDirectory(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "file.md"), []byte("hello"), 0644)

	var paths []string
	err := WalkSourceDir(dir, func(absPath, relPath string, _ os.DirEntry) error {
		paths = append(paths, relPath)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkSourceDir: %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("expected 1 file, got %d", len(paths))
	}
	if paths[0] != "file.md" {
		t.Errorf("expected file.md, got %s", paths[0])
	}
}

func TestDiff_DiscoversFilesInSymlinkedSource(t *testing.T) {
	dir := t.TempDir()

	// Init a sage-wiki project
	projectDir := filepath.Join(dir, "wiki")
	if err := wiki.InitGreenfield(projectDir, "test", "gemini-2.5-flash"); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Create real source dir with files and symlink it in as "raw"
	realDir := filepath.Join(dir, "realsrc")
	os.MkdirAll(realDir, 0755)
	os.WriteFile(filepath.Join(realDir, "a.md"), []byte("# A"), 0644)
	os.WriteFile(filepath.Join(realDir, "b.md"), []byte("# B"), 0644)

	// Remove default raw dir, replace with symlink
	rawPath := filepath.Join(projectDir, "raw")
	os.RemoveAll(rawPath)
	if err := os.Symlink(realDir, rawPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	cfg, _ := config.Load(filepath.Join(projectDir, "config.yaml"))
	mf, _ := manifest.Load(filepath.Join(projectDir, ".manifest.json"))

	diff, err := Diff(projectDir, cfg, mf)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}

	if len(diff.Added) != 2 {
		t.Fatalf("expected 2 added, got %d", len(diff.Added))
	}

	// Manifest paths must use the source name, not the real path
	for _, s := range diff.Added {
		if !filepath.HasPrefix(s.Path, "raw/") {
			t.Errorf("manifest path %q should start with raw/", s.Path)
		}
	}

	// Diff should be idempotent — mark as compiled then re-diff
	mf.AddSource("raw/a.md", diff.Added[0].Hash, "article", 3)
	mf.AddSource("raw/b.md", diff.Added[1].Hash, "article", 3)
	for _, s := range diff.Added {
		mf.MarkCompiled(s.Path, "wiki/summaries/"+filepath.Base(s.Path), nil)
	}
	mf.Save(filepath.Join(projectDir, ".manifest.json"))

	mf2, _ := manifest.Load(filepath.Join(projectDir, ".manifest.json"))
	diff2, _ := Diff(projectDir, cfg, mf2)
	if len(diff2.Added) != 0 || len(diff2.Modified) != 0 || len(diff2.Removed) != 0 {
		t.Errorf("diff after compile should be empty, got +%d ~%d -%d",
			len(diff2.Added), len(diff2.Modified), len(diff2.Removed))
	}
}
