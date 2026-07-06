package leftoff

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompatibilityReportIncludesEnvironment(t *testing.T) {
	result := CompatibilityReport("")
	assertContains(t, result.Output, "COMPATIBILITY")
	assertContains(t, result.Output, "Agent skill targets")
	assertContains(t, result.Output, "Codex")
	assertContains(t, result.Output, "Claude Code")
	assertContains(t, result.Output, "Cursor")
	assertContains(t, result.Output, "GitHub Copilot CLI")
	assertContains(t, result.Output, "Gemini CLI / Antigravity")
	assertContains(t, result.Output, "Hermes Agent")
	assertContains(t, result.Output, "OpenClaw")
	assertContains(t, result.Output, "Windsurf")
	assertContains(t, result.Output, "Zed")
	assertContains(t, result.Output, "Core workflow: no network required")
}

func TestInstallerScriptsDocumentDryRun(t *testing.T) {
	for _, path := range []string{
		filepath.Join("..", "..", "install.sh"),
		filepath.Join("..", "..", "install.ps1"),
	} {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(content)
		assertContains(t, text, "Dry run")
		if strings.HasSuffix(path, ".sh") {
			assertContains(t, text, "--agent")
		} else {
			assertContains(t, text, "Agent")
		}
	}
}

func TestExportImportAndDeleteDataFlow(t *testing.T) {
	store := fixedStore(t)
	if _, err := store.Capture(context.Background(), CaptureRequest{Project: "sample", Text: "task: Export test"}); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), "export.zip")
	exported, err := store.Export(ExportRequest{Out: out})
	if err != nil {
		t.Fatal(err)
	}
	if exported.Files == 0 {
		t.Fatalf("expected exported files")
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("expected export archive: %v", err)
	}

	importStore := fixedStore(t)
	if _, err := importStore.Import(ImportRequest{From: out}); err == nil {
		t.Fatalf("import without confirmation should fail")
	}
	if _, err := importStore.Import(ImportRequest{From: out, Confirm: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(importStore.Root, "projects", "sample", "open-loops.md")); err != nil {
		t.Fatalf("expected imported project record: %v", err)
	}

	dryRun, err := importStore.DeleteData(DeleteDataRequest{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, dryRun.Output, "Dry run")
	if _, err := os.Stat(importStore.Root); err != nil {
		t.Fatalf("dry run should retain store: %v", err)
	}
	if _, err := importStore.DeleteData(DeleteDataRequest{}); err == nil {
		t.Fatalf("delete without confirmation should fail")
	}
	if _, err := importStore.DeleteData(DeleteDataRequest{Confirm: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(importStore.Root); !os.IsNotExist(err) {
		t.Fatalf("expected store deletion")
	}
}

func TestImportRejectsPathTraversal(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "bad.zip")
	file, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	entry, err := writer.Create("../evil.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("bad")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	store := fixedStore(t)
	if _, err := store.Import(ImportRequest{From: archive, Confirm: true}); err == nil {
		t.Fatalf("expected traversal import rejection")
	}
}

func TestFixturePrivacyNoLikelySecrets(t *testing.T) {
	root := filepath.Join("..", "..", "tests", "fixtures")
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if findings := FindSecrets(string(content)); len(findings) > 0 {
			return os.ErrInvalid
		}
		if strings.Contains(strings.ToLower(string(content)), "password=") {
			return os.ErrInvalid
		}
		return nil
	})
	if err != nil {
		t.Fatalf("fixture privacy check failed: %v", err)
	}
}
