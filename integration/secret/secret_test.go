package integration_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/uchaloop/confmaker/v2"
	"github.com/uchaloop/confmaker/v2/confexport"
	"github.com/uchaloop/confx"
	"github.com/uchaloop/secret/v2"
	"go.uber.org/fx"
)

type config struct {
	Token secret.Secret `env:"TOKEN,required"`
}

func (c *config) SetDefaults() { c.Token = secret.New("private-default") }

func TestSecretInBothRegistrationModes(t *testing.T) {
	for _, external := range []bool{false, true} {
		name := "module"
		if external {
			name = "external"
		}
		t.Run(name, func(t *testing.T) {
			var got config
			var report confmaker.LoadReport
			opts := []confmaker.LoaderOption{
				confmaker.WithEnv(map[string]string{"APP_TOKEN": "private-input"}),
				confmaker.WithDiagnosticHandler(func(r confmaker.LoadReport) { report = r }),
			}
			wiring := []fx.Option{fx.NopLogger, fx.Populate(&got)}
			if external {
				loader := confmaker.MakeLoader(opts...)
				handle := loader.Register[config]("app")
				manifest, err := loader.Manifest(confmaker.IncludeDefaults())
				if err != nil {
					t.Fatal(err)
				}
				variable := manifest.Configs[0].Variables[0]
				if !variable.Secret || variable.Default.State != confmaker.DefaultRedacted {
					t.Fatal("secret default not redacted")
				}
				var out bytes.Buffer
				if err := confexport.WriteJSON(&out, manifest); err != nil {
					t.Fatal(err)
				}
				if strings.Contains(out.String(), "private-") {
					t.Fatal("manifest disclosed secret")
				}
				wiring = append(wiring, confx.FromLoader(loader), confx.FromHandle(handle))
			} else {
				wiring = append(wiring, confx.Module(opts...), confx.Provide[config]("app"))
			}
			if err := fx.New(wiring...).Err(); err != nil {
				t.Fatal(err)
			}
			if got.Token.Reveal() != "private-input" {
				t.Fatal("secret not loaded")
			}
			if report.State != confmaker.LoadSucceeded {
				t.Fatal("missing diagnostic report")
			}
			encoded, err := json.Marshal(report)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(encoded), "private-") {
				t.Fatal("report disclosed secret")
			}
		})
	}
}
