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

func (q *Query) FromFile(filePath string) (*Query, error) {
	rawQuery, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(filePath, ".sql") {
		return &Query{
			Raw: string(queryBy)
			Type: SQL
		}
	}

	if strings.HasSuffix((filePath, ".promql")) {
		return &Query{
			Raw: string(queryBy)
			Type: Prometheus
		}
	}


	return nil, fmt.Errorf("File: %s is not supported.", filePath)
}
