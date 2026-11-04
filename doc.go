/*
Package confx adapts github.com/uchaloop/confmaker/v2 to Uber Fx. The application
explicitly registers its package configs; confmaker owns configuration engines, defaults,
validation and diagnostics, while confx provides the resulting typed values.
The adapter supports both the built-in ENV engine and application-supplied engines.

Import github.com/uchaloop/confx for registration and
github.com/uchaloop/confmaker/v2 when using core options such as WithEnv or
WithPrefix.

# Quick start

For ordinary Fx injection without built-in help/describe, use Module once and
regular Fx modules. Neither MakeModule nor Runtime is required:

	var serverModule = fx.Module("server",
		confx.Provide[ServerConfig]("server"),
		fx.Provide(makeServer),
	)
	var storeModule = fx.Module("store",
		confx.Provide[StoreConfig]("store"),
		fx.Provide(makeStore),
	)
	fx.New(confx.Module(), serverModule, storeModule).Run()

For built-in help/describe, declare the same domains using MakeModule and put
ordinary Fx options in Runtime:

	var serverModule = confx.MakeModule("server",
		confx.Provide[ServerConfig]("server"),
		confx.Runtime(fx.Provide(makeServer)),
	)
	var storeModule = confx.MakeModule("store",
		confx.Provide[StoreConfig]("store"),
		confx.Runtime(fx.Provide(makeStore)),
	)
	confx.New(serverModule, storeModule).Run()

These are alternative compositions; config types and constructors belong to
the application. In domain packages, export each module so main only lists them.

New captures the composition without constructing Fx or reading configuration.
App.Run handles -help and -describe=env|markdown|json before constructing Fx.
Descriptions include defaults by default; WithManifestOptions() disables their
evaluation only for description, without changing runtime defaults or disabling
CLI. Defaults and renderers are user code and may perform their own I/O. Help
and descriptions go to stdout and return. CLI errors go to
stderr and exit with code 1. Normal execution delegates lifecycle and exit codes
to fx.App.Run. Unknown flags and positional arguments are rejected.

Use App.Manifest and App.FX for custom CLI handling or manual lifecycle control.
App has no Err, Start, Stop or Wait methods. Obtain an fx.App explicitly:

	definition := confx.New(serverModule, storeModule)
	fxApp := fx.New(definition.FX())
	if err := fxApp.Err(); err != nil {
		return err
	}

This fragment belongs inside a function returning error. FX does not parse CLI;
fx.New loads configs and runs invocations before Start. Use fxApp.Start/Stop for
manual lifecycle hooks or pass fx.Populate alongside definition.FX in tests.
The zero App is equivalent to New(). WithManifestOptions replaces description
options; WithOptions appends loader options. Both return immutable copies.

FX installs one core adapter per application. Do not add Module or FromLoader
alongside it. WithOptions returns a new composition with shared core loader
options; each Fx application gets a fresh loader.

# Registration and instances

Provide supplies an untagged value; ProvideNamed uses the registration name as
its Fx name tag. With the built-in engine, the name also determines the ENV prefix;
WithPrefix overrides only that prefix. A custom engine owns source lookup and tags.
The original Module plus Provide API remains available for ordinary Fx composition
when offline manifest collection is not needed.

# Lifecycle

Every provided config is loaded during fx.New, including configs with no
consumer. Loading errors are available from fx.App.Err and prevent startup.
Options returned by Provide and ProvideNamed may be reused across applications;
each application has its own loader and config values.

# Multiple instances and errors

Use Provide for an untagged value. Two values of the same Go type need distinct
Fx identities, even when their ENV prefixes differ. ProvideNamed uses the
instance name as that identity:

	fx.Invoke(func(params struct {
		fx.In

		Primary postgres.Config
		Replica postgres.Config `name:"replica"`
	}) {
		fmt.Println(params.Primary.Host, params.Replica.Host)
	})

Inspect construction errors with fxApp.Err() on the constructed fx.App. Core
ConfigError diagnostics remain
reachable through confmaker.ConfigErrors and errors.As. Missing Module or
conflicting Fx providers are adapter or Fx errors, not core config diagnostics.
Loaded configs should be treated as read-only, including their maps and slices.

# External registrations

[FromLoader] uses a caller-owned confmaker.Loader. [FromHandle] and
[FromHandleNamed] provide existing handles without registering them again. A
named handle uses Handle.Name as its Fx tag. All handles must belong to the same
loader. Exactly one Module or FromLoader may be installed.

For offline documentation, register configs externally and generate the manifest
before fx.New. Normal startup can then use fx.App.Run or explicit fx.App.Start/Stop. Module
accepts only Provide/ProvideNamed, while FromLoader accepts only
FromHandle/FromHandleNamed. Mixing modes or supplying a foreign handle fails
before registrations or loading and leaves the external loader unchanged.
Previously loaded loaders retain their results. Reusing an external loader
across apps shares its load state and
values. These lower-level adapters do not interpret command-line flags or exit
the process; App.Run is the opt-in process-level entry point.

# Custom engines

Module accepts confmaker.LoaderOption, including confmaker.WithEngine. An external
loader can select the same backend. Every app has its own loader, but reusing a
Module option shares the supplied Engine object, which must support concurrent
calls across loaders. Its resources remain caller-owned. Loading still occurs
at fx.New; this adapter adds no context or backend resource lifecycle.
Custom engines provide config-level reports and do not support Loader.Manifest.
Names accepted by the core remain subject to Fx named-tag restrictions.
Runnable env/v11 and koanf + TOML applications live in the separate
examples/engines module. Their dependencies are not part of this package.
ExampleModule_customEngine demonstrates a standard-library JSON decoder.

# Load diagnostics

Pass confmaker.WithDiagnosticHandler through WithOptions or Module for a value-free report
before a loading error reaches Fx, preserving fx.New(...).Run(). For explicit
inspection, enable confmaker.WithDiagnostics() on an external loader and read
loader.Report after fx.New,
before Run. External loaders accept the same core options. If Fx rejects the
graph before loading, the handler is not called and an enabled loader report
stays in the not_started state.

# Module composition and offline description

MakeModule preserves configuration declarations beside a real Fx module.
MakeModules composes one tree for Manifest and FX; WithOptions returns a copy
with shared core loader options. FX installs Module itself and creates a fresh
loader per application. Do not combine it with Module or FromLoader.

For offline description, wrap service providers, invocations and decorators in
Runtime. Runtime does not delay invocations until Start. Manifest rejects
opaque options outside Runtime with ErrOpaqueOption, before executing defaults,
and returns no partial result. Runtime asserts that its contents have no config
registrations; configurations hidden in opaque Fx wrappers cannot be discovered.
Nested MakeModule preserves module scopes. Manifest runs no Fx constructors or
invocations; IncludeDefaults still runs user defaults and renderers, which can
perform their own I/O. App.Manifest includes defaults by default; Modules.Manifest
requires IncludeDefaults explicitly. It neither
validates the Fx graph nor all load-only options. Go argument evaluation and init
are outside this guarantee. Custom engines do not support Manifest.

# Core library

The core also works without Fx:

	cfg, err := confmaker.Load[postgres.Config]("postgres")

Declaring configs, options, the manifest and loading rules are documented in
package github.com/uchaloop/confmaker/v2.
*/
package confx
