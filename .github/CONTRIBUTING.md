# Contributing to the Airplane CLI

## Releases

Releases are managed by [GoReleaser](https://github.com/goreleaser/goreleaser). This produces binaries for various architectures and uploads them as GitHub artifacts. It also releases to Homebrew through [`airplanedev/homebrew-tap`](https://github.com/airplanedev/homebrew-tap).

This all happens automatically via GitHub Actions whenever a new tag is published:

```sh
git tag v0.0.1
git push origin v0.0.1
```

You can test this build process locally by running:

```sh
# or https://goreleaser.com/install/
brew install goreleaser/tap/goreleaser

goreleaser --snapshot --skip-publish --rm-dist
```
