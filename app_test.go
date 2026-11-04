package confx

import (
	"context"
	"errors"
	"testing"

	"github.com/uchaloop/confmaker/v2"
	"go.uber.org/fx"
)

func TestAppLazyManifestDefaults(t *testing.T) {
	defaultsCalls = 0
	invokes, reports := 0, 0
	app := New(
		Provide[defaultConfig]("server"),
		Runtime(fx.Invoke(func() { invokes++ })),
	).WithOptions(
		confmaker.WithEnv(map[string]string{"SERVER_PORT": "invalid"}),
		confmaker.WithDiagnosticHandler(func(confmaker.LoadReport) { reports++ }),
	)
	without := app.WithManifestOptions()
	_ = app.FX()
	if defaultsCalls+invokes+reports != 0 {
		t.Fatal("constructing App or FX executed user code")
	}

	manifest, err := without.Manifest()
	if err != nil || defaultsCalls != 0 || manifest.Configs[0].Variables[0].Default.State != confmaker.DefaultNotEvaluated {
		t.Fatalf(
			"disabled defaults: manifest=%+v err=%v calls=%d",
			manifest,
			err,
			defaultsCalls,
		)
	}

	for _, item := range []struct {
		app  *App
		opts []confmaker.ManifestOption
	}{{app: app}, {app: without, opts: []confmaker.ManifestOption{confmaker.IncludeDefaults()}}} {
		manifest, err = item.app.Manifest(item.opts...)
		if err != nil || manifest.Configs[0].Variables[0].Default.Text != "8080" {
			t.Fatalf("defaults: manifest=%+v err=%v", manifest, err)
		}
	}
	if defaultsCalls != 2 || invokes+reports != 0 {
		t.Fatalf(
			"defaults=%d invokes=%d reports=%d",
			defaultsCalls,
			invokes,
			reports,
		)
	}
}

func TestAppOptionsIsolation(t *testing.T) {
	opts := []fx.Option{Provide[config]("server")}
	base := New(opts...)
	opts[0] = nil
	loaderOpts := []confmaker.LoaderOption{confmaker.WithEnv(map[string]string{"SERVER_PORT": "1"})}
	first := base.WithOptions(loaderOpts...)
	loaderOpts[0] = nil
	second := base.WithOptions(confmaker.WithEnv(map[string]string{"SERVER_PORT": "2"}))
	manifestOpts := []confmaker.ManifestOption{confmaker.IncludeDefaults()}
	first = first.WithManifestOptions(manifestOpts...)
	manifestOpts[0] = nil
	if _, err := first.Manifest(); err != nil {
		t.Fatal(err)
	}

	reports := 0
	first = first.WithOptions(confmaker.WithDiagnosticHandler(func(confmaker.LoadReport) { reports++ }))
	for _, item := range []struct {
		app  *App
		want int
	}{{app: first, want: 1}, {app: second, want: 2}, {app: first, want: 1}} {
		var got config
		app := fx.New(fx.NopLogger, item.app.FX(), fx.Populate(&got))
		if app.Err() != nil || got.Port != item.want {
			t.Fatalf(
				"err=%v got=%+v want=%d",
				app.Err(),
				got,
				item.want,
			)
		}
	}
	if reports != 2 {
		t.Fatalf("expected fresh loaders and preserved options, reports=%d", reports)
	}
}

func TestAppZero(t *testing.T) {
	var zero App
	for _, app := range []*App{&zero, New(), zero.WithManifestOptions(), zero.WithOptions(confmaker.WithEnv(map[string]string{}))} {
		manifest, err := app.Manifest()
		if err != nil || len(manifest.Configs) != 0 {
			t.Fatalf("manifest=%+v err=%v", manifest, err)
		}
		if err := fx.New(fx.NopLogger, app.FX()).Err(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAppManifestPreflight(t *testing.T) {
	defaultsCalls = 0
	opaque := New(Provide[defaultConfig]("server"), fx.Options(fx.NopLogger))
	if _, err := opaque.Manifest(); !errors.Is(err, ErrOpaqueOption) || defaultsCalls != 0 {
		t.Fatalf("opaque manifest err=%v defaults=%d", err, defaultsCalls)
	}
	if err := fx.New(fx.NopLogger, opaque.FX()).Err(); err != nil {
		t.Fatal(err)
	}

	calls := 0
	engine := confmaker.EngineFunc(func(_ context.Context, req confmaker.LoadRequest) error {
		calls++
		req.Target.(*config).Port = 42
		return nil
	})
	app := New(Provide[config]("server")).WithOptions(confmaker.WithEngine(engine))
	if _, err := app.Manifest(); !errors.Is(err, confmaker.ErrManifestUnsupported) || calls != 0 {
		t.Fatalf("engine manifest err=%v calls=%d", err, calls)
	}
	var got config
	if err := fx.New(fx.NopLogger, app.FX(), fx.Populate(&got)).Err(); err != nil || got.Port != 42 || calls != 1 {
		t.Fatalf(
			"runtime err=%v got=%+v calls=%d",
			err,
			got,
			calls,
		)
	}
}
