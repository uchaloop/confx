package confx

import (
	"testing"

	"github.com/uchaloop/confmaker/v2"
	"go.uber.org/fx"
)

func TestDiagnosticsBeforeFxStartup(t *testing.T) {
	for _, isExternal := range []bool{false, true} {
		for _, isValid := range []bool{false, true} {
			var captured confmaker.LoadReport
			var loader *confmaker.Loader
			callbackCalls := 0
			env := make(map[string]string)
			if isValid {
				env["APP_HOST"] = "localhost"
			}
			options := []confmaker.LoaderOption{confmaker.WithEnv(env), confmaker.WithDiagnosticHandler(func(report confmaker.LoadReport) {
				callbackCalls++
				captured = report
				if loader != nil && report.State != loader.Report().State {
					t.Error("report not published before handler")
				}
			})}
			var registration, module fx.Option
			if isExternal {
				loader = confmaker.MakeLoader(options...)
				handle := loader.Register[externalConfig]("app")
				module, registration = FromLoader(loader), FromHandle(handle)
			} else {
				module, registration = Module(options...), Provide[externalConfig]("app")
			}
			app := fx.New(fx.NopLogger, module, registration)
			if (app.Err() == nil) != isValid || callbackCalls != 1 {
				t.Fatal(
					isExternal,
					isValid,
					app.Err(),
					callbackCalls,
				)
			}
			expectedStatus := confmaker.LoadFailed
			if isValid {
				expectedStatus = confmaker.LoadSucceeded
			}

			if captured.State != expectedStatus {
				t.Fatal(captured)
			}
		}
	}
}

func TestDiagnosticsWhenFxDoesNotLoad(t *testing.T) {
	calls := 0
	app := fx.New(
		fx.NopLogger,
		Module(confmaker.WithDiagnosticHandler(func(confmaker.LoadReport) { calls++ })),
		Provide[externalConfig]("app"),
		Provide[externalConfig]("duplicate"),
	)
	if app.Err() == nil || calls != 0 {
		t.Fatal(app.Err(), calls)
	}
}
func TestModuleReuseHasIndependentReports(t *testing.T) {
	var reports []confmaker.LoadReport
	module := Module(confmaker.WithEnv(map[string]string{"APP_HOST": "local"}), confmaker.WithDiagnosticHandler(func(r confmaker.LoadReport) { reports = append(reports, r) }))
	provide := Provide[externalConfig]("app")
	for range 2 {
		if err := fx.New(fx.NopLogger, module, provide).Err(); err != nil {
			t.Fatal(err)
		}
	}
	if len(reports) != 2 {
		t.Fatal(len(reports))
	}
	reports[0].Configs[0].Variables[0].Name = "changed"
	if reports[1].Configs[0].Variables[0].Name != "APP_HOST" {
		t.Fatal("shared reports")
	}
}
