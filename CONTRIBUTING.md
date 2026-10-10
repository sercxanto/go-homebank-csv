# Contributing / Developer documentation

## Prerequisites

This software uses [Task](https://taskfile.dev) as task runner,
[golangci-lint](https://golangci-lint.run), [pkgsite](https://pkg.go.dev/golang.org/x/pkgsite/cmd/pkgsite),
[changie](https://changie.dev/) and [goreleaser](https://goreleaser.com/).

When using a Dev Container the tools are available by default. On the local
machine install Task first, either with a package manager (see the
[installation guide](https://taskfile.dev/installation/)) or with Go:

```shell
GOTOOLCHAIN=auto go install github.com/go-task/task/v3/cmd/task@v3.54.0
```

Then install the other tools with:

```shell
task install-tools
```

## Run tools locally

To lint, test and build the code run `task`, which runs the `default` task:

```shell
task
```

The single actions also have their own tasks:

```shell
task lint
task test
task build
```

`task --list` shows all tasks with a short description.

To show the documentation with `pkgsite` `doc-serve` can be used:

```shell
task doc-serve
```

It starts a server in the foreground and opens a webbrowser.

## Go version

The `go` directive in `go.mod` is the minimum Go version required to build
the code. The GitHub workflows in `.github/workflows` install the latest patch
release of the same minor version with `go-version` and `check-latest`, so that
CI and releases always get the security fixes of the standard library. When
the minor version in `go.mod` changes, update `go-version` in all workflows as
well.

## GitHub Actions

The workflows reference every action by the full commit SHA of a release,
followed by the version as a comment:

```yaml
- uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
```

Unlike a tag, a commit SHA cannot be moved to different code, so a
compromised action repository cannot change what runs in this repository's
workflows. Dependabot updates the SHA and the comment together. Reference new
actions the same way; `git ls-remote --tags https://github.com/<owner>/<repo>`
lists the commit of each tag (for annotated tags the line ending in `^{}`).

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
