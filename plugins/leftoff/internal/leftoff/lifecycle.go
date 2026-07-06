package leftoff

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const DataFormatVersion = "1"

var LeftoffVersion = "1.0.0"

type ExportRequest struct {
	Out string
}

type ImportRequest struct {
	From    string
	Confirm bool
}

type DeleteDataRequest struct {
	Confirm bool
	DryRun  bool
}

type LifecycleResult struct {
	Output string
	Path   string
	Files  int
}

type exportManifest struct {
	ToolVersion       string `json:"tool_version"`
	DataFormatVersion string `json:"data_format_version"`
	CreatedAt         string `json:"created_at"`
}

func (s *Store) Export(req ExportRequest) (LifecycleResult, error) {
	if err := s.Init(); err != nil {
		return LifecycleResult{}, err
	}
	out := strings.TrimSpace(req.Out)
	if out == "" {
		out = filepath.Join(s.Root, "backups", "leftoff-export-"+s.now().Format("20060102T150405")+".zip")
	}
	outAbs, err := filepath.Abs(out)
	if err != nil {
		return LifecycleResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(outAbs), 0o700); err != nil {
		return LifecycleResult{}, err
	}

	file, err := os.OpenFile(outAbs, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return LifecycleResult{}, err
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	manifest := exportManifest{
		ToolVersion:       LeftoffVersion,
		DataFormatVersion: DataFormatVersion,
		CreatedAt:         s.now().Format(timeFormatRFC3339()),
	}
	manifestData, _ := json.MarshalIndent(manifest, "", "  ")
	manifestEntry, err := zipWriter.Create(".leftoff-export-manifest.json")
	if err != nil {
		return LifecycleResult{}, err
	}
	if _, err := manifestEntry.Write(append(manifestData, '\n')); err != nil {
		return LifecycleResult{}, err
	}

	count := 1
	err = filepath.WalkDir(s.Root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		if filepath.Clean(path) == filepath.Clean(outAbs) {
			return nil
		}
		rel, err := filepath.Rel(s.Root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == ".leftoff-export-manifest.json" {
			return nil
		}
		writer, err := zipWriter.Create(rel)
		if err != nil {
			return err
		}
		source, err := os.Open(path)
		if err != nil {
			return err
		}
		defer source.Close()
		if _, err := io.Copy(writer, source); err != nil {
			return err
		}
		count++
		return nil
	})
	if err != nil {
		return LifecycleResult{}, err
	}

	output := fmt.Sprintf("EXPORT\n- Archive: %s\n- Files: %d\n- Data format: %s\n", outAbs, count, DataFormatVersion)
	return LifecycleResult{Output: output, Path: outAbs, Files: count}, nil
}

func (s *Store) Import(req ImportRequest) (LifecycleResult, error) {
	if strings.TrimSpace(req.From) == "" {
		return LifecycleResult{}, fmt.Errorf("import requires --from")
	}
	if !req.Confirm {
		return LifecycleResult{}, fmt.Errorf("import requires --confirm")
	}
	if err := s.Init(); err != nil {
		return LifecycleResult{}, err
	}

	source, err := zip.OpenReader(req.From)
	if err != nil {
		return LifecycleResult{}, err
	}
	defer source.Close()

	count := 0
	for _, file := range source.File {
		if file.FileInfo().IsDir() {
			continue
		}
		rel, err := cleanZipPath(file.Name)
		if err != nil {
			return LifecycleResult{}, err
		}
		if rel == ".leftoff-export-manifest.json" {
			continue
		}
		target := filepath.Join(s.Root, filepath.FromSlash(rel))
		if err := s.requireInsideRoot(target); err != nil {
			return LifecycleResult{}, err
		}
		if _, err := os.Stat(target); err == nil {
			if _, err := s.BackupFile(target, "import overwrite backup"); err != nil {
				return LifecycleResult{}, err
			}
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return LifecycleResult{}, err
		}
		reader, err := file.Open()
		if err != nil {
			return LifecycleResult{}, err
		}
		data, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			return LifecycleResult{}, err
		}
		if err := atomicWriteFile(target, data, 0o600); err != nil {
			return LifecycleResult{}, err
		}
		count++
	}
	output := fmt.Sprintf("IMPORT\n- Archive: %s\n- Files imported: %d\n- Existing files were backed up before overwrite.\n", req.From, count)
	return LifecycleResult{Output: output, Path: req.From, Files: count}, nil
}

func cleanZipPath(value string) (string, error) {
	value = filepath.ToSlash(strings.TrimSpace(value))
	if value == "" || strings.HasPrefix(value, "/") || strings.Contains(value, "\x00") {
		return "", fmt.Errorf("unsafe archive path: %q", value)
	}
	clean := filepath.ToSlash(filepath.Clean(value))
	if clean == "." || strings.HasPrefix(clean, "../") || clean == ".." {
		return "", fmt.Errorf("unsafe archive path: %q", value)
	}
	return clean, nil
}

func (s *Store) DeleteData(req DeleteDataRequest) (LifecycleResult, error) {
	root := filepath.Clean(s.Root)
	if strings.TrimSpace(root) == "" || root == "." || root == string(filepath.Separator) {
		return LifecycleResult{}, fmt.Errorf("refusing unsafe store root: %s", s.Root)
	}
	marker := filepath.Join(root, ".leftoff-store")
	if _, err := os.Stat(marker); err != nil {
		return LifecycleResult{}, fmt.Errorf("refusing to delete unmarked store; run init first or inspect manually")
	}
	files := 0
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() {
			files++
		}
		return nil
	})
	if req.DryRun {
		output := fmt.Sprintf("DELETE DATA\n- Dry run: %s would be removed.\n- Files: %d\n", root, files)
		return LifecycleResult{Output: output, Path: root, Files: files}, nil
	}
	if !req.Confirm {
		return LifecycleResult{}, fmt.Errorf("delete-data requires --confirm")
	}
	if err := os.RemoveAll(root); err != nil {
		return LifecycleResult{}, err
	}
	output := fmt.Sprintf("DELETE DATA\n- Removed store: %s\n- Files removed: %d\n", root, files)
	return LifecycleResult{Output: output, Path: root, Files: files}, nil
}
