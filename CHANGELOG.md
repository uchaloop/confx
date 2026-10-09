# Changelog

All notable changes to this module are documented in this file.

Entries are grouped by version and change type, with the newest version first.

## [0.5.0] - Unreleased

### Added

- Added support for application-supplied engines through core options in both
  registration modes, without new dependencies or an adapter-specific engine API.

### Changed

- **Breaking:** Renamed the Module option type from `confmaker.EnvOption` to
  `confmaker.LoaderOption`.
- **Breaking:** Adopted `github.com/uchaloop/confmaker/v2`; core options, loaders and
  handles from v1 are no longer compatible. Fx registration function names remain
  unchanged, and confx retains independent versioning.
- Updated examples to use loader-owned reports with external loaders and diagnostic
  callbacks with internally owned loaders. Each Fx application retains its own loader.
- Updated offline documentation examples to use explicit manifest defaults and the
  separate `confcli` and `confexport` packages.

## [0.4.1] - 2026-10-03

### Changed

- Updated confmaker to v1.0.1 and secret/v2 to v2.1.0; adapter APIs are unchanged.
- **Breaking:** Adopted a non-comparable `Secret`; JSON `null` now clears it.
- Set the minimum Go version to 1.27.0 and updated documentation.

## [0.4.0] - 2026-10-03

### Changed

- Updated confmaker to v1.0.0; confx remains on the independent v0.x release line.
- **Breaking:** Changed core JSON fields to accept collections and scalar text types,
  rejecting ordinary structs at every depth. Text methods take precedence over JSON
  methods; defaults without `MarshalText` fail manifest rendering.
- Updated usage and manifest examples for the v1 core. Fx registration APIs remain
  unchanged.

### Removed

- **Breaking:** Removed the core `WithDump` API. Use manifest for declarations and
  diagnostics for value-free load reports.

## [0.3.0] - 2026-10-01

### Added

- Added `FromLoader`, `FromHandle` and `FromHandleNamed` for external registrations and
  manifest workflows; rejected mixing with ordinary `Provide` registrations.

### Changed

- Documented and tested core diagnostics in both modes, including handlers for
  `fx.New(...).Run()`.

## [0.2.0] - 2026-10-01

### Changed

- **Breaking:** Required an explicit instance name in `Provide[T](name, opts...)`,
  matching `ProvideNamed` and the confmaker v0.8.0 API. Only `ProvideNamed` adds an Fx
  name tag.
- Updated the confmaker dependency to v0.8.0.
- Changed registration to use `Loader.Register` and delegated names and loading to
  confmaker. Updated examples and documented loading during `fx.New`.

## [0.1.0] - 2026-09-20

### Added

- Published the initial standalone release of the confx Uber Fx adapter.
- Established an independent module, dependencies, tests and release workflow.

[0.5.0]: https://github.com/uchaloop/confx/compare/v0.4.1...HEAD
[0.4.1]: https://github.com/uchaloop/confx/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/uchaloop/confx/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/uchaloop/confx/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/uchaloop/confx/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/uchaloop/confx/releases/tag/v0.1.0
