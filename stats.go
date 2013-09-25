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
	"text/tabwriter"
)

type ProjectStats struct {
	ImportStatsByPath map[string]*ImportStats
}

type ImportStats struct {
	Path               string
	Remote             bool
	ReferencePositions []token.Position
}

type SummaryItem struct {
	Origin int
	Sum    int
	Path   string
}

func (i SummaryItem) Legend() string {
	var origin string

	switch i.Origin {
	case 1:
		origin = "R"
	case 0:
		origin = "L"
	case -1:
		origin = "S"
	}

	return fmt.Sprintf("%s\t%s\t%d", origin, i.Path, i.Sum)
}

type Summary struct {
	Items []SummaryItem
}

func (s *Summary) Append(i SummaryItem)  { s.Items = append(s.Items, i) }
func (s *Summary) Get(i int) SummaryItem { return s.Items[i] }

func (s *Summary) Len() int      { return len(s.Items) }
func (s *Summary) Swap(i, j int) { s.Items[i], s.Items[j] = s.Items[j], s.Items[i] }
func (s *Summary) Less(i, j int) bool {
	i1 := s.Items[i]
	i2 := s.Items[j]

	return i1.Origin > i2.Origin || (i1.Origin == i2.Origin && i1.Sum > i2.Sum)
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

func (ps *ProjectStats) PrintSummary() {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 0, '\t', 0)
	summary := ps.GetSummary()

	fmt.Fprintln(writer, "Import stats summary:\n")
	for _, item := range summary.Items {
		fmt.Fprintln(writer, item.Legend())
	}
	fmt.Fprintln(writer, "\nR Remote, L Local, S Stdlib")
	writer.Flush()
}

func (ps *ProjectStats) GetSummary() *Summary {
	summary := &Summary{Items: []SummaryItem{}}

	for k, v := range ps.ImportStatsByPath {
		item := SummaryItem{Path: k, Sum: len(v.ReferencePositions)}
		if v.Remote {
			item.Origin = 1
		} else if strings.HasPrefix(k, ".") {
			item.Origin = 0
		} else {
			item.Origin = -1
		}
		summary.Append(item)
	}
	sort.Sort(summary)

	return summary
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
