# Contributing

Keep changes focused on the thin Fx adapter. Configuration loading rules and
engine contracts belong in confmaker. Update examples and GoDoc when public
behavior changes, and describe the change and its verification in the pull request.

## Branches and commits

Use descriptive branches such as `feat/custom-engine`, `fix/registration` or
`docs/examples`. Start commit subjects with the full source branch name, a colon
and a short English description. Use `release/x.y.z` only for release preparation.
Confx has independent v0.x versioning; core versions do not determine its tag.

## Verification

Run `go test -race ./...` and `go vet ./...`. A local workspace may connect
unpublished sibling libraries; do not commit workspace files or local replacements.
Before release, verify with `GOWORK=off` against the published dependencies.

## Changelog style

Use the same format in confmaker, confx and secret:

- Keep newest versions first. Use `## [x.y.z] - Unreleased` once the target version
  is known, or `## [Unreleased]` before choosing it. Do not add a separate target
  release paragraph. Version headings omit `v`; Git tags include it.
- At publication, replace `Unreleased` with the actual release date in
  `YYYY-MM-DD` format. Do not infer publication dates from commit dates. Preserve
  undated historical entries when the release date cannot be verified.
- Group entries under `Added`, `Changed`, `Deprecated`, `Removed`, `Fixed`,
  `Security`, in that order. Omit empty categories, even for small releases.
- Write in English, using past-tense opening verbs such as `Added`, `Changed`,
  `Removed` and `Fixed`. Each bullet describes one user-visible change, normally
  in one or two sentences. Include internal work only when it explains a useful
  result. Preserve necessary historical compatibility and retraction notes.
- Prefix incompatible changes with `**Breaking:**` in their normal category and
  explain the replacement or required action when applicable. Do not duplicate
  them in a separate breaking-changes section.
- Define heading links at the bottom of the file. An unreleased target compares
  the last released tag with `HEAD`; a released version compares its predecessor
  with its tag. Link the first release to its release page. Preserve explicit
  historical exceptions for retracted or incorrectly tagged versions.
- Separate headings, paragraphs and lists with one blank line. Wrap continuation
  lines consistently and format API names as inline code.

Example before publication:

```markdown
## [2.0.0] - Unreleased

### Added

- Added support for application-provided configuration engines.

### Changed

- **Breaking:** Renamed `EnvOption` to `LoaderOption`; update option declarations.

[2.0.0]: https://github.com/uchaloop/confmaker/compare/v1.0.1...HEAD
```

## Secret integration tests

The main module does not depend on secret. Real secret compatibility tests and
its benchmark live in [integration/secret](integration/secret/README.md), a separate
Go module. Run it explicitly; the root `go test ./...` does not include it.
The integration module intentionally replaces confx with this checkout.
