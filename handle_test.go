package confx

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/uchaloop/confmaker"
	"go.uber.org/fx"
)

type externalConfig struct {
	Host string `env:"HOST,notEmpty"`
}
type externalJob struct {
	Workers int `env:"WORKERS"`
}

func TestExternalLoaderRegistrations(t *testing.T) {
	loader := confmaker.MakeLoader(confmaker.WithEnv(map[string]string{"PRIMARY_HOST": "primary", "READ_HOST": "replica", "JOB_WORKERS": "3"}))
	primary := loader.Register[externalConfig]("primary")
	replica := loader.Register[externalConfig]("replica", confmaker.WithPrefix("READ_"))
	job := loader.Register[externalJob]("job")
	var before bytes.Buffer
	if err := loader.WriteManifestJSON(&before); err != nil {
		t.Fatal(err)
	}
	if _, err := primary.Value(); !errors.Is(err, confmaker.ErrNotLoaded) {
		t.Fatal(err)
	}
	type params struct {
		fx.In
		Primary externalConfig
		Replica externalConfig `name:"replica"`
		Job     externalJob
	}
	var got params
	app := fx.New(fx.NopLogger, FromHandleNamed(replica), FromHandle(job), FromLoader(loader), FromHandle(primary), fx.Invoke(func(p params) { got = p }))
	if err := app.Err(); err != nil {
		t.Fatal(err)
	}
	if got.Primary.Host != "primary" || got.Replica.Host != "replica" || got.Job.Workers != 3 {
		t.Fatalf("values: %+v", got)
	}
	manifest, err := loader.Manifest()
	if err != nil || len(manifest) != 3 {
		t.Fatalf("manifest: %v, %v", manifest, err)
	}
}

func TestExternalLoaderFailures(t *testing.T) {
	for _, name := range []string{"nil loader", "nil handle", "zero handle", "foreign", "missing loader", "duplicate module", "duplicate loader", "unused required", "late provide"} {
		t.Run(name, func(t *testing.T) {
			loader := confmaker.MakeLoader(confmaker.WithEnv(nil))
			handle := loader.Register[externalJob]("job")
			var opts []fx.Option
			switch name {
			case "nil loader":
				opts = []fx.Option{FromLoader(nil)}
			case "nil handle":
				opts = []fx.Option{FromLoader(loader), FromHandle[externalJob](nil)}
			case "zero handle":
				opts = []fx.Option{FromLoader(loader), FromHandle(new(confmaker.Handle[externalJob]))}
			case "foreign":
				opts = []fx.Option{FromLoader(confmaker.MakeLoader()), FromHandle(handle)}
			case "missing loader":
				opts = []fx.Option{FromHandle(handle)}
			case "duplicate module":
				opts = []fx.Option{Module(), FromLoader(loader)}
			case "duplicate loader":
				opts = []fx.Option{FromLoader(loader), FromLoader(loader)}
			case "unused required":
				loader.Register[externalConfig]("unused")
				opts = []fx.Option{FromLoader(loader)}
			case "late provide":
				if err := loader.Load(); err != nil {
					t.Fatal(err)
				}
				opts = []fx.Option{FromLoader(loader), Provide[externalConfig]("late")}
			}
			app := fx.New(append([]fx.Option{fx.NopLogger}, opts...)...)
			if app.Err() == nil {
				t.Fatal("expected error")
			}
			if name == "late provide" && !strings.Contains(app.Err().Error(), "do not mix registration modes") {
				t.Fatal(app.Err())
			}
			if name == "foreign" {
				if err := loader.Load(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestExternalLoaderAlreadyLoaded(t *testing.T) {
	loader := confmaker.MakeLoader(confmaker.WithEnv(map[string]string{"APP_HOST": "database"}))
	handle := loader.Register[externalConfig]("app")
	if err := loader.Load(); err != nil {
		t.Fatal(err)
	}
	var got externalConfig
	app := fx.New(fx.NopLogger, FromLoader(loader), FromHandle(handle), fx.Populate(&got))
	if app.Err() != nil || got.Host != "database" {
		t.Fatalf("%+v %v", got, app.Err())
	}
}

func TestMixedModesDoNotChangeExternalLoader(t *testing.T) {
	for _, reversed := range []bool{false, true} {
		loader := confmaker.MakeLoader(confmaker.WithEnv(nil))
		handle := loader.Register[externalJob]("job")
		opts := []fx.Option{FromLoader(loader), Provide[externalConfig]("postgres"), FromHandle(handle)}
		if reversed {
			opts[0], opts[2] = opts[2], opts[0]
		}
		app := fx.New(append([]fx.Option{fx.NopLogger}, opts...)...)
		if app.Err() == nil || !strings.Contains(app.Err().Error(), "do not mix registration modes") {
			t.Fatalf("mode error: %v", app.Err())
		}
		manifest, err := loader.Manifest()
		if err != nil || len(manifest) != 1 {
			t.Fatalf("loader mutated: %v %v", manifest, err)
		}
		if _, err := handle.Value(); !errors.Is(err, confmaker.ErrNotLoaded) {
			t.Fatalf("loader ran: %v", err)
		}
		retry := fx.New(fx.NopLogger, FromLoader(loader), FromHandle(handle))
		if err := retry.Err(); err != nil {
			t.Fatalf("retry: %v", err)
		}
	}
}

func TestModuleRejectsExternalHandles(t *testing.T) {
	loader := confmaker.MakeLoader(confmaker.WithEnv(nil))
	handle := loader.Register[externalJob]("job")
	app := fx.New(fx.NopLogger, Module(), FromHandle(handle))
	if app.Err() == nil || !strings.Contains(app.Err().Error(), "do not mix registration modes") {
		t.Fatalf("mode error: %v", app.Err())
	}
	if _, err := handle.Value(); !errors.Is(err, confmaker.ErrNotLoaded) {
		t.Fatal(err)
	}
}

func TestForeignUnnamedHandleReportsOwnership(t *testing.T) {
	owner := confmaker.MakeLoader(confmaker.WithEnv(nil))
	handle := owner.Register[externalJob]("")
	app := fx.New(fx.NopLogger, FromLoader(confmaker.MakeLoader()), FromHandle(handle))
	if app.Err() == nil || errors.Is(app.Err(), confmaker.ErrNotLoaded) || !strings.Contains(app.Err().Error(), "not registered with the loader") {
		t.Fatalf("ownership error: %v", app.Err())
	}
}
