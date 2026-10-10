# Secret integration

Tests cover real secret values through Module/Provide and FromLoader/FromHandle,
including diagnostics and external-loader manifest redaction. The original
secret-dependent benchmark lives here too. The root module does not depend on secret.

From GolandProjects with both integration modules and all sibling libraries in go.work:

```sh
go test -race ./confx/... ./confx/integration/secret/...
go test ./confx/integration/secret/... -run '^$' -bench BenchmarkModuleStart -benchtime=1x
```

Standalone, after confmaker v2.0.0 and secret v2.2.0 are published, run here:

```sh
GOWORK=off go mod tidy
GOWORK=off go vet ./...
GOWORK=off go test -race ./...
```

The relative replacement tests this adapter checkout. The root test pattern skips
nested modules; CI runs this one explicitly. Generate published dependency checksums
after publication; local workspace files are not committed.
