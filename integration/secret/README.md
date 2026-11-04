# Secret integration

Tests cover real secret values through Module/Provide and FromLoader/FromHandle,
including diagnostics, external-loader manifest redaction and App.Run description
redaction in env, Markdown and JSON formats. The original
secret-dependent benchmark lives here too. The root module does not depend on secret.

Optionally, from GolandProjects with both modules and sibling libraries in go.work:

```sh
go test -race ./confx/... ./confx/integration/secret/...
go test ./confx/integration/secret/... -run '^$' -bench BenchmarkModuleStart -benchtime=1x
```

For standalone checks against published confmaker/v2 v2.1.0 and secret/v2 v2.2.0,
run here without a workspace:

```sh
GOWORK=off go mod tidy -diff
GOWORK=off go build ./...
GOWORK=off go vet ./...
GOWORK=off go test -race ./...
```

The relative replacement tests this adapter checkout. The root test pattern skips
nested modules; CI runs this one explicitly. Published dependency checksums are
committed; local workspace files are not required and must not be committed.
