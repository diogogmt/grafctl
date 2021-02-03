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

func NewQueryFromFile(filePath string) (*Query, error) {
	rawQuery, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(filePath, ".sql") {
		return &Query{
			Raw:  string(rawQuery),
			Type: SQL,
		}, nil
	}

	if strings.HasSuffix(filePath, ".promql") {
		return &Query{
			Raw:  string(rawQuery),
			Type: Prometheus,
		}, nil
	}

	return nil, fmt.Errorf("file: %s is not supported", filePath)
}
