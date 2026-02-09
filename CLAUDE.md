# Claude Code Rules

## Release Process
- **Always update VERSION and CHANGELOG.md** with every commit that changes functionality (features, fixes, etc.).
- Version file: `VERSION` (single semver string)
- Changelog: `CHANGELOG.md` (Keep a Changelog format)
- Bump patch for fixes, minor for features, major for breaking changes.
- The version is embedded at build time via ldflags from the `VERSION` file.
