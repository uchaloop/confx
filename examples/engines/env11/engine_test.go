package main

import (
	"context"
	"errors"
	"testing"

	"github.com/uchaloop/confmaker/v2"
	"github.com/uchaloop/confx"
	"go.uber.org/fx"
)

func TestEngineThroughFx(t *testing.T) {
	for _, value := range []string{"8080", "0", "bad"} {
		t.Run(value, func(t *testing.T) {
			engine := makeEngine(map[string]map[string]string{
				"primary": {"PORT": value},
				"replica": {"PORT": "9090"},
			})

			var got *service
			var report confmaker.LoadReport
			callbacks := 0
			modules := confx.MakeModules(makeModule()).WithOptions(
				confmaker.WithEngine(engine),
				confmaker.WithDiagnosticHandler(func(result confmaker.LoadReport) {
					callbacks++
					report = result
				}),
			)

			app := fx.New(
				fx.NopLogger,
				modules.FX(),
				fx.Populate(&got),
			)
			valid := value == "8080"
			if valid {
				if app.Err() != nil || got == nil {
					t.Fatalf("service=%v error=%v", got, app.Err())
				}
				if got.primary.Port != 8080 || got.replica.Port != 9090 || got.primary.Label != "local" {
					t.Fatalf("unexpected values: %+v", got)
				}
			} else if app.Err() == nil || got != nil {
				t.Fatalf("failed configuration constructed service: %v %v", got, app.Err())
			}

			state := confmaker.LoadFailed
			if valid {
				state = confmaker.LoadSucceeded
			}
			if callbacks != 1 || report.State != state || report.DetailLevel != confmaker.DetailConfig {
				t.Fatalf("callbacks=%d report=%+v", callbacks, report)
			}

			if _, err := modules.Manifest(); !errors.Is(err, confmaker.ErrManifestUnsupported) {
				t.Fatalf("unexpected manifest error: %v", err)
			}
		})
	}
}

func TestEngineRejectsUnknownRegistrationAndCancellation(t *testing.T) {
	value := "8080"
	engine := makeEngine(map[string]map[string]string{
		"primary": {"PORT": value},
		"replica": {"PORT": "9090"},
	})
	if err := engine.Load(context.Background(), confmaker.LoadRequest{Name: "missing", Target: &config{}}); err == nil {
		t.Fatal("missing registration accepted")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := engine.Load(ctx, confmaker.LoadRequest{Name: "primary", Target: &config{}}); !errors.Is(err, context.Canceled) {
		t.Fatalf("unexpected cancellation: %v", err)
	}
}

func Example() {
	main()

	// Output:
	// diagnostics: succeeded config
	// primary: 8080 local
	// replica: 9090 local
}

func TestEngineSnapshotsEnvironment(t *testing.T) {
	t.Setenv("PORT", "9999")

	values := map[string]string{}
	environments := map[string]map[string]string{"primary": values}
	engine := makeEngine(environments)
	values["PORT"] = "1234"
	delete(environments, "primary")

	cfg, err := confmaker.Load[config]("primary", confmaker.WithEngine(engine))
	if err != nil || cfg.Port != 8080 || cfg.Label != "local" {
		t.Fatalf("snapshot or native default changed: %+v %v", cfg, err)
	}
}
