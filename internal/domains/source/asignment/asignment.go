package asignment

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Galdoba/ffquery/internal/domains/content"
)

type SourceFileInclusion struct {
	ID                      string //sfi-260703141602
	InitialNames            []string
	OtherAssignedNames      []string
	OtherUnassignedNames    []string
	CurrentlyOpenedProjects []string
	Content                 content.Type
	candidates              map[string][]string
	DecidedOutcome          string
	DecidionReason          string
	ConclusionTime          time.Time
}

// GetFileNames reads a directory and returns the names of all entries that are
// suitable for both reading and writing (regular files with read/write permissions).
// If any entry is unsuitable – e.g., a directory, a symlink to a non‑regular file,
// or a file without read/write access – the function returns an error immediately.
func GetFileNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	var names []string
	for _, entry := range entries {
		name := entry.Name()
		fullPath := filepath.Join(dir, name)

		// Stat follows symlinks to examine the underlying file.
		info, err := os.Stat(fullPath)
		if err != nil {
			return nil, fmt.Errorf("cannot stat %s: %w", name, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("unsuitable file %s: not a regular file", name)
		}

		// Attempt to open with O_RDWR to confirm read and write access.
		f, err := os.OpenFile(fullPath, os.O_RDWR, 0)
		if err != nil {
			return nil, fmt.Errorf("unsuitable file %s: %w", name, err)
		}
		f.Close()

		names = append(names, name)
	}
	return names, nil
}

// func prefixCandidates(projectName string) ([]string, error) {
// 	//получаем имя, транслитерируем, отсекаем по словам, конвертируем сезон/эпизод в метку.
// 	return []string{}, nil
// }

// func (sfi *SourceFileInclusion) fillCandidates() error {
// 	if sfi.candidates != nil {
// 		return fmt.Errorf("candidates already filled")
// 	}
// 	sfi.candidates = make(map[string][]string, len(sfi.CurrentlyOpenedProjects))
// 	for _, projectName := range sfi.CurrentlyOpenedProjects {
// 		vals, err := prefixCandidates(projectName)
// 		if err != nil {
// 			sfi.candidates = nil
// 			return fmt.Errorf("failed to fill candidates for %q: %w", projectName, err)
// 		}
// 		sfi.candidates[projectName] = vals
// 	}
// 	return nil
// }

// func (sfi *SourceFileInclusion) ValidatePrefix(prefix source.Prefix) error {
// 	for _, list := range sfi.candidates {
// 		if slices.Contains(list, prefix.Key) {
// 			return nil
// 		}
// 	}
// 	return fmt.Errorf("prefix %q is not suitable", prefix)
// }

// func (sfi *SourceFileInclusion) SetPrefix(prefix string) error {
// 	p := source.Prefix{Key: prefix, Type: sfi.Content}
// 	if err := sfi.ValidatePrefix(p); err != nil {
// 		return err
// 	}
// 	sfi.DecidedOutcome = p.Key
// 	sfi.ConclusionTime = time.Now()
// 	return nil
// }
