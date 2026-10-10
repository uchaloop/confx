# confx

[![Go Reference](https://pkg.go.dev/badge/github.com/uchaloop/confx.svg)](https://pkg.go.dev/github.com/uchaloop/confx) [![CI](https://github.com/uchaloop/confx/actions/workflows/ci.yml/badge.svg)](https://github.com/uchaloop/confx/actions/workflows/ci.yml) [![Release](https://img.shields.io/github/v/tag/uchaloop/confx?label=release)](https://github.com/uchaloop/confx/tags) [![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**Explicit confmaker registration for Uber Fx.** confx provides typed configs to
Fx consumers. [confmaker](https://github.com/uchaloop/confmaker) owns defaults,
configuration engines, validation and diagnostics; the adapter owns wiring them into Fx.

[Install](#installation) · [Quick start](#quick-start) · [Modules](#modules-and-instances) · [Engines](#configuration-engines) · [Diagnostics](#errors-and-diagnostics) · [Manifest](#manifest-and-describe) · [Testing](#testing)

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

## Quick start

A domain declares its config and exposes one module. The config itself needs no
Fx-specific methods:

```go
package main

import (
	"fmt"

	"github.com/uchaloop/confx"
	"go.uber.org/fx"
)

type Config struct {
	Port int `env:"PORT"`
}

func (c *Config) SetDefaults() {
	c.Port = 8080
}

type service struct {
	port int
}

func makeService(cfg Config) *service {
	return &service{port: cfg.Port}
}

func main() {
	server := confx.MakeModule("server",
		confx.Provide[Config]("server"),
		confx.Runtime(
			fx.Provide(makeService),
			fx.Invoke(func(s *service) {
				fmt.Println("server port:", s.port)
			}),
		),
	)

	modules := confx.MakeModules(server)
	fx.New(modules.FX()).Run()
}
```

The built-in engine uses `8080` by default or reads `SERVER_PORT`. In a real
application, each domain exports its own `MakeModule` result; main lists those
modules once. `FX()` installs the adapter, so no additional `confx.Module()` is
needed. Service constructors consume config values normally.

## How it works

`Provide` and `ProvideNamed` preserve config declarations beside their Fx
providers. `MakeModule` preserves nesting and scopes; `MakeModules` assembles the
application. During `fx.New`, confmaker registers and loads all configurations,
including unused ones, before any config value is provided to a consumer.

Defaults, decoding, validation and reports belong to confmaker. confx owns Fx
registration and typed injection. Loading happens before lifecycle `OnStart`;
this adapter does not start engine resources or add cancellation to Fx construction.

Use `Runtime` for ordinary providers, invocations and decorators. It marks the
parts of the module tree that offline description must skip. It does not delay
`fx.Invoke` until `Start`: normal Fx construction semantics still apply.

## Modules and instances

Nest `MakeModule` values to preserve Fx private-provider visibility and decorator
scopes. A plain `fx.Module` hides declarations from offline inspection; use
`MakeModule` on configuration-bearing branches and `Runtime` on service-only branches.


An instance name, an ENV prefix and an Fx tag serve different purposes:

| Concept | Source | Purpose |
|---|---|---|
| Instance name | First argument to Provide / ProvideNamed | Core registration identity and diagnostics |
| ENV prefix (built-in engine) | Derived from the name, or WithPrefix | Variable lookup |
| Fx name tag | ProvideNamed only | Select a named dependency |

Using `Config` from the quick start, replace its registration with:

```go
confx.Provide[Config]("server"),
confx.ProvideNamed[Config]("replica", confmaker.WithPrefix("READ_SERVER_")),
```

The first reads `SERVER_PORT`, the second `READ_SERVER_PORT`. Consume them with:

```go
fx.Invoke(func(params struct {
	fx.In

	Primary Config
	Replica Config `name:"replica"`
}) {
	fmt.Println(params.Primary.Port, params.Replica.Port)
})
```

The same parameter shape can be used by a constructor. Two untagged providers of the same type conflict in
Fx even if their instance names differ. `WithPrefix` never changes the Fx tag.

## Configuration engines

The default engine reads ENV with confmaker tags and rules. Configure all domains
through the composition, for example `modules.WithOptions(confmaker.WithEnv(env))`.
`WithOptions` returns a copy and appends options; it does not mutate the original.

A custom engine is selected once for the same composition:

```go
modules := confx.MakeModules(
	orders.Module,
	billing.Module,
).WithOptions(
	confmaker.WithEngine(engine),
	confmaker.WithDiagnosticHandler(handleReport),
)
```

The custom engine owns its sources, tags and decoding rules. confmaker still
applies root defaults before decoding and validates afterward. Built-in options
such as `WithEnv` and `WithPrefix` cannot be combined with a custom engine.

| Runnable example | What it demonstrates |
| --- | --- |
| [env/v11](examples/engines/env11) | Native ENV tags and `envDefault`, isolated input maps, two named configs |
| [koanf + TOML](examples/engines/koanf) | Embedded TOML files, overlay settings, native `koanf` tags, two named configs |

Both examples build a domain module and inject configurations into a service.
Their dependencies live in a [separate Go module](examples/engines/README.md),
not in the adapter's main module. The package GoDoc also includes a runnable
standard-library JSON engine example. For Viper and direct YAML/TOML decoders,
see [confmaker's engine examples](https://github.com/uchaloop/confmaker/tree/main/examples/engines).

Each Fx application receives its own Loader, but reused Engine objects and callback
closures are shared. Engines must support concurrent calls if reused concurrently.
The application owns their resources. Backend errors may contain input values;
the diagnostic report contains no configuration values. Custom-engine reports
have config-level detail and their Manifest returns `ErrManifestUnsupported`.

## Errors and diagnostics

Use `modules.FX()`, or exactly one legacy `confx.Module()` / `confx.FromLoader(loader)`.
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


Use `confmaker.WithDiagnosticHandler(func(report confmaker.LoadReport) { ... })`
through `modules.WithOptions` (or the standalone `Module`) for a value-free report
during app construction, including load errors.
With an external loader, `WithDiagnostics()` enables `loader.Report()` snapshots.
The loader owns the report; there is no separate receiver. Callbacks run after
publication and do not run for loading panics.

If Fx rejects the graph before loading, the diagnostic callback does not run.
An enabled external loader remains `not_started`. Inspect `app.Err()` for graph
errors; reports describe configuration loading, not the entire Fx application.

## Manifest and describe

Declare configuration once alongside the domain services:

```go
var Module = confx.MakeModule("orders",
	confx.Provide[Config]("orders"),
	confx.Runtime(
		fx.Provide(makeService),
		fx.Invoke(registerHandlers),
	),
)
```

Compose the modules once in your application. `describe` below is a parsed
`confcli.MakeDescribeFlag`; flag parsing stays with the application.

```go
modules := confx.MakeModules(
	orders.Module,
	billing.Module,
).WithOptions(
	confmaker.WithDiagnosticHandler(handleReport),
)

if describe.Requested() {
	manifest, err := modules.Manifest(confmaker.IncludeDefaults())
	if err != nil {
		log.Fatal(err)
	}

	if err := describe.Write(os.Stdout, manifest); err != nil {
		log.Fatal(err)
	}

	return
}

fx.New(modules.FX()).Run()
```

`FX()` installs the core adapter itself: do not add `confx.Module()` or
`FromLoader()` alongside it. Each application receives a fresh loader using the
same saved options. `WithOptions` returns a new composition; supplied engines and
callback closures are shared. Nested `MakeModule` preserves Fx private visibility
and decorator scopes.

Manifest does not construct an Fx application, load ENV, invoke services or call
load diagnostics. `IncludeDefaults` explicitly executes configuration defaults
and renderers. It does not validate the service graph or all load-only options.
Custom engines still return `confmaker.ErrManifestUnsupported`.

To avoid silently incomplete output, Manifest rejects every opaque option outside
`Runtime`, including ordinary `fx.Options`, `fx.Module` and `fx.Provide`. The error
wraps `confx.ErrOpaqueOption` and identifies the module path and option index.
The entire tree is checked before defaults run; errors return no partial manifest.
Such ordinary Fx options remain usable on the `FX()` startup path.

`Runtime` asserts that its options contain no configuration declarations. Direct
confx registrations, module specifications and legacy loader/handle options are
rejected inside it. A registration hidden deeper inside an opaque Fx wrapper
cannot be detected: do not put configuration-bearing wrappers inside `Runtime`.
Argument evaluation, such as `fx.Supply(makeService())`, and Go package `init`
execute normally and are not suppressed by description mode.

The original `Module`/`Provide` and external-loader APIs remain available when
application-level composition or offline description is unnecessary.

## External loader

Use an external Loader when your application already owns configuration
registration, loading or inspection. `FromLoader` and the handle adapters bridge
those registrations into Fx without registering them again.

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

## Testing

Pass an isolated ENV map to Module. This setup uses `Config` above and
imports `testing`, confmaker, confx and Fx:

```go
func TestConfig(t *testing.T) {
	t.Parallel()

	var got Config
	app := fx.New(
		fx.NopLogger,
		confx.Module(confmaker.WithEnv(map[string]string{"SERVER_PORT": "9090"})),
		confx.Provide[Config]("server"),
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

For repository checks, run each module explicitly with `GOWORK=off`: the root,
`integration/secret` and `examples/engines`. In each directory run:

```sh
GOWORK=off go mod tidy -diff
GOWORK=off go build ./...
GOWORK=off go vet ./...
GOWORK=off go test -race ./...
```

Real secret compatibility lives in [integration/secret](integration/secret/README.md).
The root module does not depend on secret or the example engines. See
[CONTRIBUTING.md](CONTRIBUTING.md) for code style and contribution requirements.

## Reference

- [confx GoDoc](https://pkg.go.dev/github.com/uchaloop/confx): adapter API.
- [confmaker README](https://github.com/uchaloop/confmaker#readme): tags, secrets, errors and loading rules.
- [Changelog](CHANGELOG.md): release and migration notes.

## Acknowledgements

Thanks to the authors and maintainers of [Uber Fx](https://github.com/uber-go/fx)
for its dependency injection and lifecycle primitives.

## License

[MIT](LICENSE)
