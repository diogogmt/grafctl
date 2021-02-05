package command

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type QueryType int

const (
	SQL = iota
	Prometheus
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
			Type: Prometheus,
		}
	} else {
		return fmt.Errorf("query file: %s is not supported", file)
	}

	q.m[name] = &query
	return nil
}
