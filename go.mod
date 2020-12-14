module github.com/diogogmt/grafctl

go 1.15

require (
	cloud.google.com/go/storage v1.12.0
	github.com/grafana-tools/sdk v0.0.0-20201123153837-5fb28a7aa2ef
	github.com/olekukonko/tablewriter v0.0.4
	github.com/peterbourgon/ff/v2 v2.0.0
)

// replace github.com/grafana-tools/sdk => /Users/diogo/proj/sandbox/grafana-sdk
replace github.com/grafana-tools/sdk v0.0.0-20201123153837-5fb28a7aa2ef => github.com/diogogmt/sdk v0.0.0-20201214224245-9753865751ec
