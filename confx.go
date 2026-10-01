package confx

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/uchaloop/confmaker"
	"github.com/uchaloop/utilfx"
	"go.uber.org/fx"
)

// registrarGroup is the value group that carries each Provide's registration to
// Module.
const registrarGroup = "confx_registrars"

// optionalRegistry lets a registration see that Module is missing and say so.
const optionalRegistry = `optional:"true"`

// registrar adds one Provide's config to an application's loader.
type registrar func(*registry)

// registry is one application's loader and the handles its Provide calls
// received. Provide options can be reused across applications, so handles are
// kept per application, keyed by the Provide call that made them.
type registry struct {
	loader  *confmaker.Loader
	handles map[*handleKey]any
}

// handleKey identifies one Provide call. It is not zero-sized: pointers to
// distinct zero-sized values may compare equal.
type handleKey struct{ _ byte }

// Module loads the configs registered by [Provide] and [ProvideNamed] through
// one confmaker.Loader per application, configured by opts. Loading happens once
// during fx.New, when a constructor or invocation requests a config. Every
// registration is checked, including unused configs. Loading errors reach
// fx.App.Err and prevent startup, regardless of registration order.
//
// An application that calls Provide or ProvideNamed must include Module exactly
// once.
func Module(opts ...confmaker.EnvOption) fx.Option {
	opts = slices.Clone(opts)

	return fx.Module(
		"confmaker",

		fx.Provide(
			fx.Annotate(
				func(registrars []registrar) *registry {
					r := &registry{
						loader:  confmaker.MakeLoader(opts...),
						handles: make(map[*handleKey]any, len(registrars)),
					}

					// Every registration is added before anything can load.
					for _, register := range registrars {
						register(r)
					}

					return r
				},
				fx.ParamTags(utilfx.GroupTag(registrarGroup)),
			),
		),
	)
}

// Provide registers a config with the required instance name and provides it
// without an Fx tag. confmaker validates the name and derives its default ENV
// prefix; [confmaker.WithPrefix] overrides only the prefix. [Module] is required.
// The config is loaded during fx.New even when no constructor consumes it.
func Provide[T any](name string, opts ...confmaker.ConfigOption) fx.Option {
	return provide[T](name, opts, "")
}

// ProvideNamed registers a config with the required instance name and provides
// it with the Fx tag name:"<name>". The name also determines the default prefix;
// [confmaker.WithPrefix] can override the prefix. [Module] is required, and
// loading happens during fx.New as for [Provide].
func ProvideNamed[T any](name string, opts ...confmaker.ConfigOption) fx.Option {
	return provide[T](name, opts, utilfx.NameTag(name))
}

// provide adds three things to the application: the registration Module
// collects, the constructor that hands out the loaded value, and an Invoke that
// asks for it even when nothing else does - so an unused config is still loaded
// and a missing Module fails the start rather than going unnoticed.
func provide[T any](name string, opts []confmaker.ConfigOption, tag string) fx.Option {
	opts = slices.Clone(opts)
	key := new(handleKey)

	return fx.Options(
		utilfx.Grouped(registrarGroup, func() registrar {
			return func(r *registry) { r.handles[key] = r.loader.Register[T](name, opts...) }
		}),
		fx.Provide(fx.Annotate(
			func(r *registry) (T, error) { return loadConfigValue[T](r, key) },
			fx.ParamTags(optionalRegistry),
			fx.ResultTags(tag),
		)),
		fx.Invoke(fx.Annotate(
			func(r *registry) error {
				_, err := loadConfigValue[T](r, key)
				return err
			},
			fx.ParamTags(optionalRegistry),
		)),
	)
}

// loadConfigValue loads the application's configs on the first call and returns T's.
func loadConfigValue[T any](r *registry, key *handleKey) (T, error) {
	var cfg T
	if r == nil {
		return cfg, fmt.Errorf("config %s requires confx.Module() in the application", reflect.TypeFor[T]())
	}

	if err := r.loader.Load(); err != nil {
		return cfg, err
	}

	return r.handles[key].(*confmaker.Handle[T]).Value()
}
