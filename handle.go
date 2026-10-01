package confx

import (
	"errors"
	"fmt"

	"github.com/uchaloop/confmaker"
	"github.com/uchaloop/utilfx"
	"go.uber.org/fx"
)

// FromHandle provides an existing registration as an untagged T. FromLoader
// must reference its owner. No second registration is made. Nil, zero and foreign
// handles fail app construction before loading the application's configs.
func FromHandle[T any](handle *confmaker.Handle[T]) fx.Option {
	return fromHandle(handle, "")
}

// FromHandleNamed provides an existing registration with its registration name
// as the Fx name tag. WithPrefix does not affect this name. FromLoader is required.
func FromHandleNamed[T any](handle *confmaker.Handle[T]) fx.Option {
	if handle == nil {
		return fx.Error(errors.New("confx.FromHandleNamed requires a non-nil handle"))
	}

	return fromHandle(handle, utilfx.NameTag(handle.Name()))
}

func fromHandle[T any](handle *confmaker.Handle[T], tag string) fx.Option {
	if handle == nil {
		return fx.Error(errors.New("confx.FromHandle requires a non-nil handle"))
	}

	key := new(handleKey)
	return fx.Options(
		utilfx.Grouped(registrarGroup, func() registrar {
			return registrar{
				mode: externalHandles,
				check: func(r *registry) error {
					if !handle.BelongsTo(r.loader) {
						return fmt.Errorf("config handle %q is not registered with the loader supplied to confx.FromLoader; use a valid handle from that loader", handle.Name())
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
