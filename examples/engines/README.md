# Configuration engines with Fx

These runnable application examples use the same pattern: prepare sources, make
an Engine, declare a domain with MakeModule, then configure MakeModules once.
ProvideNamed injects primary and replica configs into makeService through fx.In.
No app.Start is needed to inspect loaded values; unrelated constructors can still
run during fx.New.

| Directory | Backend | Native tags | Sources |
| --- | --- | --- | --- |
| env11 | caarlos0/env/v11 | env, envDefault | Complete maps per registration |
| koanf | koanf with TOML parser | koanf | Embedded TOML files plus an overlay |

```sh
GOWORK=off go run ./env11
GOWORK=off go run ./koanf
GOWORK=off go mod tidy -diff
GOWORK=off go build ./...
GOWORK=off go vet ./...
GOWORK=off go test -race ./...
```

The examples pin dependencies in this module only. The relative replacement of
confx deliberately exercises this checkout; confmaker uses its published version.
Root go test ./... skips this module, so CI checks it separately.

Adapters snapshot caller input and create a decoder for each request. Registration
names select sources; they are not automatic ENV prefixes or file paths. env/v11
receives non-nil complete maps so process ENV cannot fill missing keys. Its native
envDefault supplies the primary port. koanf merges its overlay after the base
file and uses its normal decoding and unknown-key behavior, not strict validation
of arbitrary file keys. File reads or network retrieval belong to the application;
embedded fixtures make these examples reproducible from any working directory.

Root SetDefaults supplies the label before the engine runs. These tests demonstrate
preservation for the shown inputs, not universal merging semantics for every type.
Validate rejects nonpositive ports. On a load error Fx does not construct the
consumer service. Reports have config-level detail; custom engines cannot generate
Manifest. Backend errors are passed through and may contain input values.
Cancellation is checked before decoding, not forcibly enforced during decoding.

These are application-local adapters, not supported adapter packages. They do not
add watchers, reload, engine lifecycle management or promises about secret codecs.
