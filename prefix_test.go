package confx

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/uchaloop/confmaker/v2"
	"go.uber.org/fx"
)

type prefixConfig struct {
	Host string `env:"HOST,required"`
}

func (prefixConfig) EnvPrefix() string { return "SERVICE_" }

func TestDefaultPrefixProviders(t *testing.T) {
	opts := []confmaker.ConfigOption{confmaker.WithPrefix("CUSTOM_")}
	for name, provider := range map[string]fx.Option{
		"default":  Provide[prefixConfig](),
		"empty":    Provide[prefixConfig](""),
		"explicit": Provide[prefixConfig]("service"),
		"options":  ProvideWithOptions[prefixConfig]("", opts...),
	} {
		t.Run(name, func(t *testing.T) {
			var got struct {
				fx.In
				Default prefixConfig
				Master  prefixConfig `name:"master"`
			}
			app := New(provider, ProvideNamed[prefixConfig]("master")).WithOptions(confmaker.WithEnv(map[string]string{
				"SERVICE_HOST": "default", "CUSTOM_HOST": "default", "MASTER_SERVICE_HOST": "master",
			}))
			manifest, err := app.Manifest()
			if err != nil {
				t.Fatal(err)
			}
			if len(manifest.Configs) != 2 {
				t.Fatalf("manifest: %+v", manifest)
			}
			prefixes := map[string]string{"service": "SERVICE_", "master": "MASTER_SERVICE_"}
			if name == "options" {
				prefixes["service"] = "CUSTOM_"
			}
			for _, cfg := range manifest.Configs {
				if cfg.Prefix != prefixes[cfg.InstanceName] || len(cfg.Variables) != 1 || cfg.Variables[0].Name != cfg.Prefix+"HOST" {
					t.Fatalf("manifest identity/prefix: %+v", cfg)
				}
			}
			fxApp := fx.New(fx.NopLogger, app.FX(), fx.Populate(&got))
			if fxApp.Err() != nil {
				t.Fatal(fxApp.Err())
			}
			if got.Default.Host != "default" || got.Master.Host != "master" {
				t.Fatalf("values: %+v", got)
			}
		})
	}
}

func TestProviderDeclarationErrors(t *testing.T) {
	for name, tc := range map[string]struct {
		option fx.Option
		want   string
	}{
		"multiple":    {option: Provide[prefixConfig]("one", "two"), want: "at most one instance name"},
		"named empty": {option: ProvideNamed[prefixConfig](""), want: "an instance name is required"},
	} {
		t.Run(name, func(t *testing.T) {
			app := New(MakeModule("configs", tc.option))
			_, err := app.Manifest()
			if err == nil || errors.Is(err, ErrOpaqueOption) || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("manifest error: %v", err)
			}
			for _, option := range []fx.Option{app.FX(), fx.Options(Module(), tc.option)} {
				err := fx.New(fx.NopLogger, option).Err()
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("Fx error: %v", err)
				}
			}
		})
	}
}

func TestDefaultPrefixUnusedConfigLoads(t *testing.T) {
	err := fx.New(fx.NopLogger, New(Provide[prefixConfig]()).WithOptions(confmaker.WithEnv(nil)).FX()).Err()
	if err == nil || !strings.Contains(err.Error(), "SERVICE_HOST") {
		t.Fatalf("error: %v", err)
	}
}

func TestDefaultPrefixCustomEngineRejected(t *testing.T) {
	calls := 0
	engine := confmaker.EngineFunc(func(context.Context, confmaker.LoadRequest) error { calls++; return nil })
	err := fx.New(fx.NopLogger, Module(confmaker.WithEngine(engine)), Provide[prefixConfig]()).Err()
	if err == nil || calls != 0 {
		t.Fatalf("calls=%d error=%v", calls, err)
	}
}

func TestDefaultProviderRequiresTypePrefix(t *testing.T) {
	for name, provider := range map[string]fx.Option{
		"provide": Provide[config](),
		"options": ProvideWithOptions[config]("", confmaker.WithPrefix("OVERRIDE_")),
	} {
		t.Run(name, func(t *testing.T) {
			app := New(provider).WithOptions(confmaker.WithEnv(nil))
			if _, err := app.Manifest(); err == nil {
				t.Fatal("manifest accepted missing type prefix")
			}
			if err := fx.New(fx.NopLogger, app.FX()).Err(); err == nil {
				t.Fatal("Fx accepted missing type prefix")
			}
		})
	}
}
