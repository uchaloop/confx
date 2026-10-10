package main

import (
	_ "embed"
	"fmt"
	"github.com/uchaloop/confmaker/v2"
	"github.com/uchaloop/confx"
	"go.uber.org/fx"
	"log"
)

//go:embed primary.toml
var primaryDocument []byte

//go:embed replica.toml
var replicaDocument []byte

type config struct {
	Port  int    `koanf:"port"`
	Label string `koanf:"label"`
}

func (c *config) SetDefaults() {
	c.Label = "local"
}

func (c *config) Validate() error {
	if c.Port <= 0 {
		return fmt.Errorf("port must be positive")
	}

	return nil
}

type service struct {
	primary config
	replica config
}

func makeService(params struct {
	fx.In

	Primary config `name:"primary"`
	Replica config `name:"replica"`
}) *service {
	return &service{
		primary: params.Primary,
		replica: params.Replica,
	}
}

func makeModule() *confx.ModuleSpec {
	return confx.MakeModule("server",
		confx.ProvideNamed[config]("primary"),
		confx.ProvideNamed[config]("replica"),
		confx.Runtime(fx.Provide(makeService)),
	)
}

func run() error {
	engine := makeEngine(map[string][]byte{
		"primary": primaryDocument,
		"replica": replicaDocument,
	}, []byte(`label = "overridden"`))
	modules := confx.MakeModules(makeModule()).WithOptions(
		confmaker.WithEngine(engine),
		confmaker.WithDiagnosticHandler(func(report confmaker.LoadReport) {
			fmt.Println("diagnostics:", report.State, report.DetailLevel)
		}),
	)

	app := fx.New(
		fx.NopLogger,
		modules.FX(),
		fx.Invoke(func(s *service) {
			fmt.Println("primary:", s.primary.Port, s.primary.Label)
			fmt.Println("replica:", s.replica.Port, s.replica.Label)
		}),
	)

	return app.Err()
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
