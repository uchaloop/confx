package confx_test

import (
	"reflect"
	"testing"

	"github.com/uchaloop/confmaker/v2"
	"github.com/uchaloop/confx"
	"go.uber.org/fx"
)

func TestAppMatchesOrdinaryFx(t *testing.T) {
	for _, port := range []string{"9090", "invalid"} {
		var directReport, report confmaker.LoadReport
		directCalls, calls := 0, 0
		env := confmaker.WithEnv(map[string]string{"SERVER_PORT": port})
		directOptions := confx.Module(env, confmaker.WithDiagnosticHandler(func(r confmaker.LoadReport) {
			directCalls++
			directReport = r
		}))
		options := confx.New(confx.Provide[ServerConfig]("server")).WithOptions(env, confmaker.WithDiagnosticHandler(func(r confmaker.LoadReport) {
			calls++
			report = r
		}))

		var directConfig, cfg ServerConfig
		directApp := fx.New(
			fx.NopLogger,
			directOptions,
			confx.Provide[ServerConfig]("server"),
			fx.Populate(&directConfig),
		)
		app := fx.New(fx.NopLogger, options.FX(), fx.Populate(&cfg))
		isValid := port == "9090"
		if (directApp.Err() == nil) != isValid || (app.Err() == nil) != isValid {
			t.Fatalf("unexpected load result: direct=%v current=%v", directApp.Err(), app.Err())
		}

		if directCalls != 1 || calls != 1 || !reflect.DeepEqual(directReport, report) || directConfig != cfg {
			t.Fatalf("load mismatch: direct=%+v current=%+v", directReport, report)
		}
	}
}

func TestMakeModuleOption(t *testing.T) {
	var makeModule func(string, ...fx.Option) fx.Option = confx.MakeModule
	var option fx.Option = makeModule("server", confx.Provide[ServerConfig]("server"))
	manifest, err := confx.New(option).Manifest()
	if err != nil || len(manifest.Configs) != 1 {
		t.Fatalf("unexpected manifest: %+v err=%v", manifest, err)
	}
}
