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
var ErrOpaqueOption = errors.New("opaque option")

// optionBridge promotes Fx's sealed Option interface without exposing a mutable
// embedded Option field on moduleSpec.
type optionBridge interface{ fx.Option }

type declaredOption struct {
	optionBridge
	register func(*confmaker.Loader)
}

type standaloneOption struct{ optionBridge }

// moduleSpec preserves declarations and the original Fx module scope.
type moduleSpec struct {
	optionBridge
	name    string
	options []fx.Option
}

// MakeModule groups declarations and runtime options, preserving Fx scopes.
// Manifest rejects opaque options unless explicitly enclosed in Runtime.
func MakeModule(name string, opts ...fx.Option) fx.Option {
	opts = slices.Clone(opts)
	option := fx.Module(name, opts...)
	if err := inspectOptions(
		opts,
		name,
		false,
		nil,
	); err != nil {
		option = fx.Error(err)
	}

	return &moduleSpec{
		optionBridge: option,
		name:         name,
		options:      opts,
	}
}

func inspectOptions(
	opts []fx.Option,
	path string,
	isStrict bool,
	declarations *[]func(*confmaker.Loader),
) error {
	for i, opt := range opts {
		location := fmt.Sprintf("%s/options[%d]", path, i)
		if isNilOption(opt) {
			return fmt.Errorf("nil option at %s", location)
		}

		switch opt := opt.(type) {
		case declaredOption:
			if declarations != nil {
				*declarations = append(*declarations, opt.register)
			}
		case *moduleSpec:
			if opt.optionBridge == nil {
				return fmt.Errorf("invalid module at %s", location)
			}

			if err := inspectOptions(
				opt.options,
				location+"/"+opt.name,
				isStrict,
				declarations,
			); err != nil {
				return err
			}
		case runtimeOption:
			if opt.err != nil {
				return fmt.Errorf("%s: %w", location, opt.err)
			}
		case standaloneOption:
			return fmt.Errorf("standalone loader or handle option at %s; use Provide with this composition", location)
		default:
			if isStrict {
				return fmt.Errorf("%w at %s; use MakeModule or explicitly mark service options with Runtime", ErrOpaqueOption, location)
			}
		}
	}

	return nil
}

func isNilOption(opt fx.Option) bool {
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
