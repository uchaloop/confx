package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/uchaloop/confmaker/v2"
	"github.com/uchaloop/confx"
	"go.uber.org/fx"
)

func TestAppDescribeSecret(t *testing.T) {
	for _, format := range []string{"env", "markdown", "json"} {
		t.Run(format, func(t *testing.T) {
			t.Setenv("CONFX_SECRET_DESCRIBE_HELPER", format)
			t.Setenv("APP_TOKEN", "private-input")
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAppDescribeSecretHelper$")
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("description failed: %v", err)
			}

			if strings.Contains(stdout.String(), "private-") || strings.Contains(stderr.String(), "private-") {
				t.Fatal("description disclosed a secret")
			}
			if stderr.Len() != 0 {
				t.Fatal("description wrote to stderr")
			}
			if !strings.Contains(stdout.String(), "APP_TOKEN") {
				t.Fatal("description omitted the secret variable")
			}
			if format != "json" {
				return
			}

			var result struct {
				Configs []struct {
					Variables []struct {
						Name     string `json:"name"`
						IsSecret bool   `json:"secret"`
						Default  struct {
							State string `json:"state"`
						} `json:"default"`
					} `json:"variables"`
				} `json:"configs"`
			}
			if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
				t.Fatalf("invalid JSON description: %v", err)
			}
			if len(result.Configs) != 1 || len(result.Configs[0].Variables) != 1 {
				t.Fatal("unexpected description shape")
			}

			variable := result.Configs[0].Variables[0]
			if variable.Name != "APP_TOKEN" || !variable.IsSecret || variable.Default.State != "redacted" {
				t.Fatal("secret default was not marked redacted")
			}
		})
	}
}

// Process globals are changed only in the helper subprocess.
func TestAppDescribeSecretHelper(t *testing.T) {
	format := os.Getenv("CONFX_SECRET_DESCRIBE_HELPER")
	if len(format) == 0 {
		return
	}

	os.Args = []string{os.Args[0], "-describe=" + format}
	confx.New(
		confx.MakeModule("app",
			confx.Provide[config]("app"),
			confx.Runtime(fx.Invoke(func(config) {
				t.Fatal("description constructed the runtime")
			})),
		),
	).WithOptions(confmaker.WithDiagnosticHandler(func(confmaker.LoadReport) {
		t.Fatal("description loaded configuration")
	})).Run()

	// Prevent the Go test runner from appending PASS to the exported document.
	os.Exit(0)
}
