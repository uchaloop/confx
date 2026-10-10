package integration_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/uchaloop/confmaker/v2"
	"github.com/uchaloop/confx"
	"github.com/uchaloop/secret/v2"
	"go.uber.org/fx"
)

// benchPool and benchConfig exercise a realistic config with the real secret type.
type benchPool struct {
	MaxConns        int32         `env:"MAX_CONNS"`
	MinConns        int32         `env:"MIN_CONNS"`
	MinIdleConns    int32         `env:"MIN_IDLE_CONNS"`
	MaxConnLifetime time.Duration `env:"MAX_CONN_LIFETIME"`
	MaxConnIdleTime time.Duration `env:"MAX_CONN_IDLE_TIME"`
	HealthPeriod    time.Duration `env:"HEALTH_PERIOD"`
}

type benchConfig struct {
	Host     string            `env:"HOST,notEmpty"`
	Database string            `env:"DATABASE,notEmpty"`
	User     string            `env:"USER"`
	Password secret.Secret     `env:"PASSWORD"`
	AppName  string            `env:"APP_NAME"`
	Brokers  []string          `env:"BROKERS"`
	Labels   map[string]string `env:"LABELS"`
	Pool     benchPool         `envPrefix:"POOL_"`
}

func (c *benchConfig) SetDefaults() {
	c.AppName = "bench"
	c.Pool.MaxConns = 4
	c.Pool.MaxConnLifetime = time.Hour
}

// BenchmarkModuleStart builds an Fx application with six populated instances.
// Options are prepared outside the measured loop.
func BenchmarkModuleStart(b *testing.B) {
	env := map[string]string{}
	options := []fx.Option{fx.NopLogger}
	for i := range 6 {
		name := fmt.Sprintf("confxbench%d", i)
		for variable, value := range map[string]string{
			"HOST": "localhost", "DATABASE": "test",
			"USER": "app", "PASSWORD": "benchmark-secret", "APP_NAME": "worker",
			"BROKERS": "a:9092,b:9092,c:9092", "LABELS": "env:test,team:core",
			"POOL_MAX_CONNS": "20", "POOL_MIN_CONNS": "2", "POOL_MIN_IDLE_CONNS": "1",
			"POOL_MAX_CONN_LIFETIME": "1h", "POOL_MAX_CONN_IDLE_TIME": "5m", "POOL_HEALTH_PERIOD": "30s",
		} {
			env[fmt.Sprintf("CONFXBENCH%d_%s", i, variable)] = value
		}

		options = append(options, confx.ProvideNamed[benchConfig](name))
	}

	options = append(options, confx.Module(confmaker.WithEnv(env)))

	b.ReportAllocs()
	for b.Loop() {
		if err := fx.New(options...).Err(); err != nil {
			b.Fatal(err)
		}
	}
}
