package confx

import (
	"context"
	"errors"
	"testing"

	c "github.com/uchaloop/confmaker/v2"
	"go.uber.org/fx"
)

type config struct {
	Port int `env:"PORT,required"`
}
type nestedConfig struct {
	Host string `env:"HOST,required"`
}
type privateValue struct{}
type scopeValue string

func TestNestedManifestRunsNoApplicationCode(t *testing.T) {
	constructors, invokes, hooks, callbacks := 0, 0, 0, 0
	modules := MakeModules(MakeModule("parent",
		Provide[config]("server", c.WithPrefix("WEB_")),
		MakeModule("child", Provide[nestedConfig]("db")),
		Runtime(fx.Provide(func() *privateValue {
			constructors++

			return &privateValue{}
		}),
			fx.Invoke(func(*privateValue) { invokes++ }),
			fx.Invoke(func(lc fx.Lifecycle) {
				lc.Append(fx.Hook{OnStart: func(context.Context) error {
					hooks++

					return nil
				}})
			})),
	)).WithOptions(c.WithEnv(map[string]string{"WEB_PORT": "9090", "DB_HOST": "localhost"}), c.WithDiagnosticHandler(func(c.LoadReport) { callbacks++ }))

	manifest, err := modules.Manifest()
	if err != nil || len(manifest.Configs) != 2 {
		t.Fatalf("manifest=%+v err=%v", manifest, err)
	}
	if constructors+invokes+hooks+callbacks != 0 {
		t.Fatal("manifest executed application code")
	}
	if manifest.Configs[1].Prefix != "WEB_" {
		t.Fatal(manifest)
	}

	var got config

	app := fx.New(fx.NopLogger, modules.FX(), fx.Populate(&got))
	if app.Err() != nil || got.Port != 9090 || constructors != 1 || invokes != 1 || callbacks != 1 || hooks != 0 {
		t.Fatalf("app=%v got=%+v counts=%d/%d/%d/%d", app.Err(), got, constructors, invokes, hooks, callbacks)
	}
}

func TestPrivateVisibilityAndDecorate(t *testing.T) {
	var inside, child, outside scopeValue
	module := MakeModule("parent",
		fx.Provide(func() *privateValue { return &privateValue{} }, fx.Private),
		fx.Decorate(func(v scopeValue) scopeValue { return v + "/parent" }),
		fx.Invoke(func(v scopeValue, _ *privateValue) { inside = v }),
		MakeModule("child", fx.Invoke(func(v scopeValue, _ *privateValue) { child = v })),
	)
	modules := MakeModules(module)

	app := fx.New(fx.NopLogger, fx.Supply(scopeValue("root")), modules.FX(), fx.Invoke(func(v scopeValue) { outside = v }))
	if app.Err() != nil || inside != "root/parent" || child != inside || outside != "root" {
		t.Fatalf("%v scopes=%q/%q/%q", app.Err(), inside, child, outside)
	}
	app = fx.New(fx.NopLogger, fx.Supply(scopeValue("root")), modules.FX(), fx.Invoke(func(*privateValue) {}))
	if app.Err() == nil {
		t.Fatal("private value escaped module")
	}
}

func TestOptionsIsolationAndLoaderReuse(t *testing.T) {
	base := MakeModules(MakeModule("server", Provide[config]("server")))
	first := base.WithOptions(c.WithEnv(map[string]string{"SERVER_PORT": "1"}))
	second := base.WithOptions(c.WithEnv(map[string]string{"SERVER_PORT": "2"}))
	if len(base.options) != 0 {
		t.Fatal("mutated base")
	}
	for _, item := range []struct {
		m    *Modules
		want int
	}{{first, 1}, {second, 2}, {first, 1}} {
		var got config
		app := fx.New(fx.NopLogger, item.m.FX(), fx.Populate(&got))
		if app.Err() != nil || got.Port != item.want {
			t.Fatalf("%v %+v", app.Err(), got)
		}
	}
}

func TestCustomEngineManifestDoesNotCallEngine(t *testing.T) {
	calls := 0
	modules := MakeModules(MakeModule("server", Provide[config]("server"))).WithOptions(c.WithEngine(c.EngineFunc(func(context.Context, c.LoadRequest) error { calls++; return nil })))
	_, err := modules.Manifest()
	if !errors.Is(err, c.ErrManifestUnsupported) || calls != 0 {
		t.Fatalf("%v calls=%d", err, calls)
	}
}

func TestOpaqueFxWrapperHidesMetadata(t *testing.T) {
	modules := MakeModules(fx.Module("opaque", Provide[config]("server")))

	manifest, err := modules.Manifest()
	if !errors.Is(err, ErrOpaqueOption) || len(manifest.Configs) != 0 {
		t.Fatalf("expected documented opaque boundary: %+v %v", manifest, err)
	}
}

var defaultsCalls int

type defaultConfig struct {
	Port int `env:"PORT"`
}

func (c *defaultConfig) SetDefaults() { defaultsCalls++; c.Port = 8080 }

func TestManifestDefaultsAreExplicit(t *testing.T) {
	defaultsCalls = 0
	modules := MakeModules(MakeModule("server", Provide[defaultConfig]("server")))
	if _, err := modules.Manifest(); err != nil {
		t.Fatal(err)
	}
	if defaultsCalls != 0 {
		t.Fatal("schema executed defaults")
	}

	result, err := modules.Manifest(c.IncludeDefaults())
	if err != nil || defaultsCalls != 1 || result.Configs[0].Variables[0].Default.Text != "8080" {
		t.Fatalf("%+v %v calls=%d", result, err, defaultsCalls)
	}
}

func TestManifestDoesNotValidateLoadOnlyOptions(t *testing.T) {
	modules := MakeModules(MakeModule("server", Provide[config]("server"))).WithOptions(nil)
	if _, err := modules.Manifest(); err != nil {
		t.Fatal(err)
	}
	if app := fx.New(fx.NopLogger, modules.FX()); app.Err() == nil {
		t.Fatal("load must reject nil option")
	}
}
