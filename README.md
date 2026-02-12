# gh-semver

| ⚠️ Project archived |
|---------------------|
| Feel free to fork   |

This GitHub CLI extension can be used determine the [semantic version] to
release.

First it will search all tags and traverse up and down the git log to find the
latest tag and inspect all commits made since against the [conventional commits]
standard v1.0.0.

## Prerequisites

* [GitHub commandline interface]
* **Repository cloned with full depth**, a shallow clone cannot be traversed.

## Usage (commandline)

Install the extension by running:

```bash
gh extension install koozz/gh-semver
```

Run this extension with its keyword (default is semver):

```bash
gh semver
```

View more options with:

```bash
gh semver -help
```

In case of a newer version, upgrade by running:

```bash
gh extension upgrade koozz/gh-semver
```

## Usage (GitHub Actions)

This extension can be used in a [GitHub Actions] workflow to determine the next
semantic version.

In your workflow;

* make sure the checkout has `fetch-depth: 0`
* install the extension
* call the extension
* use the version as you see fit

```yaml
# ...

    steps:
      - name: Checkout repository
        uses: actions/checkout@v3
        with:
          fetch-depth: 0
      - name: SemVer
        id: semver
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          gh extension install koozz/gh-semver
          gh semver -action
      - name: Tag
        run: |
          git tag ${{ steps.semver.outputs.version }}

# ...
```

Or let the extension create the tag:

```yaml
# ...

    steps:
      - name: Checkout repository
        uses: actions/checkout@v3
        with:
          fetch-depth: 0
      - name: SemVer
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          gh extension install koozz/gh-semver
          gh semver -tag
          git push --tags

# ...
```

Example can be found in [.github/workflows/auto-tag-main.yml][workflow]

## Roadmap

Things on the roadmap:

* Semantic Versioning(-ish) for modules in a mono-repo;
  * Limit versioning to a directory within the repository.
  * Prefix the version with a module name.
* Tests to make the software more robust and reliable.

## Issues?

Help me out and describe the [issue] as complete a possible.

## License

Apache 2.0

<!-- Markdown links -->
[conventional commits]: https://www.conventionalcommits.org/en/v1.0.0/
[GitHub Actions]: https://docs.github.com/en/actions
[GitHub commandline interface]: https://cli.github.com/
[issue]: https://github.com/koozz/gh-semver/issues/new/choose
[semantic version]: https://semver.org/
[workflow]: .github/workflows/auto-tag-main.yml
