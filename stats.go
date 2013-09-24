package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type ProjectStats struct {
	ImportStatsByPath map[string]*ImportStats
}

type ImportStats struct {
	Path               string
	Remote             bool
	ReferencePositions []token.Position
}

func NewProjectStats() *ProjectStats {
	return &ProjectStats{
		make(map[string]*ImportStats),
	}
}

func AnalyzeSourceTree(dir string) (*ProjectStats, error) {
	ps := NewProjectStats()
	err := filepath.Walk(
		dir,
		func(path string, info os.FileInfo, err error) error {
			fileDir := filepath.Dir(path)
			baseName := filepath.Base(path)
			if strings.HasSuffix(baseName, ".go") {
				// Bail if not analyzing the gopack dir specifically
				// and we hit that directory as part of this analysis.
				// (should only ever be an issue with running gopack on itself and running tests)
				// (use Contains rather than HasPrefix to handle absolute and relative paths)
				if !strings.Contains(dir, GopackDir) &&
					strings.Contains(fileDir, GopackDir) {
					return nil
				}
				e := ps.analyzeSourceFile(path)
				if e != nil {
					return e
				}
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	return ps, nil
}

func (ps *ProjectStats) analyzeSourceFile(path string) error {
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, path, nil, parser.ParseComments)
	if err != nil {
		return err
	}
	for _, i := range f.Imports {
		err = ps.foundImport(fs, i, path)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ps *ProjectStats) foundImport(fs *token.FileSet, i *ast.ImportSpec, path string) error {
	importPath, err := strconv.Unquote(i.Path.Value)
	if err != nil {
		return err
	}
	ref := fs.Position(i.Pos())
	_, found := ps.ImportStatsByPath[importPath]
	if found {
		ps.ImportStatsByPath[importPath].ReferencePositions = append(ps.ImportStatsByPath[importPath].ReferencePositions, ref)
	} else {
		ps.ImportStatsByPath[importPath] = NewImportStats(importPath, ref)
	}
	return nil
}

func (ps *ProjectStats) IsImportUsed(importPath string) bool {
	_, used := ps.ImportStatsByPath[importPath]
	return used
}

func (ps *ProjectStats) Summary() string {
	lines := []string{}
	for k, v := range ps.ImportStatsByPath {
		if v.Remote {
			lines = append(lines, fmt.Sprintf("[R] %s:%d", k, len(v.ReferencePositions)))
		} else {
			lines = append(lines, fmt.Sprintf("[S] %s:%d", k, len(v.ReferencePositions)))
		}
	}
	sort.Strings(lines)

	return fmt.Sprintf("Import stats summary:\n\n* %s\n\n%s", strings.Join(lines, "\n* "), "[R] Remotes, [S] Stdlib")
}

func NewImportStats(importPath string, pos token.Position) *ImportStats {
	parts := strings.Split(importPath, "/")
	remote := false
	if len(parts) > 0 && strings.Contains(parts[0], ".") && strings.Index(parts[0], ".") > 0 {
		remote = true
	}
	return &ImportStats{
		importPath, remote, []token.Position{pos},
	}
}

func (i *ImportStats) ReferenceList() string {
	lines := []string{}
	for _, ref := range i.ReferencePositions {
		lines = append(lines, fmt.Sprintf("%s:%d", ref.Filename, ref.Line))
	}
	return fmt.Sprintf("* %s", strings.Join(lines, "\n* "))
}
