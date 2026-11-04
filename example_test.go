package confx_test

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/uchaloop/confmaker/v2"
	"github.com/uchaloop/confx"
	"go.uber.org/fx"
)

// StoreConfig is the kind of config a library declares: plain fields, env tags,
// and no knowledge of where the values come from.
type StoreConfig struct {
	Host string `env:"HOST"`
}

func (c *StoreConfig) SetDefaults() { c.Host = "localhost:5432" }

// Provide makes the config available to constructors. WithEnv stands in for the
// process environment here; an application calls confx.Module() without it.
func Example() {
	env := confmaker.WithEnv(map[string]string{"STORE_HOST": "db:5432"})

	server := fx.Module("server",
		fx.Invoke(func(cfg ServerConfig) {
			fmt.Println("server port:", cfg.Port)
		}),
	)
	store := fx.Module("store",
		fx.Invoke(func(cfg StoreConfig) {
			fmt.Println("store host:", cfg.Host)
		}),
	)

	fxApp := fx.New(
		fx.NopLogger,
		confx.Module(env),
		confx.Provide[ServerConfig]("server"),
		confx.Provide[StoreConfig]("store"),
		server,
		store,
	)
	if err := fxApp.Err(); err != nil {
		fmt.Println(err)
	}

	// Output:
	// server port: 8080
	// store host: db:5432
}

// ProvideNamed provides a second instance of the same type under an Fx tag, with
// its own prefix.
func ExampleProvideNamed() {
	env := confmaker.WithEnv(map[string]string{
		"STORE_HOST":         "db:5432",
		"REPLICA_STORE_HOST": "replica:5432",
	})

	app := fx.New(
		fx.NopLogger,
		confx.Module(env),
		confx.Provide[StoreConfig]("store"),
		confx.ProvideNamed[StoreConfig]("replica", confmaker.WithPrefix("REPLICA_STORE_")),
		fx.Invoke(func(p struct {
			fx.In

			Primary StoreConfig
			Replica StoreConfig `name:"replica"`
		}) {
			fmt.Println(p.Primary.Host, p.Replica.Host)
		}),
	)
	if err := app.Err(); err != nil {
		fmt.Println(err)
	}

	// Output:
	// db:5432 replica:5432
}

func ExampleModule_customEngine() {
	type config struct {
		Port int `json:"port"`
	}

	engine := confmaker.EngineFunc(func(ctx context.Context, req confmaker.LoadRequest) error {
		if err := ctx.Err(); err != nil {
			return err
		}

		if req.Name != "server" {
			return fmt.Errorf("unknown configuration %q", req.Name)
		}

		return json.Unmarshal([]byte(`{"port":9090}`), req.Target)
	})

	app := fx.New(
		fx.NopLogger,
		confx.Module(confmaker.WithEngine(engine)),
		confx.Provide[config]("server"),
		fx.Invoke(func(cfg config) {
			fmt.Println(cfg.Port)
		}),
	)
	if err := app.Err(); err != nil {
		fmt.Println(err)
	}

	// Output: 9090
}

// One module tree is used for offline description and normal Fx construction.
func ExampleMakeModule() {
	orders := confx.MakeModule("orders",
		confx.Provide[StoreConfig]("store"),
		confx.Runtime(
			fx.Invoke(func(cfg StoreConfig) {
				fmt.Println("loaded:", cfg.Host)
			}),
		),
	)

	modules := confx.New(orders).WithManifestOptions().WithOptions(
		confmaker.WithEnv(map[string]string{"STORE_HOST": "db:5432"}),
	)

	manifest, err := modules.Manifest()
	if err != nil {
		fmt.Println(err)

		return
	}

	fmt.Println("described:", manifest.Configs[0].InstanceName)

	app := fx.New(fx.NopLogger, modules.FX())
	if err := app.Err(); err != nil {
		fmt.Println(err)
	}

	// Output:
	// described: store
	// loaded: db:5432
}

// A config without a default can require an explicit nonempty value.
func ExampleProvide() {
	type requiredConfig struct {
		Host string `env:"HOST,notEmpty"`
	}

	fxApp := fx.New(
		fx.NopLogger,
		confx.Module(confmaker.WithEnv(map[string]string{})),
		confx.Provide[requiredConfig]("store"),
	)
	fmt.Println("missing host:", fxApp.Err() != nil)
	// Output: missing host: true
}
