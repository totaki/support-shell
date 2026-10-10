# Coverage reports

Go tests run in GitHub Actions with race detection and produce `coverage.out`. After successful tests on `main`, `codecov/codecov-action@v5` uploads that file to [Codecov](https://codecov.io/gh/totaki/support-shell) via GitHub Actions OIDC (`id-token: write` on the Go job). No GitHub `CODECOV_TOKEN` secret is required for this method.

## One-time setup

1. Sign in to [Codecov](https://codecov.io) using GitHub and authorize access to the public repository `totaki/support-shell`.
2. Enable the repository in Codecov if it is not enabled already.
3. Trigger the `CI` workflow on `main` and inspect the **Upload Go coverage to Codecov** step. If authentication or upload fails, check the repository's Codecov configuration and OIDC support.
4. Open the Codecov page to see history and individual file coverage. The badge in `README.md` and `README.ru.md` resolves after Codecov receives coverage for the repository.

Uploads are limited to successful non-PR runs, so forked pull requests do not require upload credentials. The upload is currently **non-blocking** (`fail_ci_if_error: false`), so problems with an external service do not make passing tests red. GitHub Actions still publishes a Go test log and `coverage.out` as build artifacts.

Coverage percentages measure **Go statements exercised by unit and integration tests**. Rust SDK tests and the compiled WASM-to-MCP E2E flow are executed in other CI steps; they are not included in this Go coverage number.
