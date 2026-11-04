package confx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/uchaloop/confmaker/v2"
	"go.uber.org/fx"
)

type appFailWriter struct{ err error }

func (w appFailWriter) Write([]byte) (int, error) { return 0, w.err }

func TestAppDescribeWriterError(t *testing.T) {
	want := errors.New("output unavailable")
	for _, args := range [][]string{{"-help"}, {"-describe=json"}} {
		isHandled, err := New().isDescribeHandled(args, appFailWriter{err: want})
		if isHandled || !errors.Is(err, want) {
			t.Fatalf("handled=%v err=%v", isHandled, err)
		}
	}
}

func TestAppRunProcess(t *testing.T) {
	for _, test := range []struct {
		name       string
		args       []string
		code       int
		stdout     string
		stderr     string
		runtime    int
		isReturned bool
	}{
		{
			name:       "help",
			args:       []string{"-help"},
			stdout:     "-describe",
			isReturned: true,
		},
		{
			name:       "short-help",
			args:       []string{"-h"},
			stdout:     "-describe",
			isReturned: true,
		},
		{
			name:       "env",
			args:       []string{"-describe=env"},
			stdout:     "SERVER_PORT=8080",
			isReturned: true,
		},
		{
			name:       "json",
			args:       []string{"-describe=json"},
			stdout:     "SERVER_PORT",
			isReturned: true,
		},
		{
			name:       "markdown",
			args:       []string{"-describe", "markdown"},
			stdout:     "SERVER_PORT",
			isReturned: true,
		},
		{
			name:       "no-defaults",
			args:       []string{"-describe=env"},
			stdout:     "SERVER_PORT",
			isReturned: true,
		},
		{
			name:   "bad-format",
			args:   []string{"-describe=xml"},
			code:   1,
			stderr: "unknown description format",
		},
		{
			name:   "unknown",
			args:   []string{"-unknown"},
			code:   1,
			stderr: "flag provided but not defined",
		},
		{
			name:   "positional",
			args:   []string{"extra"},
			code:   1,
			stderr: "unexpected argument",
		},
		{
			name:   "opaque",
			args:   []string{"-describe=json"},
			code:   1,
			stderr: "opaque option",
		},
		{
			name:   "engine-describe",
			args:   []string{"-describe=json"},
			code:   1,
			stderr: "manifest requires the built-in engine",
		},
		{
			name:       "engine-help",
			args:       []string{"-help"},
			stdout:     "-describe",
			isReturned: true,
		},
		{name: "runtime", runtime: 1, isReturned: true},
		{name: "twice", runtime: 2, isReturned: true},
		{name: "start-failure", code: 1, runtime: 1},
		{name: "load-failure", code: 1},
		{name: "shutdown-code", code: 7, runtime: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			args := append([]string{"-test.run=^TestAppRunHelper$", "--"}, test.args...)
			cmd := exec.CommandContext(ctx, os.Args[0], args...)
			cmd.Env = append(os.Environ(), "CONFX_APP_TEST_HELPER="+test.name)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err := cmd.Run()
			if ctx.Err() != nil {
				t.Fatalf("child timed out: stdout=%s stderr=%s", &stdout, &stderr)
			}
			code := 0
			if err != nil {
				var exit *exec.ExitError
				if !errors.As(err, &exit) {
					t.Fatal(err)
				}
				code = exit.ExitCode()
			}
			if code != test.code || !strings.Contains(stdout.String(), test.stdout) || !strings.Contains(stderr.String(), test.stderr) {
				t.Fatalf(
					"exit=%d want=%d stdout=%s stderr=%s",
					code,
					test.code,
					&stdout,
					&stderr,
				)
			}
			if strings.Count(stderr.String(), "runtime-marker\n") != test.runtime {
				t.Fatalf("unexpected runtime construction: %s", &stderr)
			}
			if strings.Contains(stdout.String(), "returned-marker") != test.isReturned {
				t.Fatalf("Run return contract: stdout=%s", &stdout)
			}
			if test.runtime == 0 && test.code == 0 && stderr.Len() != 0 {
				t.Fatalf("CLI-only path produced stderr: %s", &stderr)
			}
			if test.code != 0 && stdout.Len() != 0 {
				t.Fatalf("error output reached stdout: %s", &stdout)
			}
			if test.name == "no-defaults" && strings.Contains(stdout.String(), "8080") {
				t.Fatalf("disabled defaults evaluated: %s", &stdout)
			}
		})
	}
}

// The helper changes process globals only in its own subprocess.
func TestAppRunHelper(t *testing.T) {
	scenario := os.Getenv("CONFX_APP_TEST_HELPER")
	if len(scenario) == 0 {
		return
	}
	separator := slices.Index(os.Args, "--")
	if separator < 0 {
		t.Fatal("missing helper argument separator")
	}
	os.Args = append([]string{os.Args[0]}, os.Args[separator+1:]...)

	modules := []fx.Option{
		Provide[defaultConfig]("server"),
		Runtime(fx.NopLogger, fx.Invoke(func(_ defaultConfig, lc fx.Lifecycle, shutdown fx.Shutdowner) {
			fmt.Fprintln(os.Stderr, "runtime-marker")
			lc.Append(fx.Hook{OnStart: func(context.Context) error {
				if scenario == "start-failure" {
					return errors.New("start failed")
				}
				code := 0
				if scenario == "shutdown-code" {
					code = 7
				}

				return shutdown.Shutdown(fx.ExitCode(code))
			}})
		})),
	}
	if scenario == "opaque" {
		modules = append(modules, fx.Options(fx.NopLogger))
	}
	app := New(modules...)
	if strings.HasPrefix(scenario, "engine-") {
		app = app.WithOptions(confmaker.WithEngine(confmaker.EngineFunc(func(context.Context, confmaker.LoadRequest) error {
			fmt.Fprintln(os.Stderr, "engine-marker")
			return errors.New("engine must not run in help or describe")
		})))
	} else {
		value := "invalid"
		if slices.Contains([]string{"runtime", "twice", "start-failure", "shutdown-code"}, scenario) {
			value = "9090"
		}
		app = app.WithOptions(confmaker.WithEnv(map[string]string{"SERVER_PORT": value}))
	}
	if scenario == "no-defaults" {
		app = app.WithManifestOptions()
	}

	app.Run()
	if scenario == "twice" {
		app.Run()
	}
	fmt.Fprintln(os.Stdout, "returned-marker")
	os.Exit(0)
}
