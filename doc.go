/*
Package confx provides confmaker configuration values to Uber Fx. Confmaker owns
defaults, decoding, validation and diagnostics; confx owns registration and typed
injection. Import github.com/uchaloop/confmaker/v2 for core loader/config options.

# Ordinary Fx

Install [Module] once and register configs with [Provide] or [ProvideNamed].
Ordinary Fx modules need no description-specific wrappers:

	server := fx.Module("server", fx.Invoke(showServer))
	store := fx.Module("store", fx.Invoke(showStore))
	fx.New(
		confx.Module(),
		confx.Provide[ServerConfig]("server"),
		confx.Provide[StoreConfig]("store"),
		server,
		store,
	).Run()

ServerConfig and StoreConfig are application types. The package example shows
both modules with consumers and an isolated ENV map. For a required field,
see the Provide example; for multiple instances, see the ProvideNamed example.

# App with help and describe

Declare configs at the root and wrap ordinary service modules in [Runtime]:

	server := fx.Module("server", fx.Invoke(showServer))
	store := fx.Module("store", fx.Invoke(showStore))
	confx.New(
		confx.Provide[ServerConfig]("server"),
		confx.Provide[StoreConfig]("store"),
		confx.Runtime(server, store),
	).Run()

The New example shows both service modules with consumers. [New] captures the
composition lazily. [App.Run] handles help and describe before constructing Fx,
then delegates normal startup to Fx. It may exit the process, skipping caller
defers. For custom CLI handling or manual lifecycle control, use [App.Manifest]
and [App.FX]; the FX example demonstrates injection through fx.Populate. App has
no Err/Start/Stop methods. Its method documentation defines option precedence.
For advanced nested declaration scopes, [MakeModule] preserves Fx module boundaries.

# Loading and ownership

All registered configs, including unused ones, load during fx.New, before Start.
Errors reach fx.App.Err and prevent startup. Runtime does not defer invocations
until Start. Module and [App.FX] create a fresh loader per Fx app;
App.FX installs Module automatically. Do not add another adapter.
Treat loaded config values, including maps and slices, as read-only.

Loader options may supply custom engines and diagnostic callbacks. Reused engines
and callbacks remain shared and must support concurrent use when applicable.
Engine resources remain caller-owned; confx adds no cancellation or resource
lifecycle to Fx construction. Custom engines do not support manifest/describe.

# Registration, errors and diagnostics

Provide supplies an untagged value; ProvideNamed uses the registration name as
its Fx name tag. With the built-in engine, the name also determines the ENV
prefix; confmaker.WithPrefix changes only that prefix. Multiple values of one Go
type need distinct Fx identities. A custom engine owns source lookup and tags.

Use errors.Is with [ErrLoaderIsNil], [ErrHandleIsNil] and [ErrOpaqueOption]. Core
diagnostics remain accessible through confmaker.ConfigErrors and errors.As.
Graph errors need not be configuration errors: always handle fx.App.Err itself.
confmaker.WithDiagnosticHandler provides a value-free load report; it does not
run if Fx rejects the graph before loading. Backend errors may contain values.

# External loaders and offline description

[FromLoader] uses a caller-owned loader with [FromHandle] / [FromHandleNamed].
Do not mix that mode with Module/Provide. Reusing an external loader shares its
load state and values. Its LoadContext and Report methods remain available;
the adapter documentation specifies ownership checks and nil-handle behavior.

[App.Manifest] inspects declarations, traverses MakeModule and skips Runtime.
Opaque options outside Runtime are rejected before defaults run. App includes
defaults by default; WithManifestOptions() disables their evaluation. User
defaults/renderers can perform I/O; manifest does not validate the Fx graph or
all load-only options. See Runtime for the boundary of offline inspection.

# Migration from 0.6.x

App replaces the removed Modules and MakeModules API. Replace
MakeModules(opts...) with New(opts...).WithManifestOptions() to preserve the
old manifest defaults policy. MakeModule now returns fx.Option; use that type
instead of the removed ModuleSpec in application signatures. The internal
module representation is private. External loader adapters remain supported.
See the README migration table.
*/
package confx
