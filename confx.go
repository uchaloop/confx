package confx

import (
	"errors"
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
type registrar struct {
	mode     registrationMode
	check    func(*registry) error
	register func(*registry) error
}

type registrationMode uint8

const (
	providedConfigs registrationMode = iota
	externalHandles
)

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
// Include exactly one Module or FromLoader. Module also checks its options
// when no configs are registered.
func Module(opts ...confmaker.EnvOption) fx.Option {
	opts = slices.Clone(opts)
	return makeModule(func() *confmaker.Loader { return confmaker.MakeLoader(opts...) }, providedConfigs)
}

// FromLoader uses an existing loader instead of Module. Use only FromHandle or
// FromHandleNamed with it; Provide registrations are rejected before loading.
// Load runs during fx.New, even without config consumers.
// A previously loaded loader retains its result and refuses new registrations.
// Include exactly one Module or FromLoader per app. Sharing an external loader
// across apps shares its values and one-shot load state.
func FromLoader(loader *confmaker.Loader) fx.Option {
	if loader == nil {
		return fx.Error(errors.New("confx.FromLoader requires a non-nil loader"))
	}

	return makeModule(func() *confmaker.Loader { return loader }, externalHandles)
}

func makeModule(makeLoader func() *confmaker.Loader, mode registrationMode) fx.Option {
	return fx.Module("confmaker",
		fx.Provide(fx.Annotate(
			func(registrars []registrar) (*registry, error) {
				r := &registry{loader: makeLoader(), handles: make(map[*handleKey]any, len(registrars))}
				var errs []error
				for _, registration := range registrars {
					if registration.mode != mode {
						return nil, errors.New("confx: do not mix registration modes; use Module with Provide/ProvideNamed, or FromLoader with FromHandle/FromHandleNamed")
					}
				}

				for _, registration := range registrars {
					if registration.check != nil {
						if err := registration.check(r); err != nil {
							errs = append(errs, err)
						}
					}
				}
				if err := errors.Join(errs...); err != nil {
					return nil, err
				}

				for _, registration := range registrars {
					if err := registration.register(r); err != nil {
						errs = append(errs, err)
					}
				}
				return r, errors.Join(errs...)
			},
			fx.ParamTags(utilfx.GroupTag(registrarGroup)),
		)),
		fx.Invoke(func(r *registry) error { return r.loader.Load() }),
	)
}

// Provide registers a config with the required instance name and provides it
// without an Fx tag. confmaker validates the name and derives its default ENV
// prefix; [confmaker.WithPrefix] overrides only the prefix. [Module] is required;
// FromLoader cannot be mixed with Provide.
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
			return registrar{mode: providedConfigs, register: func(r *registry) error {
				handle := r.loader.Register[T](name, opts...)
				if !handle.BelongsTo(r.loader) {
					_, err := handle.Value()
					return err
				}
				r.handles[key] = handle
				return nil
			}}
		}),
		provideHandleValue[T](key, tag),
	)
}

func provideHandleValue[T any](key *handleKey, tag string) fx.Option {
	return fx.Options(
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
		return cfg, fmt.Errorf("config %s requires confx.Module() or confx.FromLoader() in the application", reflect.TypeFor[T]())
	}

	if err := r.loader.Load(); err != nil {
		return cfg, err
	}

	return r.handles[key].(*confmaker.Handle[T]).Value()
}
