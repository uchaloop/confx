package confx

import (
	"errors"
	"strings"
	"sync"
	"testing"

	c "github.com/uchaloop/confmaker/v2"
	"go.uber.org/fx"
)

type hostileOption struct{ fx.Option }

func (*hostileOption) String() string { panic("String must not run") }

func TestOpaqueOptionsRejectedBeforeDefaults(t *testing.T) {
	for _, opt := range []fx.Option{
		fx.Options(fx.Supply(1)),
		fx.Module("hidden", Provide[config]("hidden")),
		fx.Provide(func() int { return 1 }),
		&hostileOption{},
	} {
		defaultsCalls = 0
		modules := MakeModules(MakeModule("parent", Provide[defaultConfig]("first"), MakeModule("child", opt)))
		result, err := modules.Manifest(c.IncludeDefaults())
		if !errors.Is(err, ErrOpaqueOption) || !strings.Contains(err.Error(), "child/options[0]") || result.Configs != nil || defaultsCalls != 0 {
			t.Fatalf("result=%+v error=%v defaults=%d", result, err, defaultsCalls)
		}
	}
}

func TestRuntimeRejectsConfigurationOptions(t *testing.T) {
	var typedNil *ModuleSpec
	loader := c.MakeLoader()
	handle := loader.Register[config]("server")

	for _, opt := range []fx.Option{
		nil,
		typedNil,
		&ModuleSpec{},
		Provide[config]("server"),
		ProvideNamed[config]("server"),
		MakeModule("child"),
		Module(),
		FromLoader(loader),
		FromLoader(nil),
		FromHandle(handle),
		FromHandleNamed(handle),
	} {
		modules := MakeModules(Runtime(opt))
		if _, err := modules.Manifest(); err == nil {
			t.Fatal("accepted invalid Runtime")
		}
		if app := fx.New(fx.NopLogger, modules.FX()); app.Err() == nil {
			t.Fatal("FX accepted invalid Runtime")
		}
	}
}

func TestNilCompositionOptions(t *testing.T) {
	var typedNil *ModuleSpec

	for _, opt := range []fx.Option{nil, typedNil, &ModuleSpec{}} {
		modules := MakeModules(MakeModule("child", opt))
		if _, err := modules.Manifest(); err == nil {
			t.Fatal("accepted nil")
		}
		if app := fx.New(fx.NopLogger, modules.FX()); app.Err() == nil {
			t.Fatal("FX accepted nil")
		}
	}
}

func TestNamedCompositionAndDuplicateDeclarations(t *testing.T) {
	module := MakeModule("server",
		ProvideNamed[config]("primary"),
		MakeModule("replica",
			ProvideNamed[config]("replica", c.WithPrefix("READ_")),
		),
	)
	modules := MakeModules(module).WithOptions(c.WithEnv(map[string]string{"PRIMARY_PORT": "1", "READ_PORT": "2"}))

	result, err := modules.Manifest()
	if err != nil || len(result.Configs) != 2 {
		t.Fatalf("%+v %v", result, err)
	}
	consumed := false

	app := fx.New(fx.NopLogger, modules.FX(), fx.Invoke(func(p struct {
		fx.In

		Primary config `name:"primary"`
		Replica config `name:"replica"`
	}) {
		consumed = true
		if p.Primary.Port != 1 || p.Replica.Port != 2 {
			t.Fatal(p)
		}
	}))
	if app.Err() != nil || !consumed {
		t.Fatal(app.Err())
	}
	if _, err := MakeModules(module, module).Manifest(); err == nil {
		t.Fatal("duplicate silently removed")
	}
}

func TestConcurrentCompositionReuse(t *testing.T) {
	reports := make(chan c.LoadReport, 2)
	modules := MakeModules(
		MakeModule("server", Provide[config]("server")),
	).WithOptions(
		c.WithEnv(map[string]string{"SERVER_PORT": "1"}),
		c.WithDiagnosticHandler(func(report c.LoadReport) {
			reports <- report
		}),
	)
	option := modules.FX()
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() { errs <- fx.New(fx.NopLogger, option).Err() })
	}
	wg.Wait()
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	first, second := <-reports, <-reports
	first.Configs[0].Variables[0].Name = "changed"
	if second.Configs[0].Variables[0].Name == "changed" {
		t.Fatal("shared report")
	}
}

func TestEmptyCompositionAndRuntimeScope(t *testing.T) {
	modules := MakeModules()

	result, err := modules.Manifest()
	if err != nil || len(result.Configs) != 0 {
		t.Fatalf("%+v %v", result, err)
	}
	if app := fx.New(fx.NopLogger, modules.FX()); app.Err() != nil {
		t.Fatal(app.Err())
	}
	var child, sibling string
	scoped := MakeModules(MakeModule("parent",
		Runtime(fx.Decorate(func(v string) string { return v + "/parent" })),
		MakeModule("child", Runtime(fx.Invoke(func(v string) { child = v }))),
	), MakeModule("sibling", Runtime(fx.Invoke(func(v string) { sibling = v }))))
	if _, err := scoped.Manifest(); err != nil {
		t.Fatal(err)
	}

	app := fx.New(fx.NopLogger, fx.Supply("root"), scoped.FX())
	if app.Err() != nil || child != "root/parent" || sibling != "root" {
		t.Fatalf("%v %s %s", app.Err(), child, sibling)
	}
}
