package confx

import (
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/uchaloop/confmaker/v2"
	"github.com/uchaloop/confmaker/v2/confcli"
	"go.uber.org/fx"
)

// App is an immutable application composition with configuration description
// and process-level startup. Its zero value is equivalent to New().
// Use Manifest and FX for custom command-line handling or a manual Fx lifecycle.
type App struct {
	modules              Modules
	manifestOptions      []confmaker.ManifestOption
	isManifestOptionsSet bool
}

// New captures modules without constructing an Fx app, loading configuration,
// or evaluating defaults. Go argument evaluation and package init still run
// normally. Use MakeModule for configuration branches and Runtime for ordinary
// Fx options that offline description should skip.
func New(modules ...fx.Option) *App {
	return &App{modules: *MakeModules(modules...)}
}

// WithOptions returns a clone with additional confmaker loader options. Each Fx
// application receives a fresh loader; supplied engines and callbacks are shared.
func (a *App) WithOptions(opts ...confmaker.LoaderOption) *App {
	clone := *a
	clone.modules = *a.modules.WithOptions(opts...)

	return &clone
}

// WithManifestOptions returns a clone with replacement description options.
// App includes defaults unless this method is called. Calling it with no
// arguments disables default evaluation; it does not affect runtime loading.
func (a *App) WithManifestOptions(opts ...confmaker.ManifestOption) *App {
	clone := *a
	clone.manifestOptions = slices.Clone(opts)
	clone.isManifestOptionsSet = true

	return &clone
}

// Manifest describes the composition without constructing Fx or loading ENV.
// Unlike Modules.Manifest, it includes defaults by default, executing user
// defaults and renderers. Use WithManifestOptions() to disable that behavior.
// Additional opts follow the saved options. Custom engines do not support
// manifests; opaque options outside Runtime are rejected before defaults run.
func (a *App) Manifest(opts ...confmaker.ManifestOption) (confmaker.ManifestResult, error) {
	options := slices.Clone(a.manifestOptions)
	if !a.isManifestOptionsSet {
		options = []confmaker.ManifestOption{confmaker.IncludeDefaults()}
	}

	return a.modules.Manifest(append(options, opts...)...)
}

// FX returns the composition's Fx option without constructing an application
// or parsing command-line arguments. Each fx.New gets a fresh loader. Do not
// combine it with Module or FromLoader. Use fx.New(a.FX(), ...) for manual
// lifecycle control or additional runtime-only options such as fx.Populate.
func (a *App) FX() fx.Option {
	return a.modules.FX()
}

// Run handles -help and -describe=env|markdown|json before constructing Fx.
// It reads the current process arguments and writes help/descriptions to stdout.
// Unknown flags, positional arguments, and description/output errors are printed
// to stderr and exit with code 1. Successful help or description returns normally.
// Otherwise Run delegates to fx.New(a.FX()).Run(), including Fx's signal,
// timeout, shutdown and exit-code behavior. Each call creates a fresh Fx app;
// concurrent Run calls are unsupported. Use Manifest and FX for custom CLI or
// lifecycle control. Run does not recover user panics.
// Run is the explicitly agreed process-entry exception to the library no-exit
// rule. Process exits skip caller defers; use Manifest and FX when the caller
// must retain control of cleanup and exit decisions.
func (a *App) Run() {
	isHandled, err := a.isDescribeHandled(os.Args[1:], os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}

	if isHandled {
		return
	}

	fx.New(a.FX()).Run()
}

func (a *App) isDescribeHandled(args []string, writer io.Writer) (bool, error) {
	return confcli.Describe(a, args, writer)
}
