package confx_test

import (
	"fmt"

	"github.com/uchaloop/confmaker/v2"
	"github.com/uchaloop/confx"
	"go.uber.org/fx"
)

type ServerConfig struct {
	Port int `env:"PORT"`
}

func (c *ServerConfig) SetDefaults() { c.Port = 8080 }

func ExampleNew() {
	server := fx.Module("server", fx.Invoke(func(cfg ServerConfig) {
		fmt.Println("server port:", cfg.Port)
	}))
	store := fx.Module("store", fx.Invoke(func(cfg StoreConfig) {
		fmt.Println("store host:", cfg.Host)
	}))

	// In main, Run handles -help and -describe before starting Fx.
	confx.New(
		confx.Provide[ServerConfig]("server"),
		confx.Provide[StoreConfig]("store"),
		confx.Runtime(server, store),
	).Run()
}

func ExampleApp_Manifest() {
	definition := confx.New(
		confx.Provide[ServerConfig]("server"),
		confx.Provide[StoreConfig]("store"),
	)
	manifest, err := definition.Manifest()
	if err != nil {
		fmt.Println(err)

		return
	}

	variable := manifest.Configs[0].Variables[0]
	fmt.Println(variable.Name, variable.Default.Text)
	// Output: SERVER_PORT 8080
}

// FX exposes the composition for testing and manual Fx lifecycle control.
// Unlike Run, it does not parse the process arguments.
func ExampleApp_FX() {
	definition := confx.New(
		confx.Provide[ServerConfig]("server"),
		confx.Provide[StoreConfig]("store"),
	).WithOptions(
		confmaker.WithEnv(map[string]string{
			"SERVER_PORT": "9090",
			"STORE_HOST":  "db:5432",
		}),
	)

	var serverCfg ServerConfig
	var storeCfg StoreConfig
	fxApp := fx.New(
		fx.NopLogger,
		definition.FX(),
		fx.Populate(&serverCfg, &storeCfg),
	)
	if err := fxApp.Err(); err != nil {
		fmt.Println(err)

		return
	}

	// Loading happens during fx.New; no Start is needed to inspect values.
	fmt.Println(serverCfg.Port, storeCfg.Host)
	// Output: 9090 db:5432
}
