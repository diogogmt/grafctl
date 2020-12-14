module github.com/diogogmt/grafctl

go 1.15

require (
	cloud.google.com/go/storage v1.12.0
	github.com/TylerBrock/colorjson v0.0.0-20200706003622-8a50f05110d2
	github.com/fatih/color v1.10.0 // indirect
	github.com/grafana-tools/sdk v0.0.0-20201123153837-5fb28a7aa2ef
	github.com/hokaccha/go-prettyjson v0.0.0-20190818114111-108c894c2c0e // indirect
	github.com/olekukonko/tablewriter v0.0.4
	github.com/peterbourgon/ff/v2 v2.0.0
	github.com/pkg/errors v0.9.1 // indirect
)

// replace github.com/grafana-tools/sdk => /Users/diogo/proj/sandbox/grafana-sdk
replace github.com/grafana-tools/sdk v0.0.0-20201123153837-5fb28a7aa2ef => github.com/diogogmt/sdk v0.0.0-20201204212852-38b761b8ff17
