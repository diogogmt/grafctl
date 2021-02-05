package command

import (
	"fmt"
	"io/ioutil"
	"strings"
)

type QueryType int

const (
	SQL = iota
	Prometheus
)

type Query struct {
	Raw  string
	Type QueryType
}

type QueryManager map[string]*Query

func (qm QueryManager) SupportedQueryFile(file string) bool {
	if strings.HasSuffix(file, ".sql") {
		return true
	} else if strings.HasSuffix(file, ".promql") {
		return true
	}
	return false
}

func (qm QueryManager) Get(file string) *Query {
	if query, ok := qm[file]; ok {
		return query
	}

	if query, ok := qm[fmt.Sprintf("%s.sql", file)]; ok {
		return query
	}

	if query, ok := qm[fmt.Sprintf("%s.promql", file)]; ok {
		return query
	}
	return nil
}

func (qm QueryManager) Put(file string, dirAbsPath string) error {
	rawQuery, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	var query Query

	if strings.HasSuffix(file, ".sql") {
		query = Query{
			Raw:  string(rawQuery),
			Type: SQL,
		}
	} else if strings.HasSuffix(file, ".promql") {
		query = Query{
			Raw:  string(rawQuery),
			Type: Prometheus,
		}
	} else {
		return fmt.Errorf("query file: %s is not supported", file)
	}

	qm[strings.TrimLeft(strings.ReplaceAll(file, dirAbsPath, ""), "/")] = &query
	return nil
}
