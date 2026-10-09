package confx

import (
	"context"
	"errors"
	"github.com/uchaloop/confmaker/v2"
	"go.uber.org/fx"
	"sync"
	"sync/atomic"
	"testing"
)

type backendConfig struct {
	Value int `env:"VALUE,foreign"`
}

func TestModuleCustomEngine(t *testing.T) {
	e := confmaker.EngineFunc(func(_ context.Context, r confmaker.LoadRequest) error {
		r.Target.(*backendConfig).Value = 42
		return nil
	})
	var got backendConfig
	app := fx.New(fx.NopLogger, Module(confmaker.WithEngine(e)), Provide[backendConfig]("service/db"), fx.Populate(&got))
	if app.Err() != nil || got.Value != 42 {
		t.Fatalf("%+v %v", got, app.Err())
	}
	cause := errors.New("backend error")
	app = fx.New(fx.NopLogger, Module(confmaker.WithEngine(confmaker.EngineFunc(func(context.Context, confmaker.LoadRequest) error { return cause }))), Provide[backendConfig]("x"))
	if !errors.Is(app.Err(), cause) {
		t.Fatal(app.Err())
	}
}
func TestExternalCustomEngine(t *testing.T) {
	l := confmaker.MakeLoader(confmaker.WithEngine(confmaker.EngineFunc(func(_ context.Context, r confmaker.LoadRequest) error {
		r.Target.(*backendConfig).Value = 7
		return nil
	})), confmaker.WithDiagnostics())
	h := l.Register[backendConfig]("service/db")
	var got backendConfig
	app := fx.New(fx.NopLogger, FromLoader(l), FromHandle(h), fx.Populate(&got))
	if app.Err() != nil || got.Value != 7 || l.Report().DetailLevel != confmaker.DetailConfig {
		t.Fatalf("%+v %v %+v", got, app.Err(), l.Report())
	}
}
func TestModuleEngineReuse(t *testing.T) {
	var calls atomic.Int32
	e := confmaker.EngineFunc(func(_ context.Context, r confmaker.LoadRequest) error {
		r.Target.(*backendConfig).Value = int(calls.Add(1))
		return nil
	})
	opts := fx.Options(Module(confmaker.WithEngine(e)), Provide[backendConfig]("x"))
	var wg sync.WaitGroup
	values := make(chan int, 2)
	errs := make(chan error, 2)
	for range 2 {
		wg.Go(func() {
			var got backendConfig
			app := fx.New(fx.NopLogger, opts, fx.Populate(&got))
			errs <- app.Err()
			values <- got.Value
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	a, b := <-values, <-values
	if calls.Load() != 2 || a == b || a+b != 3 {
		t.Fatalf("calls=%d values=%d,%d", calls.Load(), a, b)
	}
}
