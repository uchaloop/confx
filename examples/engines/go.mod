module github.com/uchaloop/confx/examples/engines

go 1.27.0

require (
	github.com/caarlos0/env/v11 v11.4.1
	github.com/knadh/koanf/parsers/toml/v2 v2.2.2
	github.com/knadh/koanf/providers/rawbytes v1.0.1
	github.com/knadh/koanf/v2 v2.3.8
	github.com/uchaloop/confmaker/v2 v2.0.0
	github.com/uchaloop/confx v0.5.0
	go.uber.org/fx v1.24.0
)

require (
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/knadh/koanf/maps v0.1.2 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.4.3 // indirect
	github.com/uchaloop/utilfx v0.2.0 // indirect
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
)

// Exercise this checkout without a workspace.
replace github.com/uchaloop/confx => ../..
