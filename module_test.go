package confx

import (
	"strings"
	"sync/atomic"
	"testing"

	"github.com/uchaloop/confmaker"
	"go.uber.org/fx"
)

// Tests pass their environment through confmaker.WithEnv and run in parallel,
// except those that count SetDefaults calls in a shared counter.

type loaderConfig struct {
	Host string `env:"HOST,notEmpty"`
}

var loaderDefaultCalls atomic.Int32

func (*loaderConfig) SetDefaults() { loaderDefaultCalls.Add(1) }

type loaderOtherConfig struct {
	Port int `env:"PORT,required"`
}

type unnamedConfig struct {
	Host string `env:"HOST"`
}

type namedConfig struct {
	Host string `env:"HOST,notEmpty"`
}

func TestProvideWithoutModuleFailsTheStart(t *testing.T) {
	t.Parallel()

	cases := map[string]fx.Option{
		"unused":   Provide[loaderConfig]("confxloader"),
		"consumed": fx.Options(Provide[loaderConfig]("confxloader"), fx.Invoke(func(loaderConfig) {})),
		"named":    ProvideNamed[loaderConfig]("confxloader"),
	}

	for name, option := range cases {
		t.Run(name, func(t *testing.T) {
			err := fx.New(fx.NopLogger, option).Err()
			if err == nil || !strings.Contains(err.Error(), `config confx.loaderConfig requires confx.Module()`) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestModuleChecksAnUnusedConfig(t *testing.T) {
	t.Parallel()

	err := fx.New(fx.NopLogger, Module(confmaker.WithEnv(nil)), Provide[loaderConfig]("confxloader")).Err()
	if err == nil || !strings.Contains(err.Error(), `config "confxloader": required variable "CONFXLOADER_HOST" is not set`) {
		t.Fatalf("an unused config went unchecked: %v", err)
	}
}

func TestModuleReportsEveryConfigAndTypoAtOnce(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"CONFXLOADER_HSOT": "typo",
		"CONFXOTHER_PORT":  "eighty",
	}

	err := fx.New(
		fx.NopLogger,
		Module(confmaker.WithEnv(env)),
		Provide[loaderConfig]("confxloader"),
		Provide[loaderOtherConfig]("confxother"),
		Provide[unnamedConfig](""),
		fx.Invoke(func(loaderConfig) {}),
	).Err()
	if err == nil {
		t.Fatal("expected the start to fail")
	}

	for _, want := range []string{
		`an instance name is required`,
		`unknown configuration variable "CONFXLOADER_HSOT" (did you mean "CONFXLOADER_HOST"?)`,
		`config "confxloader": required variable "CONFXLOADER_HOST" is not set`,
		`config "confxother": variable "CONFXOTHER_PORT"`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("report misses %q:\n%v", want, err)
		}
	}
}

func TestModuleReportDoesNotDependOnRegistrationOrder(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"CONFXOTHER_PORT": "eighty",
	}

	var reports []string
	for _, options := range [][]fx.Option{
		{Module(confmaker.WithEnv(env)), Provide[loaderConfig]("confxloader"), Provide[loaderOtherConfig]("confxother")},
		{fx.Invoke(func(loaderOtherConfig) {}), Provide[loaderOtherConfig]("confxother"), Provide[loaderConfig]("confxloader"), Module(confmaker.WithEnv(env))},
	} {
		err := fx.New(append([]fx.Option{fx.NopLogger}, options...)...).Err()
		if err == nil {
			t.Fatal("expected the start to fail")
		}

		// Fx may wrap the report in context of its own; the report is the tail.
		msg := err.Error()
		reports = append(reports, msg[strings.Index(msg, `config "confxloader"`):])
	}

	if reports[0] != reports[1] {
		t.Fatalf("reports differ:\n%s\n---\n%s", reports[0], reports[1])
	}
}

// Not parallel: it counts SetDefaults calls in a shared counter.
func TestModuleLoadsOnceForEveryConsumer(t *testing.T) {
	loaderDefaultCalls.Store(0)

	var first, second loaderConfig
	err := fx.New(
		fx.NopLogger,
		fx.Invoke(func(cfg loaderConfig) { first = cfg }),
		Provide[loaderConfig]("confxloader"),
		Module(confmaker.WithEnv(map[string]string{"CONFXLOADER_HOST": "db"})),
		fx.Invoke(func(cfg loaderConfig) { second = cfg }),
	).Err()
	if err != nil {
		t.Fatal(err)
	}

	if first.Host != "db" || second.Host != "db" || loaderDefaultCalls.Load() != 1 {
		t.Fatalf("first %+v, second %+v, defaults calls %d", first, second, loaderDefaultCalls.Load())
	}
}

func TestModuleRegisteredTwiceFailsTheStart(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"CONFXLOADER_HOST": "db",
	}

	err := fx.New(fx.NopLogger, Module(confmaker.WithEnv(env)), Module(confmaker.WithEnv(env)), Provide[loaderConfig]("confxloader")).Err()
	if err == nil {
		t.Fatal("a second Module was accepted")
	}
}

func TestProvideNamedTagsInstance(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"CONFXDEFAULT_HOST":          "primary",
		"CONFXREPLICA_POSTGRES_HOST": "replica",
		"CONFXCUSTOM_HOST":           "custom",
	}

	type configs struct {
		fx.In
		Primary namedConfig
		Replica namedConfig `name:"replica"`
		Custom  namedConfig `name:"custom"`
	}

	var got configs
	err := fx.New(
		fx.NopLogger,
		Module(confmaker.WithEnv(env)),
		Provide[namedConfig]("confxdefault"),
		ProvideNamed[namedConfig]("replica", confmaker.WithPrefix("CONFXREPLICA_POSTGRES_")),
		ProvideNamed[namedConfig]("custom", confmaker.WithPrefix("CONFXCUSTOM_")),
		fx.Invoke(func(c configs) { got = c }),
	).Err()
	if err != nil {
		t.Fatal(err)
	}

	if got.Primary.Host != "primary" || got.Replica.Host != "replica" || got.Custom.Host != "custom" {
		t.Fatalf("instances crossed: %+v", got)
	}
}

func TestProvidersRejectInvalidNames(t *testing.T) {
	t.Parallel()

	for providerName, provider := range map[string]func(string, ...confmaker.ConfigOption) fx.Option{
		"Provide":      Provide[namedConfig],
		"ProvideNamed": ProvideNamed[namedConfig],
	} {
		for _, tc := range []struct {
			name         string
			instanceName string
			opts         []confmaker.ConfigOption
			want         string
		}{
			{"empty", "", nil, "an instance name is required"},
			{"invalid", "UPPER", nil, "may hold only lowercase"},
		} {
			t.Run(providerName+"/"+tc.name, func(t *testing.T) {
				err := fx.New(fx.NopLogger, Module(confmaker.WithEnv(nil)), provider(tc.instanceName, tc.opts...)).Err()
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("got %v, want %q", err, tc.want)
				}
			})
		}
	}
}

func TestProvidePrefixOverridePreservesInstanceName(t *testing.T) {
	t.Parallel()

	var got namedConfig
	diagnostics := confmaker.MakeDiagnostics()
	err := fx.New(
		fx.NopLogger,
		Module(confmaker.WithEnv(map[string]string{"CUSTOM_HOST": "db"}), confmaker.WithDiagnostics(diagnostics)),
		Provide[namedConfig]("primary", confmaker.WithPrefix("CUSTOM_")),
		fx.Populate(&got),
	).Err()
	if err != nil || got.Host != "db" {
		t.Fatalf("got %+v, %v", got, err)
	}
	report := diagnostics.Report()
	if len(report.Configs) != 1 || report.Configs[0].InstanceName != "primary" || report.Configs[0].Variables[0].Name != "CUSTOM_HOST" {
		t.Fatalf("instance name or prefix lost: %+v", report)
	}
}

func TestProvideNameAddsNoFxTag(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"CONFXCUSTOM_HOST": "custom",
	}

	var got namedConfig
	err := fx.New(fx.NopLogger, Module(confmaker.WithEnv(env)), Provide[namedConfig]("confxcustom"), fx.Populate(&got)).Err()
	if err != nil || got.Host != "custom" {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestProvideReusedAcrossAppsDoesNotShareValues(t *testing.T) {
	t.Parallel()

	option := Provide[loaderConfig]("confxloader")

	var first loaderConfig
	firstEnv := confmaker.WithEnv(map[string]string{"CONFXLOADER_HOST": "first"})
	if err := fx.New(fx.NopLogger, Module(firstEnv), option, fx.Populate(&first)).Err(); err != nil {
		t.Fatal(err)
	}

	var second loaderConfig
	secondEnv := confmaker.WithEnv(map[string]string{"CONFXLOADER_HOST": "second"})
	if err := fx.New(fx.NopLogger, Module(secondEnv), option, fx.Populate(&second)).Err(); err != nil {
		t.Fatal(err)
	}

	if first.Host != "first" || second.Host != "second" {
		t.Fatalf("apps shared a value: %+v, %+v", first, second)
	}
}

func TestModuleReadsWithEnv(t *testing.T) {
	t.Parallel()

	var got loaderConfig
	err := fx.New(
		fx.NopLogger,
		Module(confmaker.WithEnv(map[string]string{"CONFXLOADER_HOST": "from the map"})),
		Provide[loaderConfig]("confxloader"),
		fx.Populate(&got),
	).Err()
	if err != nil || got.Host != "from the map" {
		t.Fatalf("got %+v, %v", got, err)
	}
}
