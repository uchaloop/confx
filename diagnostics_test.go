package confx

import (
	"testing"

	"github.com/uchaloop/confmaker"
	"go.uber.org/fx"
)

func TestDiagnosticsBeforeFxStartup(t *testing.T) {
	for _, external := range []bool{false, true} {
		for _, valid := range []bool{false, true} {
			diagnostics := confmaker.MakeDiagnostics()
			callbackCalls := 0
			env := map[string]string{}
			if valid {
				env["APP_HOST"] = "localhost"
			}
			options := []confmaker.EnvOption{confmaker.WithDiagnostics(diagnostics), confmaker.WithEnv(env), confmaker.WithDiagnosticHandler(func(report confmaker.LoadReport) {
				callbackCalls++
				if report.State != diagnostics.Report().State {
					t.Error("report not published before handler")
				}
			})}
			var registration, module fx.Option
			if external {
				loader := confmaker.MakeLoader(options...)
				handle := loader.Register[externalConfig]("app")
				module, registration = FromLoader(loader), FromHandle(handle)
			} else {
				module, registration = Module(options...), Provide[externalConfig]("app")
			}
			app := fx.New(fx.NopLogger, module, registration)
			if (app.Err() == nil) != valid || callbackCalls != 1 {
				t.Fatal(external, valid, app.Err(), callbackCalls)
			}
			expectedStatus := confmaker.LoadFailed
			if valid {
				expectedStatus = confmaker.LoadSucceeded
			}

			if diagnostics.Report().State != expectedStatus {
				t.Fatal(diagnostics.Report())
			}
		}
	}
}

func TestDiagnosticsWhenFxDoesNotLoad(t *testing.T) {
	diagnostics := confmaker.MakeDiagnostics()
	app := fx.New(fx.NopLogger, Module(confmaker.WithDiagnostics(diagnostics)), Provide[externalConfig]("app"), Provide[externalConfig]("duplicate"))
	if app.Err() == nil || diagnostics.Report().State != confmaker.LoadNotStarted {
		t.Fatal(app.Err(), diagnostics.Report())
	}
}
