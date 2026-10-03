# Changelog

## [Unreleased]

## [0.4.1] - 2026-10-03

- Updated confmaker to v1.0.1 and secret/v2 to v2.1.0; adapter APIs are unchanged.
- **Compatibility:** `Secret` is no longer comparable; JSON `null` clears it.
- Set the minimum Go version to 1.27.0 and updated documentation.

## [0.4.0] - 2026-10-03

- Updated confmaker to v1.0.0; confx remains on the independent v0.x release line.
- **Breaking:** the core no longer provides `WithDump`. Use manifest for
  declarations and diagnostics for value-free load reports.
- **Breaking:** core JSON fields accept collections and scalar text types,
  rejecting ordinary structs at every depth. Text methods take precedence over
  JSON methods; defaults without `MarshalText` fail manifest rendering.
- Updated usage and manifest examples for the v1 core. Fx registration APIs
  remain unchanged.

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

[Unreleased]: https://github.com/uchaloop/confx/compare/v0.4.1...HEAD
[0.4.1]: https://github.com/uchaloop/confx/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/uchaloop/confx/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/uchaloop/confx/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/uchaloop/confx/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/uchaloop/confx/releases/tag/v0.1.0
