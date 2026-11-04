# confx

[![Go Reference](https://pkg.go.dev/badge/github.com/uchaloop/confx.svg)](https://pkg.go.dev/github.com/uchaloop/confx) [![CI](https://github.com/uchaloop/confx/actions/workflows/ci.yml/badge.svg)](https://github.com/uchaloop/confx/actions/workflows/ci.yml) [![Release](https://img.shields.io/github/v/tag/uchaloop/confx?label=release)](https://github.com/uchaloop/confx/tags) [![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**Explicit confmaker registration for Uber Fx.** confx provides typed configs to
Fx consumers. [confmaker](https://github.com/uchaloop/confmaker) owns defaults,
configuration engines, validation and diagnostics; the adapter owns wiring them into Fx.

[Install](#installation) · [Quick start](#quick-start) · [App options](#app-options-and-lifecycle) · [Modules](#modules-and-instances) · [Engines](#configuration-engines) · [Diagnostics](#errors-and-diagnostics) · [Manifest](#manifest-and-describe) · [Testing](#testing)

## Installation

Requires **Go 1.27.0 or later**. The command below installs the latest stable release.
This README describes the current branch; for a pinned release, use its tagged documentation.
The `App` API uses `confcli.Describe` from confmaker/v2 v2.1.0 or later.

```sh
go get github.com/uchaloop/confx@latest
```

```go
import "github.com/uchaloop/confx"
```

Import `github.com/uchaloop/confmaker/v2` when using core options such as `WithEnv`,
`WithPrefix` or `WithDiagnostics`.

## Quick start

Choose the entry point that matches your application:

| Need | Entry point |
| --- | --- |
| Ordinary Fx injection, no built-in help/describe | `fx.New(confx.Module(), server.Module, store.Module).Run()` |
| Built-in help and configuration description | `confx.New(modules...).Run()` |
| Custom CLI or manual Fx lifecycle | `App.Manifest` / `App.FX` |
| Caller-owned loader and registrations | `FromLoader` / `FromHandle` |

Both variants below use these shared configs and service modules. Create a
`main.go` with this block, then append **one** of the startup blocks:

```go
package main

import (
	"fmt"

	"github.com/uchaloop/confx"
	"go.uber.org/fx"
)

type ServerConfig struct {
	Port int `env:"PORT"`
}

func (c *ServerConfig) SetDefaults() { c.Port = 8080 }

type StoreConfig struct {
	Host string `env:"HOST"`
}

func (c *StoreConfig) SetDefaults() { c.Host = "localhost:5432" }

func showServer(cfg ServerConfig) { fmt.Println("server port:", cfg.Port) }
func showStore(cfg StoreConfig) { fmt.Println("store host:", cfg.Host) }

var ServerModule = fx.Module("server", fx.Invoke(showServer))
var StoreModule = fx.Module("store", fx.Invoke(showStore))
```

### Ordinary Fx without describe

Register configs at the root and install `confx.Module()` once. Service modules
use ordinary Fx options; neither `MakeModule` nor `Runtime` is required:

```go
func main() {
	fx.New(
		confx.Module(),
		confx.Provide[ServerConfig]("server"),
		confx.Provide[StoreConfig]("store"),
		ServerModule,
		StoreModule,
	).Run()
}
```

This is a supported API for applications that do not need offline description.
To configure loading, pass core options to `confx.Module(opts...)`, for example
`confx.Module(confmaker.WithDiagnosticHandler(handleReport))`.

### App with help and describe

Keep config declarations visible at the root and wrap service modules in
`Runtime`. The same service modules work with both entry points:

```go
func main() {
	confx.New(
		confx.Provide[ServerConfig]("server"),
		confx.Provide[StoreConfig]("store"),
		confx.Runtime(
			ServerModule,
			StoreModule,
		),
	).Run()
}
```

App installs the adapter itself; do not add `confx.Module()` or `FromLoader`.
Service constructors consume the same typed configs normally. Keep registration
outside `Runtime`, including inside its nested ordinary Fx modules.

Both examples read `SERVER_PORT` and `STORE_HOST`, with defaults `8080` and
`localhost:5432`. The invocations only demonstrate injection; they do not start
an HTTP server or database. Normal execution waits for Fx shutdown signals:

```sh
go build -o service .
SERVER_PORT=9090 STORE_HOST=db:5432 ./service
```

Only the App variant handles the following CLI commands.

`New` is lazy: it captures the module list without constructing Fx, reading ENV,
or evaluating defaults. `Run` reads the current process arguments and supports:

```sh
./service -help
./service -describe=env
./service -describe=markdown
./service -describe=json
```

Help and description return without constructing Fx. For configurable defaults,
custom flags and manual startup, see the following sections.

## App options and lifecycle

`WithOptions` appends loader options and returns a new composition:

```go
confx.New(
	confx.Provide[ServerConfig]("server"),
	confx.Provide[StoreConfig]("store"),
	confx.Runtime(ServerModule, StoreModule),
).WithOptions(
	confmaker.WithDiagnosticHandler(handleReport),
).Run()
```

`WithManifestOptions` replaces description options and also returns a copy.
App descriptions include defaults unless configured otherwise. To skip default
evaluation without changing runtime loading or disabling CLI:

```go
confx.New(
	confx.Provide[ServerConfig]("server"),
	confx.Provide[StoreConfig]("store"),
	confx.Runtime(ServerModule, StoreModule),
).WithManifestOptions().Run()
```

`App.Manifest(opts...)` applies saved options followed by the call's options.
See [Manifest and describe](#manifest-and-describe) for the inspection boundary.
The zero `App` is equivalent to `New()`.

Without a CLI action, `Run` delegates to `fx.New(app.FX()).Run()`. Fx owns signals,
timeouts, startup rollback and shutdown exit codes. Help and descriptions go to
stdout and return. CLI/description/output errors print to stderr and exit with
code 1; unknown flags and positional arguments are rejected. User panics are not
recovered. Concurrent Run calls are unsupported.

Process exits skip caller defers. For custom CLI flags use `App.Manifest` with
`confcli.MakeDescribeFlag`; for manual lifecycle control construct
`fx.New(app.FX(), ...)` and use Fx's `Start`/`Stop`. `confx.App` itself has no
`Err`, `Start`, `Stop` or `Wait` methods. See [Testing](#testing) for `fx.Populate`.

All registered configurations, including unused ones, load during `fx.New`,
before lifecycle `OnStart`. Ordinary constructors and invocations can also run
at construction time; `Runtime` does not defer them until Start. Loading errors
prevent startup. The adapter adds no cancellation or engine resource lifecycle
to Fx construction.

`Module` and `App.FX` create a fresh loader per Fx application.
App installs the adapter automatically; do not combine its `FX()` with
`Module` or `FromLoader`. Options can be reused; supplied engines and callbacks
remain shared. Treat loaded configuration values, including maps and slices, as
read-only. Caller-owned loaders follow the [external loader](#external-loader)
contract instead.

## Modules and instances

For advanced compositions that keep config registration inside domain modules,
nest `MakeModule` values to preserve Fx private-provider visibility and decorator
scopes. A plain `fx.Module` hides declarations from offline inspection; use
`MakeModule` on configuration-bearing branches and `Runtime` on service-only branches.

An instance name, an ENV prefix and an Fx tag serve different purposes:

| Concept | Source | Purpose |
|---|---|---|
| Instance name | First argument to Provide / ProvideNamed | Core registration identity and diagnostics |
| ENV prefix (built-in engine) | Derived from the name, or WithPrefix | Variable lookup |
| Fx name tag | ProvideNamed only | Select a named dependency |

Using `ServerConfig` from the quick start, replace its registration with:

```go
confx.Provide[ServerConfig]("server"),
confx.ProvideNamed[ServerConfig]("replica", confmaker.WithPrefix("READ_SERVER_")),
```

The first reads `SERVER_PORT`, the second `READ_SERVER_PORT`. Consume them with:

```go
fx.Invoke(func(params struct {
	fx.In

	Primary ServerConfig
	Replica ServerConfig `name:"replica"`
}) {
	fmt.Println(params.Primary.Port, params.Replica.Port)
})
```

The same parameter shape can be used by a constructor. Two untagged providers of the same type conflict in
Fx even if their instance names differ. `WithPrefix` never changes the Fx tag.

## Configuration engines

The default engine reads ENV with confmaker tags and rules. Configure it through
`Module(opts...)` or `App.WithOptions`.

A custom engine is selected once for the same composition:

```go
definition := confx.New(
	confx.Provide[ServerConfig]("server"),
	confx.Provide[StoreConfig]("store"),
	confx.Runtime(ServerModule, StoreModule),
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

Engines and callbacks must support concurrent calls if reused concurrently.
The application owns their resources. Backend errors may contain input values;
the diagnostic report contains no configuration values. Custom-engine reports
have config-level detail. Custom engines do not support manifest/describe; they
return `confmaker.ErrManifestUnsupported`. App help remains available.

## Errors and diagnostics

Inspect construction errors on the `fx.App` returned by `fx.New`.

With the App composition above (and imports for `log` and confmaker), core structured
errors remain reachable through Fx's wrappers:

```go
definition := confx.New(
	confx.Provide[ServerConfig]("server"),
	confx.Provide[StoreConfig]("store"),
	confx.Runtime(ServerModule, StoreModule),
)
fxApp := fx.New(definition.FX())
if err := fxApp.Err(); err != nil {
	for _, problem := range confmaker.ConfigErrors(err) {
		fmt.Println(problem.Kind, problem.InstanceName, problem.VariableName)
	}
	log.Fatal(err)
}
```

Always handle the original error: missing Module and Fx dependency conflicts are
not core `ConfigError` diagnostics. Nil adapter arguments can be recognized with
`errors.Is(err, confx.ErrLoaderIsNil)` or `errors.Is(err, confx.ErrHandleIsNil)`;
this also works through `fxApp.Err()`.

Use `confmaker.WithDiagnosticHandler(func(report confmaker.LoadReport) { ... })`
through `App.WithOptions` (or the standalone `Module`) for a value-free report
during app construction, including load errors.
With an external loader, `WithDiagnostics()` enables `loader.Report()` snapshots.
The loader owns the report; there is no separate receiver. Callbacks run after
publication and do not run for loading panics.

If Fx rejects the graph before loading, the diagnostic callback does not run.
An enabled external loader remains `not_started`. Inspect `fxApp.Err()` for graph
errors; reports describe configuration loading, not the entire Fx application.

## Manifest and describe

Use the App composition from the quick start as the declaration tree. Manifest
does not build Fx, load ENV, invoke services or call load diagnostics. It does
not validate the Fx graph or all load-only options.

`App.Manifest` includes defaults by default. Use `WithManifestOptions()` to
skip default evaluation. Defaults and renderers are application
code and may perform I/O, including reading ENV.

To avoid incomplete output, manifest rejects opaque options outside `Runtime`,
including ordinary `fx.Options`, `fx.Module` and `fx.Provide`, with
`confx.ErrOpaqueOption`. The error identifies the module path and option index.
The entire tree is checked before defaults run; errors return no partial result.
These ordinary Fx options remain usable on the runtime `FX()` path.

`Runtime` asserts that its contents have no configuration declarations. Direct
registrations, module specifications and standalone loader/handle options are
rejected inside it. Registrations hidden deeper inside an opaque Fx wrapper
cannot be detected. Go argument evaluation and package `init` run normally, even
in description mode. See [Configuration engines](#configuration-engines) for
the custom-engine limitation.

### Custom CLI and manual lifecycle

Use `App.Manifest` with your own flag parser, or pass the App to
`confcli.Describe`. The same composition supplies the Fx application:

```go
definition := confx.New(
	confx.Provide[ServerConfig]("server"),
	confx.Provide[StoreConfig]("store"),
	confx.Runtime(ServerModule, StoreModule),
)

isDescribed, err := confcli.Describe(definition, os.Args[1:], os.Stdout)
if err != nil {
	log.Fatal(err)
}

if isDescribed {
	return
}

fxApp := fx.New(definition.FX())
fxApp.Run()
```

For a cronjob, use Fx's `Start`/`Stop` and `fx.Populate` instead of `Run` to retain
control of the task and cleanup. `App.FX()` does not parse flags or exit.

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
External loaders also provide `LoadContext` and report snapshots for callers
that need explicit loading control. These adapters do not parse CLI flags or
terminate the process.

## Testing

Pass an isolated ENV map to Module. This setup uses `ServerConfig` above and
imports `testing`, confmaker, confx and Fx:

```go
func TestConfig(t *testing.T) {
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

These tests exercise Fx construction; `Start` is needed only for lifecycle hooks.

For the App variant, keep CLI parsing out of the test and use `FX()` with
`fx.Populate`. This fragment belongs inside a test using the App modules above:

```go
definition := confx.New(
	confx.Provide[ServerConfig]("server"),
	confx.Provide[StoreConfig]("store"),
	confx.Runtime(ServerModule, StoreModule),
).WithOptions(
	confmaker.WithEnv(map[string]string{"SERVER_PORT": "9090"}),
)
var got ServerConfig
fxApp := fx.New(fx.NopLogger, definition.FX(), fx.Populate(&got))
if err := fxApp.Err(); err != nil {
	t.Fatal(err)
}
if got.Port != 9090 {
	t.Fatalf("unexpected port: %d", got.Port)
}
```

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
