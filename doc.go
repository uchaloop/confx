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
configs are registered. [Provide] passes the explicit instance name to the core
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

# Core library

The core also works without Fx:

	cfg, err := confmaker.Load[postgres.Config]("postgres")

Declaring configs, options, the manifest and loading rules are documented in
package github.com/uchaloop/confmaker.
*/
package confx
