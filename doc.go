/*
Package confx provides confmaker configuration values to Uber Fx. Confmaker owns
defaults, decoding, validation and diagnostics; confx owns registration and typed
injection. Import github.com/uchaloop/confmaker/v2 for core loader/config options.

# Ordinary Fx

Install [Module] once and register configs with [Provide] or [ProvideNamed].
Ordinary Fx modules need no description-specific wrappers:

	server := fx.Module("server", confx.Provide[ServerConfig]("server"))
	store := fx.Module("store", confx.Provide[StoreConfig]("store"))
	fx.New(confx.Module(), server, store).Run()

ServerConfig and StoreConfig are application types. The package example shows
both modules with consumers and an isolated ENV map. For a required field,
see the Provide example; for multiple instances, see the ProvideNamed example.

# App with help and describe

For configuration description, preserve declarations with [MakeModule] and wrap
service constructors, invocations and decorators in [Runtime]:

	server := confx.MakeModule("server",
		confx.Provide[ServerConfig]("server"),
		confx.Runtime(fx.Invoke(showServer)),
	)
	store := confx.MakeModule("store",
		confx.Provide[StoreConfig]("store"),
		confx.Runtime(fx.Invoke(showStore)),
	)
	confx.New(server, store).Run()

The New example shows both modules with consumers. [New] captures the composition
lazily. [App.Run] handles help and describe before constructing Fx, then delegates
normal startup to Fx. It may exit the process, skipping caller defers. For custom
CLI handling or manual lifecycle control, use [App.Manifest] and [App.FX]; the FX
example demonstrates injection through fx.Populate. App has no Err/Start/Stop
methods. Its method documentation defines CLI behavior and option precedence.

# Loading and ownership

All registered configs, including unused ones, load during fx.New, before Start.
Errors reach fx.App.Err and prevent startup. Runtime does not defer invocations
until Start. Module, [App.FX] and [Modules.FX] create a fresh loader per Fx app;
the FX methods install Module automatically. Do not add another adapter.
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

[MakeModules] supports composition without App. Manifest inspects MakeModule
declarations and skips Runtime; opaque options outside Runtime are rejected
before defaults run. [App.Manifest] includes defaults by default, while
[Modules.Manifest] requires confmaker.IncludeDefaults. User defaults/renderers
can perform I/O; manifest does not validate the Fx graph or all load-only options.
See Runtime for the boundary of offline inspection and the MakeModules example
for explicit description followed by Fx construction.
*/
package confx
