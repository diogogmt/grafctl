package command

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type QueryType int

const (
	SQL = iota
	PromQL
)

type Query struct {
	Name string
	Raw  string
	Type QueryType
}

type QueryManager struct {
	m   map[string]*Query
	dir string
}

var (
	beforeQueryRegex = regexp.MustCompile(`.*/queries/(.+)`)
)

func NewQueryManager(queryDir string) (*QueryManager, error) {
	dirAbs := queryDir
	if !filepath.IsAbs(queryDir) {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		dirAbs = filepath.Join(wd, queryDir)
	}

	return &QueryManager{
		dir: dirAbs,
		m:   make(map[string]*Query),
	}, nil
}

func (q QueryManager) SupportedQueryFile(file string) bool {
	if strings.HasSuffix(file, ".sql") {
		return true
	} else if strings.HasSuffix(file, ".promql") {
		return true
	}
	return false
}

func (q QueryManager) Get(file string) *Query {
	if query, ok := q.m[file]; ok {
		return query
	}

	if query, ok := q.m[fmt.Sprintf("%s.sql", file)]; ok {
		return query
	}

	if query, ok := q.m[fmt.Sprintf("%s.promql", file)]; ok {
		return query
	}
	return nil
}

// GetByBaseAndRefId gets a query by base name and refId
// For example, GetByBaseAndRefId("queries/panel1", "F") will look for "queries/panel1_f.sql" or "queries/panel1_f.promql"
// If refId is empty or there's only one target, it will also try the base name without suffix
func (q QueryManager) GetByBaseAndRefId(baseName string, refId string) *Query {
	// First try the base name without refId (for single target cases)
	if query := q.Get(baseName); query != nil {
		return query
	}

	if refId == "" {
		return nil
	}

	// Try with lowercase refId (as used in export)
	lowerRefId := strings.ToLower(refId)

	// Try SQL first
	if query, ok := q.m[fmt.Sprintf("%s_%s.sql", baseName, lowerRefId)]; ok {
		return query
	}

	// Try PromQL
	if query, ok := q.m[fmt.Sprintf("%s_%s.promql", baseName, lowerRefId)]; ok {
		return query
	}

	return nil
}

func (q QueryManager) Put(file string) error {
	rawQuery, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	var query Query
	name := strings.TrimLeft(strings.ReplaceAll(file, q.dir, ""), "/")
	if strings.HasSuffix(file, ".sql") {
		query = Query{
			Name: name,
			Raw:  string(rawQuery),
			Type: SQL,
		}
	} else if strings.HasSuffix(file, ".promql") {
		query = Query{
			Name: name,
			Raw:  string(rawQuery),
			Type: PromQL,
		}
	} else {
		return fmt.Errorf("query file: %s is not supported", file)
	}

	// Trim everything before /queries/
	match := beforeQueryRegex.FindStringSubmatch(name)
	if len(match) > 1 {
		name = match[1]
	}

	q.m[name] = &query
	return nil
}
