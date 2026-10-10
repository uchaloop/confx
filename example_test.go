package confx_test

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/uchaloop/confmaker/v2"
	"github.com/uchaloop/confx"
	"go.uber.org/fx"
)

// StoreConfig is the kind of config a library declares: plain fields, env tags,
// and no knowledge of where the values come from.
type StoreConfig struct {
	Host    string        `env:"HOST,notEmpty"`
	Timeout time.Duration `env:"TIMEOUT"`
}

func (c *StoreConfig) SetDefaults() { c.Timeout = 30 * time.Second }

// Provide makes the config available to constructors. WithEnv stands in for the
// process environment here; an application calls confx.Module() without it.
func Example() {
	env := confmaker.WithEnv(map[string]string{"STORE_HOST": "db:5432"})

	app := fx.New(
		fx.NopLogger,
		confx.Module(env),
		confx.Provide[StoreConfig]("store"),
		fx.Invoke(func(cfg StoreConfig) {
			fmt.Println(cfg.Host, cfg.Timeout)
		}),
	)
	if err := app.Err(); err != nil {
		fmt.Println(err)
	}

	// Output:
	// db:5432 30s
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
func ExampleMakeModules() {
	orders := confx.MakeModule("orders",
		confx.Provide[StoreConfig]("store"),
		confx.Runtime(
			fx.Invoke(func(cfg StoreConfig) {
				fmt.Println("loaded:", cfg.Host)
			}),
		),
	)

	modules := confx.MakeModules(orders).WithOptions(
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
