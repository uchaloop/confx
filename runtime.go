package confx

import (
	"fmt"
	"slices"

	"go.uber.org/fx"
)

type runtimeOption struct {
	optionBridge
	err error
}

// Runtime marks options as containing no configuration declarations. Manifest
// skips them; Fx applies them normally in their original module scope. Known
// direct configuration options and nil options are rejected. Registrations
// hidden inside opaque fx.Options or fx.Module cannot be detected: callers must
// not hide them here. Argument evaluation and package init run normally in Go.
func Runtime(opts ...fx.Option) fx.Option {
	opts = slices.Clone(opts)
	err := checkRuntimeOptions(opts)

	var option fx.Option = fx.Options(opts...)
	if err != nil {
		option = fx.Error(err)
	}

	return runtimeOption{
		optionBridge: option,
		err:          err,
	}
}

func checkRuntimeOptions(opts []fx.Option) error {
	for i, opt := range opts {
		if nilOption(opt) {
			return fmt.Errorf("confx: nil Runtime option at index %d", i)
		}

		switch opt := opt.(type) {
		case declaredOption, *ModuleSpec, legacyOption:
			return fmt.Errorf("confx: configuration option inside Runtime at index %d", i)
		case runtimeOption:
			if opt.err != nil {
				return opt.err
			}
		}
	}

	return nil
}
