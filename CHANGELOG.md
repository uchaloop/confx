# Changelog

## [Unreleased]

## [0.3.0] - 2026-10-01

- Added `FromLoader`, `FromHandle` and `FromHandleNamed` for external registrations
  and manifest workflows; reject mixing with ordinary `Provide` registrations.
- Documented and tested core diagnostics in both modes, including handlers for
  `fx.New(...).Run()`.

## [0.2.0] - 2026-10-01

- **Breaking:** `Provide[T](name, opts...)` requires an explicit instance name,
  matching `ProvideNamed` and the confmaker v0.8.0 API. Only `ProvideNamed` adds
  an Fx name tag.
- Updated the confmaker dependency to v0.8.0.
- Register configs through `Loader.Register`; delegate names and loading to
  confmaker. Updated examples and documented loading during `fx.New`.

## [0.1.0] - 2026-09-20

- Initial standalone release of the confx Uber Fx adapter.
- Independent module, dependencies, tests and release workflow.

[Unreleased]: https://github.com/uchaloop/confx/compare/v0.3.0...HEAD
[0.2.0]: https://github.com/uchaloop/confx/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/uchaloop/confx/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/uchaloop/confx/releases/tag/v0.1.0
