module github.com/uchaloop/confx/integration/secret

go 1.27.0

require (
	github.com/uchaloop/confmaker/v2 v2.0.0
	github.com/uchaloop/confx v0.5.0
	github.com/uchaloop/secret/v2 v2.2.0
	go.uber.org/fx v1.24.0
)

// Test the adapter checkout. The core is supplied by the workspace or publication.
replace github.com/uchaloop/confx => ../..
