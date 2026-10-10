# Release process

## MVP v0.1.0

The v0.1.0 baseline is documented in [CHANGELOG.md](../CHANGELOG.md).

Before tagging, verify the latest `main` GitHub Actions CI is green, including Go race/coverage tests and the WASM + PTY E2E jobs. The tag must point to that reviewed commit.

To publish a Git tag from a local checkout:

```bash
git fetch origin
git switch main
git pull --ff-only origin main
git tag -a v0.1.0 -m "Support Shell MVP v0.1.0"
git push origin v0.1.0
```

Then open GitHub → Releases → Draft a new release, choose `v0.1.0`, and use the corresponding changelog section as the release notes.

**Do not claim downloadable Linux/macOS archives are available until their builds have been tested.** The current CI checks Linux Go and Extism/WASM, but not a complete macOS/Linux amd64/arm64 release matrix. This initial baseline may be published as a source-only release. A separate signed/verified binary release workflow can be added later.

Do not retag or force-push `v0.1.0` once published; subsequent corrections should be released as `v0.1.1`.
