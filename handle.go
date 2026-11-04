package confx

import (
	"errors"
	"fmt"

	"github.com/uchaloop/confmaker/v2"
	"github.com/uchaloop/utilfx"
	"go.uber.org/fx"
)

// ErrHandleIsNil means a handle adapter was called without a handle.
var ErrHandleIsNil = errors.New("handle is nil")

// FromHandle provides an existing registration as an untagged T. FromLoader
// must reference its owner. No second registration is made. Nil, zero and foreign
// handles fail app construction before loading the application's configs.
// A nil handle produces [ErrHandleIsNil] in fx.App.Err.
func FromHandle[T any](handle *confmaker.Handle[T]) fx.Option {
	return legacyOption{optionBridge: fromHandle(handle, "")}
}

// FromHandleNamed provides an existing registration with its registration name
// as the Fx name tag. WithPrefix does not affect this name. FromLoader is required.
// A nil handle produces [ErrHandleIsNil] in fx.App.Err.
func FromHandleNamed[T any](handle *confmaker.Handle[T]) fx.Option {
	if handle == nil {
		return legacyOption{optionBridge: fx.Error(ErrHandleIsNil)}
	}

	return legacyOption{optionBridge: fromHandle(handle, utilfx.NameTag(handle.Name()))}
}

func fromHandle[T any](handle *confmaker.Handle[T], tag string) fx.Option {
	if handle == nil {
		return fx.Error(ErrHandleIsNil)
	}

	key := new(handleKey)

	return fx.Options(
		utilfx.Grouped(registrarGroup, func() registrar {
			return registrar{
				mode: externalHandles,
				check: func(r *registry) error {
					if !handle.BelongsTo(r.loader) {
						return fmt.Errorf("config handle %q is not registered with the supplied loader", handle.Name())
					}

					return nil
				},
				register: func(r *registry) error {
					r.handles[key] = handle

					return nil
				},
			}
		}),
		provideHandleValue[T](key, tag),
	)
}
