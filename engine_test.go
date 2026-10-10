package confx

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/uchaloop/confmaker/v2"
	"go.uber.org/fx"
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
	app := fx.New(
		fx.NopLogger,
		Module(confmaker.WithEngine(e)),
		Provide[backendConfig]("service/db"),
		fx.Populate(&got),
	)
	if app.Err() != nil || got.Value != 42 {
		t.Fatalf("%+v %v", got, app.Err())
	}
	cause := errors.New("backend error")
	failingEngine := confmaker.EngineFunc(func(context.Context, confmaker.LoadRequest) error {
		return cause
	})

	app = fx.New(
		fx.NopLogger,
		Module(confmaker.WithEngine(failingEngine)),
		Provide[backendConfig]("x"),
	)
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

func TestCustomEngineNamedRegistrations(t *testing.T) {
	for _, external := range []bool{false, true} {
		t.Run(fmt.Sprint(external), func(t *testing.T) {
			engine := confmaker.EngineFunc(func(_ context.Context, req confmaker.LoadRequest) error {
				values := map[string]int{"primary": 11, "replica": 22}
				req.Target.(*backendConfig).Value = values[req.Name]

				return nil
			})

			options := fx.Options(
				Module(confmaker.WithEngine(engine)),
				ProvideNamed[backendConfig]("primary"),
				ProvideNamed[backendConfig]("replica"),
			)
			if external {
				loader := confmaker.MakeLoader(confmaker.WithEngine(engine))
				primary := loader.Register[backendConfig]("primary")
				replica := loader.Register[backendConfig]("replica")
				options = fx.Options(FromLoader(loader), FromHandleNamed(primary), FromHandleNamed(replica))
			}

			consumed := false
			app := fx.New(fx.NopLogger, options, fx.Invoke(func(p struct {
				fx.In

				Primary backendConfig `name:"primary"`
				Replica backendConfig `name:"replica"`
			}) {
				consumed = true
				if p.Primary.Value != 11 || p.Replica.Value != 22 {
					t.Fatalf("incorrect named values: %+v", p)
				}
			}))
			if app.Err() != nil || !consumed {
				t.Fatalf("consumer=%v error=%v", consumed, app.Err())
			}
		})
	}
}

func TestCustomEngineDiagnosticCallback(t *testing.T) {
	for _, external := range []bool{false, true} {
		for _, outcome := range []string{"success", "error", "panic"} {
			t.Run(fmt.Sprintf("external=%v/%s", external, outcome), func(t *testing.T) {
				cause := errors.New("backend failure")
				engine := confmaker.EngineFunc(func(context.Context, confmaker.LoadRequest) error {
					switch outcome {
					case "error":
						return cause
					case "panic":
						panic(cause)
					}

					return nil
				})

				calls := 0
				var report confmaker.LoadReport
				opts := []confmaker.LoaderOption{
					confmaker.WithEngine(engine),
					confmaker.WithDiagnosticHandler(func(value confmaker.LoadReport) {
						calls++
						report = value
					}),
				}
				options := fx.Options(Module(opts...), Provide[backendConfig]("app"))
				if external {
					loader := confmaker.MakeLoader(opts...)
					options = fx.Options(FromLoader(loader), FromHandle(loader.Register[backendConfig]("app")))
				}

				var app *fx.App
				var recovered any
				func() {
					defer func() { recovered = recover() }()
					app = fx.New(fx.NopLogger, options)
				}()

				if outcome == "panic" {
					if recovered != cause || calls != 0 {
						t.Fatalf("panic=%v callbacks=%d", recovered, calls)
					}

					return
				}

				if recovered != nil || app == nil {
					t.Fatalf("unexpected panic: %v", recovered)
				}
				state := confmaker.LoadSucceeded
				if outcome == "error" {
					state = confmaker.LoadFailed
					if !errors.Is(app.Err(), cause) {
						t.Fatalf("lost backend error: %v", app.Err())
					}
				} else if app.Err() != nil {
					t.Fatal(app.Err())
				}

				if calls != 1 || report.State != state || report.DetailLevel != confmaker.DetailConfig {
					t.Fatalf("callbacks=%d report=%+v", calls, report)
				}
			})
		}
	}
}
