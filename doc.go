/*
Package confx adapts github.com/uchaloop/confmaker to Uber Fx. The application
explicitly registers its package configs; confmaker owns ENV parsing, defaults,
validation and diagnostics, while confx provides the resulting typed values.

Import github.com/uchaloop/confx for registration and
github.com/uchaloop/confmaker when using core options such as WithEnv or
WithPrefix.

# Registration

	fx.New(
		confx.Module(),
		confx.Provide[postgres.Config]("postgres"),
		confx.ProvideNamed[postgres.Config]("replica", confmaker.WithPrefix("REPLICA_POSTGRES_")),
	)

[Module] creates one confmaker.Loader per application and is required once when
configs are registered, unless [FromLoader] supplies an existing loader instead. [Provide] passes the explicit instance name to the core
and provides the result without an Fx tag. [ProvideNamed] additionally uses that
name as the Fx tag name:"<name>". confmaker validates names, derives ENV prefixes
and loads the configurations; the adapter handles their registration in Fx.

# Lifecycle

Every provided config is loaded during fx.New, including configs with no
consumer. Loading errors are available from fx.App.Err and prevent startup.
Options returned by Provide and ProvideNamed may be reused across applications;
each application has its own loader and config values.

# Multiple instances and errors

Use Provide for an untagged value. Two values of the same Go type need distinct
Fx identities, even when their ENV prefixes differ. ProvideNamed uses the
instance name as that identity:

	type Params struct {
		fx.In
		Primary postgres.Config
		Replica postgres.Config `name:"replica"`
	}

Inspect construction errors with app.Err(). Core ConfigError diagnostics remain
reachable through confmaker.ConfigErrors and errors.As. Missing Module or
conflicting Fx providers are adapter or Fx errors, not core config diagnostics.
Loaded configs should be treated as read-only, including their maps and slices.

# External registrations

[FromLoader] uses a caller-owned confmaker.Loader. [FromHandle] and
[FromHandleNamed] provide existing handles without registering them again. A
named handle uses Handle.Name as its Fx tag. All handles must belong to the same
loader. Exactly one Module or FromLoader may be installed.

For offline documentation, register configs externally and generate the manifest
before fx.New. Normal startup can then use Run or explicit Start/Stop. Module accepts only Provide/ProvideNamed, while FromLoader accepts only
FromHandle/FromHandleNamed. Mixing modes or supplying a foreign handle fails
before registrations or loading and leaves the external loader unchanged.
Previously loaded loaders retain their results. Reusing an external loader across apps shares its load state and
values. The adapter does not interpret command-line flags or exit the process.

# Load diagnostics

Pass confmaker.WithDiagnosticHandler to Module to process a value-free report
before a loading error reaches Fx, preserving fx.New(...).Run(). For explicit
inspection, pass confmaker.WithDiagnostics and read the receiver after fx.New,
before Run. External loaders accept the same core options. If Fx rejects the
graph before loading, the handler is not called and the receiver stays not started.

# Core library

The core also works without Fx:

	cfg, err := confmaker.Load[postgres.Config]("postgres")

Declaring configs, options, the manifest and loading rules are documented in
package github.com/uchaloop/confmaker.
*/
package confx
