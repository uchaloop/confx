# confx

[![Go Reference](https://pkg.go.dev/badge/github.com/uchaloop/confx.svg)](https://pkg.go.dev/github.com/uchaloop/confx) [![CI](https://github.com/uchaloop/confx/actions/workflows/ci.yml/badge.svg)](https://github.com/uchaloop/confx/actions/workflows/ci.yml) [![Release](https://img.shields.io/github/v/tag/uchaloop/confx?label=release)](https://github.com/uchaloop/confx/tags) [![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**Explicit confmaker registration for Uber Fx.** confx provides typed configs to
Fx consumers. [confmaker](https://github.com/uchaloop/confmaker) owns defaults,
ENV parsing, validation and diagnostics; the adapter owns wiring them into Fx.

[Install](#installation) · [Quick start](#quick-start) · [Instances](#multiple-instances) · [Lifecycle](#lifecycle-and-errors) · [Testing](#testing) · [Reference](#reference)

## Installation

Requires **Go 1.27 or later** and confmaker **v0.8.0**. This README describes
the **v0.2.0** API.

```sh
go get github.com/uchaloop/confx@v0.2.0
```

```go
import "github.com/uchaloop/confx"
```

Import `github.com/uchaloop/confmaker` when using core options such as `WithEnv`,
`WithPrefix` or `WithDump`.

## Quick start

```go
package main

import (
	"fmt"
	"log"

	"github.com/uchaloop/confx"
	"go.uber.org/fx"
)

type ServerConfig struct {
	Port int `env:"PORT"`
}

func (c *ServerConfig) SetDefaults() { c.Port = 8080 }

func main() {
	app := fx.New(
		confx.Module(),
		confx.Provide[ServerConfig]("server"),
		fx.Invoke(func(cfg ServerConfig) {
			fmt.Println(cfg.Port) // Pass the config to your server constructor.
		}),
	)
	if err := app.Err(); err != nil {
		log.Fatal(err)
	}
	app.Run()
}
```

Run with no ENV for the default `8080`, or set `SERVER_PORT` in your IDE or shell.
The config type needs no dependency on either confmaker or Fx.

```mermaid
flowchart LR
    A["Provide / ProvideNamed"] --> B["Module: one core loader"]
    B --> C["Register and load all configs during fx.New"]
    C -->|Success| D["Typed values for consumers"]
    C -->|Error| E["app.Err; startup prevented"]
```

## Multiple instances

An instance name, an ENV prefix and an Fx tag serve different purposes:

| Concept | Source | Purpose |
|---|---|---|
| Instance name | First argument to Provide / ProvideNamed | Core registration identity and diagnostics |
| ENV prefix | Derived from the name, or WithPrefix | Variable lookup |
| Fx name tag | ProvideNamed only | Select a named dependency |

Using `ServerConfig` from the quick start, replace its registration with:

```go
confx.Provide[ServerConfig]("server"),
confx.ProvideNamed[ServerConfig]("replica", confmaker.WithPrefix("READ_SERVER_")),
```

The first reads `SERVER_PORT`, the second `READ_SERVER_PORT`. Consume them with:

```go
type Params struct {
	fx.In
	Primary ServerConfig
	Replica ServerConfig `name:"replica"`
}
```

Use `fx.Invoke(func(p Params) { /* use p.Primary and p.Replica */ })` or take
`Params` in a constructor. Two untagged providers of the same type conflict in
Fx even if their instance names differ. `WithPrefix` never changes the Fx tag.

## Lifecycle and errors

Include `confx.Module()` exactly once when using Provide or ProvideNamed.
All provided configs load during `fx.New`, including those with no consumers.
A core loading error is available through `app.Err()` and prevents startup.

Core structured errors remain reachable through Fx's wrappers:

```go
if err := app.Err(); err != nil {
	for _, problem := range confmaker.ConfigErrors(err) {
		fmt.Println(problem.Kind, problem.InstanceName, problem.VariableName)
	}
	log.Fatal(err)
}
```

Always handle the original error: missing Module and Fx dependency conflicts are
not core `ConfigError` diagnostics. Config values should be treated as read-only.
Provider options may be reused across applications; each app receives its own
loader and values.

## Testing

Pass an isolated ENV map to Module. This setup uses `ServerConfig` above and
imports `testing`, confmaker, confx and Fx:

```go
func TestServerConfig(t *testing.T) {
	t.Parallel()

	var got ServerConfig
	app := fx.New(
		fx.NopLogger,
		confx.Module(confmaker.WithEnv(map[string]string{"SERVER_PORT": "9090"})),
		confx.Provide[ServerConfig]("server"),
		fx.Populate(&got),
	)
	if err := app.Err(); err != nil {
		t.Fatal(err)
	}
	if got.Port != 9090 {
		t.Fatalf("unexpected port: %d", got.Port)
	}
}
```

No `app.Start` is needed to inspect configuration values: loading already
happened during construction. In a real app, unrelated constructors can run
during `fx.New`; this is not a general-purpose dry-run mode.

## Manifest and documentation

Manifest and export APIs belong to confmaker. See its
[manifest guide](https://github.com/uchaloop/confmaker#manifest-and-configuration-documentation)
for `.env.example`, JSON and Markdown generation.

confx does not currently expose a manifest-only mode for a list of Provide
options. Do not construct an entire Fx application just to extract metadata:
use the core APIs for standalone descriptions. The simple Provide API remains
the application registration entry point.

## Reference

- [confx GoDoc](https://pkg.go.dev/github.com/uchaloop/confx): adapter API.
- [confmaker README](https://github.com/uchaloop/confmaker#readme): tags, secrets, errors and loading rules.
- [Changelog](CHANGELOG.md): release and migration notes.

## Acknowledgements

Thanks to the authors and maintainers of [Uber Fx](https://github.com/uber-go/fx)
for its dependency injection and lifecycle primitives.

## License

[MIT](LICENSE)
