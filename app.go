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
	modules              []fx.Option
	loaderOptions        []confmaker.LoaderOption
	manifestOptions      []confmaker.ManifestOption
	isManifestOptionsSet bool
}

// New captures modules without constructing an Fx app, loading configuration,
// or evaluating defaults. Go argument evaluation and package init still run
// normally. Declare configs at the root and wrap service modules in Runtime.
// MakeModule also supports nested configuration branches with Fx scopes.
func New(modules ...fx.Option) *App {
	return &App{modules: slices.Clone(modules)}
}

// WithOptions returns a clone with additional confmaker loader options. Each Fx
// application receives a fresh loader; supplied engines and callbacks are shared.
func (a *App) WithOptions(opts ...confmaker.LoaderOption) *App {
	clone := *a
	clone.loaderOptions = append(slices.Clone(a.loaderOptions), opts...)

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
// It includes defaults by default, executing user defaults and renderers.
// Use WithManifestOptions() to disable that behavior.
// Additional opts follow the saved options. Custom engines do not support
// manifests; opaque options outside Runtime are rejected before defaults run.
func (a *App) Manifest(opts ...confmaker.ManifestOption) (confmaker.ManifestResult, error) {
	options := slices.Clone(a.manifestOptions)
	if !a.isManifestOptionsSet {
		options = []confmaker.ManifestOption{confmaker.IncludeDefaults()}
	}

	var declarations []func(*confmaker.Loader)

	if err := inspectOptions(
		a.modules,
		"root",
		true,
		&declarations,
	); err != nil {
		return confmaker.ManifestResult{}, err
	}

	loader := confmaker.MakeLoader(a.loaderOptions...)
	for _, register := range declarations {
		register(loader)
	}

	return loader.Manifest(append(options, opts...)...)
}

// FX returns the composition's Fx option without constructing an application
// or parsing command-line arguments. Each fx.New gets a fresh loader. Do not
// combine it with Module or FromLoader. Use fx.New(a.FX(), ...) for manual
// lifecycle control or additional runtime-only options such as fx.Populate.
func (a *App) FX() fx.Option {
	if err := inspectOptions(
		a.modules,
		"root",
		false,
		nil,
	); err != nil {
		return fx.Error(err)
	}

	opts := []fx.Option{Module(a.loaderOptions...)}

	return fx.Options(append(opts, a.modules...)...)
}

// Run handles -help and -describe=env|markdown|json before constructing Fx.
// It reads the current process arguments and writes help/descriptions to stdout.
// Unknown flags, positional arguments, and description/output errors are printed
// to stderr and exit with code 1. Successful help or description returns normally.
// Otherwise Run delegates to fx.New(a.FX()).Run(), including Fx's signal,
// timeout, shutdown and exit-code behavior. Each call creates a fresh Fx app;
// concurrent Run calls are unsupported. Use HandleCLI and FX for custom CLI or
// lifecycle control. Run does not recover user panics.
// Process exits skip caller defers; use HandleCLI and FX when the caller
// must retain control of cleanup and exit decisions.
func (a *App) Run() {
	isHandled, err := a.HandleCLI(os.Args[1:], os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}

	if isHandled {
		return
	}

	fx.New(a.FX()).Run()
}

// HandleCLI handles help and configuration descriptions without constructing Fx
// or invoking configuration loading. Description may call user defaults, which
// can perform I/O. A true result means the CLI request completed;
// false with no error means normal startup should continue. Errors are returned
// without printing or exiting. The caller owns writer; nil uses io.Discard.
func (a *App) HandleCLI(args []string, writer io.Writer) (isHandled bool, err error) {
	if writer == nil {
		writer = io.Discard
	}

	return confcli.Describe(a, args, writer)
}
