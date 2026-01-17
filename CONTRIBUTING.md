# Contributing / Developer documentation

## Prerequisites

This software uses [golangci-lint](https://golangci-lint.run), [pkgsite](https://pkg.go.dev/golang.org/x/pkgsite/cmd/pkgsite),
[changie](https://changie.dev/) and [goreleaser](https://goreleaser.com/).

When using a Dev Container the tools are available by default. On the local
machine they can be installed with:

```shell
make install-tools
```

## Run tools locally

To lint, test and build the code run `make all` or simply `make` as `all` is the
default make target:

```shell
make
```

The single actions also have their own make targets:

```shell
make lint
make test
make build
```

To show the documentation with `pkgsite` `doc-server` can be used:

```shell
make doc-serve
```

It starts a server in the foreground and opens a webbrowser.

## Start with a new change

Call `changie new`:

```shell
changie new
```

This will ask for the kind of change and create a new file in `./changes/unreleased`.

## Create new release

The release process consists of the following steps:

1. Create changelog locally
2. Test goreleaser locally
3. Tag the release locally and trigger goreleaser on Github CI

### Create changelog locally

`changie batch` collects unreleased changes info from `./changes/unreleased` and
creates a new version file like `./changes/v1.2.3.md`.

`changie merge` collects version files from the `./changes` folder and updates `CHANGELOG.md`.

Change `minor`to the type of change:

```shell
changie batch minor
changie merge
```

You may want to call the changie commands with the `--dry-run` to preview the changelog.

Don't forget to commit the changes so that the workspace is clean:

```shell
git add .
git commit -m "Prepare release $(changie latest)"
```

### Test goreleaser locally

```shell
goreleaser release --snapshot --clean --release-notes .changes/$(changie latest).md
```

### Tag the release locally and trigger goreleaser on Github CI

```shell
git tag $(changie latest)
git push origin main && git push --tags
```
