package confx

import (
	"errors"
	"fmt"
	"reflect"
	"slices"

	"github.com/uchaloop/confmaker/v2"
	"github.com/uchaloop/utilfx"
	"go.uber.org/fx"
)

// ErrLoaderIsNil means FromLoader was called without a loader.
var ErrLoaderIsNil = errors.New("loader is nil")

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
func Module(opts ...confmaker.LoaderOption) fx.Option {
	opts = slices.Clone(opts)

	return standaloneOption{optionBridge: makeModule(func() *confmaker.Loader { return confmaker.MakeLoader(opts...) }, providedConfigs)}
}

// FromLoader uses an existing loader instead of Module. A nil loader produces
// [ErrLoaderIsNil] in fx.App.Err. Use only FromHandle or
// FromHandleNamed with it; Provide registrations are rejected before loading.
// Load runs during fx.New, even without config consumers.
// A previously loaded loader retains its result and refuses new registrations.
// Include exactly one Module or FromLoader per app. Sharing an external loader
// across apps shares its values and one-shot load state.
func FromLoader(loader *confmaker.Loader) fx.Option {
	if loader == nil {
		return standaloneOption{optionBridge: fx.Error(ErrLoaderIsNil)}
	}

	return standaloneOption{optionBridge: makeModule(func() *confmaker.Loader { return loader }, externalHandles)}
}

func makeModule(makeLoader func() *confmaker.Loader, mode registrationMode) fx.Option {
	return fx.Module("confmaker",
		fx.Provide(fx.Annotate(
			func(registrars []registrar) (*registry, error) {
				r := &registry{
					loader:  makeLoader(),
					handles: make(map[*handleKey]any, len(registrars)),
				}
				var errs []error
				for _, registration := range registrars {
					if registration.mode != mode {
						return nil, errors.New("do not mix registration modes; use providers or handles consistently")
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

// Provide registers a config and provides it without an Fx tag. Zero or one
// instance name is accepted. An omitted or empty name asks the built-in engine
// to derive the identity and ENV prefix from T's EnvPrefix() string method.
// Custom engines require an explicit name. Use [ProvideWithOptions] for options.
// Install [Module] once with direct fx.New; [App.FX] installs it automatically.
// FromLoader cannot be mixed with Provide. Loading happens during fx.New even
// when no constructor consumes the config.
func Provide[T any](names ...string) fx.Option {
	if len(names) > 1 {
		return invalidDeclaration(errors.New("at most one instance name is allowed"))
	}

	var name string
	if len(names) == 1 {
		name = names[0]
	}

	return provide[T](name, nil, "")
}

// ProvideWithOptions registers an untagged config with typed confmaker options.
// An empty name requests the built-in engine's type-derived identity and prefix,
// as in [Provide]. WithPrefix overrides ENV lookup without changing identity.
func ProvideWithOptions[T any](name string, opts ...confmaker.ConfigOption) fx.Option {
	return provide[T](name, opts, "")
}

// ProvideNamed registers a config with a nonempty instance name and supplies it
// with the Fx tag name:"<name>". The built-in engine composes the name with T's
// optional EnvPrefix; WithPrefix overrides ENV lookup without changing the tag.
// Install [Module] once with direct fx.New; [App.FX] installs it automatically.
// Loading happens during fx.New as for [Provide].
func ProvideNamed[T any](name string, opts ...confmaker.ConfigOption) fx.Option {
	if len(name) == 0 {
		return invalidDeclaration(errors.New("an instance name is required"))
	}

	return provide[T](name, opts, utilfx.NameTag(name))
}

func invalidDeclaration(err error) fx.Option {
	return declaredOption{optionBridge: fx.Error(err), err: err}
}

// provide adds three things to the application: the registration Module
// collects, the constructor that hands out the loaded value, and an Invoke that
// asks for it even when nothing else does - so an unused config is still loaded
// and a missing Module fails the start rather than going unnoticed.
func provide[T any](name string, opts []confmaker.ConfigOption, tag string) fx.Option {
	opts = slices.Clone(opts)
	key := new(handleKey)

	option := fx.Options(
		utilfx.Grouped(registrarGroup, func() registrar {
			return registrar{
				mode: providedConfigs,
				register: func(r *registry) error {
					handle := r.loader.Register[T](name, opts...)
					if !handle.BelongsTo(r.loader) {
						_, err := handle.Value()

						return err
					}

					r.handles[key] = handle

					return nil
				},
			}
		}),
		provideHandleValue[T](key, tag),
	)

	return declaredOption{
		optionBridge: option,
		register:     func(loader *confmaker.Loader) { loader.Register[T](name, opts...) },
	}
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
		return cfg, fmt.Errorf("config %s requires a loader module", reflect.TypeFor[T]())
	}

	if err := r.loader.Load(); err != nil {
		return cfg, err
	}

	return r.handles[key].(*confmaker.Handle[T]).Value()
}
