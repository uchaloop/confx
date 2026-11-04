package confx

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/uchaloop/confmaker/v2"
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
		modules := New(MakeModule("parent", Provide[defaultConfig]("first"), MakeModule("child", opt))).WithManifestOptions()
		result, err := modules.Manifest(confmaker.IncludeDefaults())
		if !errors.Is(err, ErrOpaqueOption) || !strings.Contains(err.Error(), "child/options[0]") || result.Configs != nil || defaultsCalls != 0 {
			t.Fatalf(
				"result=%+v error=%v defaults=%d",
				result,
				err,
				defaultsCalls,
			)
		}
	}
}

func TestRuntimeRejectsConfigurationOptions(t *testing.T) {
	var typedNil *moduleSpec
	loader := confmaker.MakeLoader()
	handle := loader.Register[config]("server")

	for _, opt := range []fx.Option{
		nil,
		typedNil,
		&moduleSpec{},
		Provide[config]("server"),
		ProvideNamed[config]("server"),
		MakeModule("child"),
		Module(),
		FromLoader(loader),
		FromLoader(nil),
		FromHandle(handle),
		FromHandleNamed(handle),
	} {
		modules := New(Runtime(opt)).WithManifestOptions()
		if _, err := modules.Manifest(); err == nil {
			t.Fatal("accepted invalid Runtime")
		}
		if app := fx.New(fx.NopLogger, modules.FX()); app.Err() == nil {
			t.Fatal("FX accepted invalid Runtime")
		}
	}
}

func TestNilCompositionOptions(t *testing.T) {
	var typedNil *moduleSpec

	for _, opt := range []fx.Option{nil, typedNil, &moduleSpec{}} {
		modules := New(MakeModule("child", opt)).WithManifestOptions()
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
			ProvideNamed[config]("replica", confmaker.WithPrefix("READ_")),
		),
	)
	modules := New(module).WithManifestOptions().WithOptions(confmaker.WithEnv(map[string]string{"PRIMARY_PORT": "1", "READ_PORT": "2"}))

	result, err := modules.Manifest()
	if err != nil || len(result.Configs) != 2 {
		t.Fatalf("%+v %v", result, err)
	}
	isConsumed := false

	app := fx.New(fx.NopLogger, modules.FX(), fx.Invoke(func(p struct {
		fx.In

		Primary config `name:"primary"`
		Replica config `name:"replica"`
	}) {
		isConsumed = true
		if p.Primary.Port != 1 || p.Replica.Port != 2 {
			t.Fatal(p)
		}
	}))
	if app.Err() != nil || !isConsumed {
		t.Fatal(app.Err())
	}
	if _, err := New(module, module).WithManifestOptions().Manifest(); err == nil {
		t.Fatal("duplicate silently removed")
	}
}

func TestConcurrentCompositionReuse(t *testing.T) {
	reports := make(chan confmaker.LoadReport, 2)
	modules := New(
		MakeModule("server", Provide[config]("server")),
	).WithManifestOptions().WithOptions(
		confmaker.WithEnv(map[string]string{"SERVER_PORT": "1"}),
		confmaker.WithDiagnosticHandler(func(report confmaker.LoadReport) {
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
	modules := New().WithManifestOptions()

	result, err := modules.Manifest()
	if err != nil || len(result.Configs) != 0 {
		t.Fatalf("%+v %v", result, err)
	}
	if app := fx.New(fx.NopLogger, modules.FX()); app.Err() != nil {
		t.Fatal(app.Err())
	}
	var child, sibling string
	scoped := New(MakeModule("parent",
		Runtime(fx.Decorate(func(v string) string { return v + "/parent" })),
		MakeModule("child", Runtime(fx.Invoke(func(v string) { child = v }))),
	), MakeModule("sibling", Runtime(fx.Invoke(func(v string) { sibling = v })))).WithManifestOptions()
	if _, err := scoped.Manifest(); err != nil {
		t.Fatal(err)
	}

	app := fx.New(fx.NopLogger, fx.Supply("root"), scoped.FX())
	if app.Err() != nil || child != "root/parent" || sibling != "root" {
		t.Fatalf(
			"%v %s %s",
			app.Err(),
			child,
			sibling,
		)
	}
}
