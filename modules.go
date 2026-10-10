package confx

import (
	"errors"
	"fmt"
	"reflect"
	"slices"

	"github.com/uchaloop/confmaker/v2"
	"go.uber.org/fx"
)

// ErrOpaqueOption means Manifest cannot inspect an option. Use MakeModule for
// configuration modules and Runtime for options containing no registrations.
var ErrOpaqueOption = errors.New("confx: opaque Fx option")

// optionBridge promotes Fx's sealed Option interface without exposing a mutable
// embedded Option field on ModuleSpec.
type optionBridge interface{ fx.Option }

type declaredOption struct {
	optionBridge
	register func(*confmaker.Loader)
}

type legacyOption struct{ optionBridge }

// ModuleSpec preserves configuration declarations alongside a real Fx module.
// Create it with MakeModule. Its zero value is invalid.
type ModuleSpec struct {
	optionBridge
	name    string
	options []fx.Option
}

// MakeModule groups declarations and runtime options, preserving Fx scopes.
// Manifest rejects opaque options unless explicitly enclosed in Runtime.
func MakeModule(name string, opts ...fx.Option) *ModuleSpec {
	opts = slices.Clone(opts)
	option := fx.Module(name, opts...)
	if err := inspectOptions(opts, name, false, nil); err != nil {
		option = fx.Error(err)
	}

	return &ModuleSpec{optionBridge: option, name: name, options: opts}
}

// Modules is an immutable application composition. Use MakeModules to create it.
// A zero Modules describes an empty application.
type Modules struct {
	modules []fx.Option
	options []confmaker.LoaderOption
}

// MakeModules captures one module list for Manifest and FX.
func MakeModules(modules ...fx.Option) *Modules {
	return &Modules{modules: slices.Clone(modules)}
}

// WithOptions returns a copy with additional shared core options. Engine objects
// and callback closures are shared; the loader itself is never shared.
func (m *Modules) WithOptions(opts ...confmaker.LoaderOption) *Modules {
	options := append(slices.Clone(m.options), opts...)

	return &Modules{modules: m.modules, options: options}
}

// Manifest describes all visible declarations without constructing an Fx app.
// It first rejects opaque branches, then delegates to the core. IncludeDefaults
// executes user defaults and renderers. This does not validate the Fx graph or
// all load-only options. A custom engine returns ErrManifestUnsupported.
func (m *Modules) Manifest(opts ...confmaker.ManifestOption) (confmaker.ManifestResult, error) {
	var declarations []func(*confmaker.Loader)
	if err := inspectOptions(m.modules, "root", true, &declarations); err != nil {
		return confmaker.ManifestResult{}, err
	}

	loader := confmaker.MakeLoader(m.options...)
	for _, register := range declarations {
		register(loader)
	}

	return loader.Manifest(opts...)
}

// FX builds an Fx option with one fresh loader per application. Do not combine
// it with Module or FromLoader. Ordinary opaque Fx options work in this path.
func (m *Modules) FX() fx.Option {
	if err := inspectOptions(m.modules, "root", false, nil); err != nil {
		return fx.Error(err)
	}

	opts := []fx.Option{Module(m.options...)}

	return fx.Options(append(opts, m.modules...)...)
}

func inspectOptions(opts []fx.Option, path string, strict bool, declarations *[]func(*confmaker.Loader)) error {
	for i, opt := range opts {
		location := fmt.Sprintf("%s/options[%d]", path, i)
		if nilOption(opt) {
			return fmt.Errorf("confx: nil option at %s", location)
		}

		switch opt := opt.(type) {
		case declaredOption:
			if declarations != nil {
				*declarations = append(*declarations, opt.register)
			}
		case *ModuleSpec:
			if opt.optionBridge == nil {
				return fmt.Errorf("confx: zero ModuleSpec at %s", location)
			}
			if err := inspectOptions(opt.options, location+"/"+opt.name, strict, declarations); err != nil {
				return err
			}
		case runtimeOption:
			if opt.err != nil {
				return fmt.Errorf("%s: %w", location, opt.err)
			}
		case legacyOption:
			return fmt.Errorf("confx: legacy loader or handle option at %s; use Provide with this composition", location)
		default:
			if strict {
				return fmt.Errorf("%w at %s; use MakeModule or explicitly mark service options with Runtime", ErrOpaqueOption, location)
			}
		}
	}

	return nil
}

func nilOption(opt fx.Option) bool {
	if opt == nil {
		return true
	}

	value := reflect.ValueOf(opt)
	switch value.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return value.IsNil()
	}

	return false
}
