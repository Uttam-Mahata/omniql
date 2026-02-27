# OmniQL Release Strategy

OmniQL publishes four release tiers. Publishing is intentional and
tag-driven — there are no automatic publishes on every commit.

---

## Release tiers

| Tier | Trigger | Frequency | Purpose |
|------|---------|-----------|---------|
| **Stable** | `git tag v0.9.0` | On demand | Production-ready |
| **Release Candidate** | `git tag v0.9.0-rc.1` | On demand | Final validation before stable |
| **Beta** | `git tag v0.9.0-beta.1` | On demand | Feature-complete pre-release for early adopters |
| **Nightly** | Scheduled (02:00 UTC) | Daily | Latest dev snapshot |

---

## Tag conventions

```
v<major>.<minor>.<patch>           → stable   (e.g. v0.9.0)
v<major>.<minor>.<patch>-rc.<n>    → rc       (e.g. v0.9.0-rc.1)
v<major>.<minor>.<patch>-beta.<n>  → beta     (e.g. v0.9.0-beta.1)
(no tag — cron schedule)           → nightly
```

## Cutting a release

```bash
# 1. (Optional) Update the base version in bindings/python/pyproject.toml
#    Only needed for nightly SNAPSHOT builds — tagged releases use the tag
#    version directly and override pyproject.toml at publish time.

# 2. Push the appropriate tag
git tag v0.9.0-beta.1 && git push origin v0.9.0-beta.1
git tag v0.9.0-rc.1   && git push origin v0.9.0-rc.1
git tag v0.9.0        && git push origin v0.9.0
```

The publish workflow detects the tag suffix and sets the correct version
string and distribution channel for every ecosystem automatically.

> **Note — tag version vs. pyproject.toml version**
>
> For all tagged releases (stable, rc, beta) the published version is taken
> directly from the git tag, **not** from `bindings/python/pyproject.toml`.
> The workflow overwrites `pyproject.toml` at build time with the tag version.
> This means a tag like `v1.0.1` will publish `1.0.1` across all ecosystems
> even if `pyproject.toml` still reads `0.8.1`.
>
> `pyproject.toml` only influences the nightly build, where its value is used
> as the base for the `-nightly.YYYYMMDD` / `-SNAPSHOT` suffix.

---

## Version matrix per ecosystem

| Tier | Python / NuGet | npm version | npm dist-tag | Maven |
|------|----------------|-------------|--------------|-------|
| stable | `0.9.0` | `0.9.0` | `latest` | `0.9.0` |
| rc | `0.9.0-rc.1` | `0.9.0-rc.1` | `next` | `0.9.0-rc.1` |
| beta | `0.9.0-beta.1` | `0.9.0-beta.1` | `beta` | `0.9.0-beta.1` |
| nightly | `0.9.0-nightly.YYYYMMDD` | `0.9.0-nightly.YYYYMMDD` | `nightly` | `0.9.0-SNAPSHOT` |

Maven nightly uses `-SNAPSHOT` so each daily build overwrites the
previous one without accumulating permanent versions in GitHub Packages.

---

## Installing a specific tier

### Python (PyPI)

```bash
pip install omniql                              # stable
pip install omniql==0.9.0-beta.1               # beta
pip install omniql==0.9.0-rc.1                 # rc
pip install omniql==0.9.0-nightly.20260223     # specific nightly
```

### npm

```bash
npm install omniql            # stable  (latest tag)
npm install omniql@beta       # latest beta
npm install omniql@next       # latest rc
npm install omniql@nightly    # latest nightly
npm install omniql@0.9.0-beta.1  # specific beta version
```

### NuGet (.NET)

```bash
dotnet add package OmniQL                              # stable
dotnet add package OmniQL --version 0.9.0-beta.1      # beta
dotnet add package OmniQL --version 0.9.0-rc.1        # rc
dotnet add package OmniQL --version 0.9.0-nightly.20260223 --prerelease  # nightly
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
<!-- repository declaration (needed for nightly SNAPSHOTs) -->
<repositories>
  <repository>
    <id>github-omniql</id>
    <url>https://maven.pkg.github.com/uttam-mahata/omniql</url>
    <snapshots><enabled>true</enabled></snapshots>
  </repository>
</repositories>

<!-- dependency -->
<dependency>
  <groupId>io.github.uttam-mahata</groupId>
  <artifactId>omniql</artifactId>
  <version>0.9.0</version>          <!-- or 0.9.0-beta.1 / 0.9.0-SNAPSHOT -->
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

1. **`get-version`** — reads the base version from `bindings/python/pyproject.toml`, inspects the tag or schedule trigger, and outputs `version`, `maven_version`, `npm_tag`, and `release_type`.
2. **`build-core`** — compiles `libomniql.so` / `libomniql.dylib` / `omniql.dll` using Go with CGO.
3. **Publish jobs** — each binding downloads the three native binaries, stamps the version, and publishes to its registry.
4. **`goreleaser`** *(added v0.6.0)* — builds the `omniql` CLI binary for all platforms, creates archives, `.deb`/`.rpm` packages, and publishes to GitHub Releases / Homebrew / Scoop. Runs on tag pushes only (not nightly).

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
