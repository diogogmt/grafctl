package command

import "fmt"

type QueryManager map[string]*Query

func (q QueryManager) Get(file string) *Query {
	if query, ok := q[file]; ok {
		return query
	}

	if query, ok := q[fmt.Sprintf("%s.sql", file)]; ok {
		return query
	}

	if query, ok := q[fmt.Sprintf("%s.promql", file)]; ok {
		return query
	}
	return nil
}
