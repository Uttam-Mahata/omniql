# OmniQL Release Strategy

OmniQL publishes two release tiers. Publishing is intentional and
tag-driven — there are no automatic publishes on every commit.

---

## Release tiers

| Tier | Trigger | Frequency | Purpose |
|------|---------|-----------|---------|
| **Stable** | `git tag v0.9.0` | On demand | Production-ready |
| **Release Candidate** | `git tag v0.9.0-rc.1` | On demand | Final validation before stable |

---

## Tag conventions

```
v<major>.<minor>.<patch>           → stable   (e.g. v0.9.0)
v<major>.<minor>.<patch>-rc.<n>    → rc       (e.g. v0.9.0-rc.1)
```

## Cutting a release

```bash
# 1. Update the base version in bindings/python/pyproject.toml
#    This version is used as the base for the published version.

# 2. Push the appropriate tag
git tag v0.9.0-rc.1   && git push origin v0.9.0-rc.1
git tag v0.9.0        && git push origin v0.9.0
```

The publish workflow detects the tag suffix and sets the correct version
string and distribution channel for every ecosystem automatically.

> **Note — tag version vs. pyproject.toml version**
>
> For all tagged releases (stable, rc) the published version is taken
> directly from the git tag, **not** from `bindings/python/pyproject.toml`.
> The workflow overwrites `pyproject.toml` at build time with the tag version.
> This means a tag like `v1.0.1` will publish `1.0.1` across all ecosystems
> even if `pyproject.toml` still reads `0.8.1`.

---

## Version matrix per ecosystem

| Tier | Python / NuGet | npm version | npm dist-tag | Maven |
|------|----------------|-------------|--------------|-------|
| stable | `0.9.0` | `0.9.0` | `latest` | `0.9.0` |
| rc | `0.9.0-rc.1` | `0.9.0-rc.1` | `next` | `0.9.0-rc.1` |

---

## Installing a specific tier

### Python (PyPI)

```bash
pip install omniql                              # stable
pip install omniql==0.9.0-rc.1                 # rc
```

### npm

```bash
npm install omniql            # stable  (latest tag)
npm install omniql@next       # latest rc
```

### NuGet (.NET)

```bash
dotnet add package OmniQL                              # stable
dotnet add package OmniQL --version 0.9.0-rc.1        # rc
```

### Maven (Java — GitHub Packages)

Add the GitHub Packages repository to your `~/.m2/settings.xml`:

```xml
<settings>
  <servers>
    <server>
      <id>github-omniql</id>
      <username>YOUR_GITHUB_USERNAME</username>
      <password>YOUR_PAT</password>   <!-- needs read:packages scope -->
    </server>
  </servers>
</settings>
```

Then in `pom.xml`:

```xml
<!-- dependency -->
<dependency>
  <groupId>io.github.uttam-mahata</groupId>
  <artifactId>omniql</artifactId>
  <version>0.9.0</version>          <!-- or 0.9.0-rc.1 -->
</dependency>
```

---

## Workflow internals

The publish workflow (`.github/workflows/publish.yml`) runs as follows:

```
get-version ──┬──► build-core (linux / macos / windows in parallel)
              │         │
              │         ├──► publish-pypi
              │         ├──► publish-npm
              │         ├──► publish-nuget
              │         └──► publish-maven
              │
              └──► goreleaser   (tags only — CLI binaries, deb, rpm, Homebrew, Scoop)
```

1. **`get-version`** — reads the base version from `bindings/python/pyproject.toml`, inspects the tag trigger, and outputs `version`, `maven_version`, `npm_tag`, and `release_type`.
2. **`build-core`** — compiles `libomniql.so` / `libomniql.dylib` / `omniql.dll` using Go with CGO.
3. **Publish jobs** — each binding downloads the three native binaries, stamps the version, and publishes to its registry.
4. **`goreleaser`** *(added v0.6.0)* — builds the `omniql` CLI binary for all platforms, creates archives, `.deb`/`.rpm` packages, and publishes to GitHub Releases / Homebrew / Scoop. Runs on tag pushes only.

---

## Binary Distribution (v0.6.0+)

Starting with v0.6.0, pre-built `omniql` CLI binaries are published via GoReleaser
alongside the language-binding packages.

### Platforms & formats

| Platform | amd64 | arm64 | Formats |
|----------|-------|-------|---------|
| Linux | ✅ | ✅ | `.tar.gz`, `.deb`, `.rpm` |
| macOS | ✅ | ✅ | `.tar.gz`, Homebrew tap |
| Windows | ✅ | — | `.zip`, Scoop bucket |

### GitHub Actions secrets required

| Secret | Used for |
|--------|----------|
| `GITHUB_TOKEN` | GitHub Release uploads (provided automatically) |
| `HOMEBREW_TAP_TOKEN` | Push formula to `Uttam-Mahata/homebrew-omniql` (PAT, `repo` scope) |
| `SCOOP_BUCKET_TOKEN` | Push manifest to `Uttam-Mahata/scoop-omniql` (PAT, `repo` scope) |

### Local dry-run

```bash
# Produces artifacts in ./dist without touching GitHub
goreleaser release --snapshot --clean
```
