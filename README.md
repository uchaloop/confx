# confx

[![Go Reference](https://pkg.go.dev/badge/github.com/uchaloop/confx.svg)](https://pkg.go.dev/github.com/uchaloop/confx) [![CI](https://github.com/uchaloop/confx/actions/workflows/ci.yml/badge.svg)](https://github.com/uchaloop/confx/actions/workflows/ci.yml) [![Release](https://img.shields.io/github/v/tag/uchaloop/confx?label=release)](https://github.com/uchaloop/confx/tags) [![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**Explicit confmaker registration for Uber Fx.** confx provides typed configs to
Fx consumers. [confmaker](https://github.com/uchaloop/confmaker) owns defaults,
ENV parsing, validation and diagnostics; the adapter owns wiring them into Fx.

[Install](#installation) · [Quick start](#quick-start) · [Instances](#multiple-instances) · [Lifecycle](#lifecycle-and-errors) · [Diagnostics](#load-diagnostics) · [Testing](#testing) · [Manifest](#external-loader-and-manifest) · [Reference](#reference)

## Installation

Requires **Go 1.27.0 or later**. The command below installs the latest stable release.
This README describes the current branch; for a pinned release, use its tagged documentation.

```sh
go get github.com/uchaloop/confx@latest
```

```go
import "github.com/uchaloop/confx"
```

Import `github.com/uchaloop/confmaker/v2` when using core options such as `WithEnv`,
`WithPrefix` or `WithDiagnostics`.

## Core version and compatibility

This version uses **confmaker v2 API**. confx has its own versioning and remains
on **v0.x**; a stable core does not imply a stable adapter API.

When migrating, update all core imports and options/handles to v2. Internal
Module loaders expose diagnostics through WithDiagnosticHandler; external loaders
support WithDiagnostics() and Report(). Manifest defaults require IncludeDefaults,
and exporters now consume a prepared result. See the matching core version's
README for the configuration contract.

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

Include exactly one `confx.Module()` or `confx.FromLoader(loader)`.
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
With `Module`, `Provide` and `ProvideNamed` options may be reused across
applications; each app receives its own loader and values. External loaders
retain shared values and load state, as described below.

## Load diagnostics

Use `confmaker.WithDiagnosticHandler(func(report confmaker.LoadReport) { ... })`
with Module for a value-free report during app construction, including load errors.
With an external loader, `WithDiagnostics()` enables `loader.Report()` snapshots.
The loader owns the report; there is no separate receiver. Callbacks run after
publication and do not run for loading panics.

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

## External loader and manifest

The simple Module + Provide API remains the default. For documentation generation
without reading ENV or constructing services, register configurations in the core
before creating Fx, then bridge their handles into the application:

```go
package main

import (
    "flag"
    "log"
    "os"

    "github.com/uchaloop/confmaker/v2"
    "github.com/uchaloop/confmaker/v2/confcli"
    "github.com/uchaloop/confx"
    "go.uber.org/fx"
)

type ServerConfig struct {
    Port int `env:"PORT,required"`
}

func main() {
    describe := confcli.MakeDescribeFlag(flag.CommandLine)
    flag.Parse()

    loader := confmaker.MakeLoader()
    server := loader.Register[ServerConfig]("server")

    if describe.Requested() {
        manifest, err := loader.Manifest(confmaker.IncludeDefaults())
        if err != nil { log.Fatal(err) }
        if err := describe.Write(os.Stdout, manifest); err != nil {
            log.Fatal(err)
        }
        return
    }

    fx.New(
        confx.FromLoader(loader),
        confx.FromHandle(server),
        fx.Invoke(func(cfg ServerConfig) { log.Printf("server port: %d", cfg.Port) }),
        // Add runtime modules here.
    ).Run()
}
```

`go run . -describe=markdown` completes before `fx.New`; required ENV is not
needed. Use `-describe=env` or `-describe=json` for the other export formats.
Normal startup loads during `fx.New`. Either Run or explicit Start/Stop is valid:
separating description from application construction is what matters.

| Adapter | Purpose |
|---|---|
| `FromLoader(loader)` | Use an existing loader instead of Module |
| `FromHandle(handle)` | Provide an existing registration as an untagged value |
| `FromHandleNamed(handle)` | Provide it with its registration name as the Fx tag |

All registrations load, even when no handle has a consumer. Handles must belong
to the supplied loader; nil, zero and foreign handles fail construction.
An already-loaded loader reuses its result. Reusing an external loader across
apps also shares its values and load state; create separate loaders for isolation.

The two modes cannot be mixed: Module accepts only Provide/ProvideNamed;
FromLoader accepts only FromHandle/FromHandleNamed. Mode and handle ownership
checks finish before any registration or loading. Invalid connections leave the
external loader unchanged, so a corrected connection can reuse it. Once Load
has run, its result remains final as usual.

Register every config outside Fx in the extended mode. The manifest generated
before fx.New then describes the complete registration set.

For export formats and guarantees, see the core
[manifest guide](https://github.com/uchaloop/confmaker#manifest-and-exports).
The adapters do not parse flags or terminate the process.

## Reference

- [confx GoDoc](https://pkg.go.dev/github.com/uchaloop/confx): adapter API.
- [confmaker README](https://github.com/uchaloop/confmaker#readme): tags, secrets, errors and loading rules.
- [Changelog](CHANGELOG.md): release and migration notes.

## Acknowledgements

Thanks to the authors and maintainers of [Uber Fx](https://github.com/uber-go/fx)
for its dependency injection and lifecycle primitives.

## License

[MIT](LICENSE)

## Custom configuration engines

Pass `confmaker.WithEngine(engine)` to `confx.Module`, or configure an external
Loader before `FromLoader`. Module accepts `confmaker.LoaderOption`. Parsing and
tags belong to the backend; Fx wiring is unchanged:

```go
confx.Module(confmaker.WithEngine(engine), confmaker.WithDiagnostics())
```

Each app gets its own Loader when Module options are reused, but the Engine object
is shared: it must support concurrent calls across loaders. The caller owns its
resources. Loading still occurs during fx.New with the ordinary Load lifecycle;
this adapter adds no cancellation or engine resource lifecycle. Arbitrary core
registration names do not override Fx restrictions on named tags.

Custom engines provide config-level reports and do not support Loader.Manifest.
See the [core engine example](https://github.com/uchaloop/confmaker#custom-engines)
for a dependency-free extension point and an application-level env/v11 adapter.
