module github.com/diogogmt/grafctl

go 1.15

require (
	github.com/grafana-tools/sdk v0.0.0-20201123153837-5fb28a7aa2ef
	github.com/peterbourgon/ff/v2 v2.0.0
	github.com/pkg/errors v0.9.1 // indirect
)

// replace github.com/grafana-tools/sdk => /Users/diogo/proj/sandbox/grafana-sdk
replace github.com/grafana-tools/sdk v0.0.0-20201123153837-5fb28a7aa2ef => github.com/diogogmt/sdk v0.0.0-20201204212852-38b761b8ff17
