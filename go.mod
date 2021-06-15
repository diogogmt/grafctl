module github.com/diogogmt/grafctl

go 1.16

require (
	cloud.google.com/go/storage v1.12.0
	github.com/grafana-tools/sdk v0.0.0-20210107173053-d7dc7721321b
	github.com/olekukonko/tablewriter v0.0.4
	github.com/peterbourgon/ff/v2 v2.0.0
)

replace github.com/grafana-tools/sdk v0.0.0-20210107173053-d7dc7721321b => github.com/diogogmt/sdk v0.0.0-20210615013009-e94e88581b1b
