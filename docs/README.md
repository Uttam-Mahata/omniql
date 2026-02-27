# OmniQL Documentation

Version-specific documentation for OmniQL.

| Version | Highlights |
|---------|------------|
| [v0.7.0](v0.7.0/README.md) | SQL Server driver, Redis driver, sqlutil shared WHERE builder, terminal convenience methods for all bindings, MySQL in compliance suite, fuzz tests |
| [v0.6.0](v0.6.0/README.md) | MySQL driver, batchInsert API for all bindings, compliance suite, omniql.yaml config, GoReleaser |
| [v0.5.0](v0.5.0/README.md) | Nested field support, deterministic fallback driver, BatchInsert end-to-end, schema recursion |
| [v0.4.0](v0.4.0/README.md) | Logical operators ($or/$and), Sort, Projection, Insert returning data, Mongo tests |
| [v0.3.0](v0.3.0/README.md) | FFI driver registration + routing, working CLI, C# JSON fix, binding enhancements, FFI test suite |
| [v0.2.0](v0.2.0/README.md) | Initial release — core engine, SQLite/Postgres/Mongo drivers, FFI layer, language bindings |
| [Agent Skills](gemini-skills/README.md) | Specialized on-demand expertise for Gemini CLI |

---

Each version folder contains:

| File | Contents |
|------|----------|
| `README.md` | Overview, architecture, quick-start |
| `cli.md` | CLI flag reference and examples |
| `ffi.md` | C ABI reference for language binding authors |
| `bindings.md` | Per-language usage guide (Python, Java, C#, TypeScript) |
| `CHANGELOG.md` | What changed relative to the previous version |

---

## Release Strategy

See [releases.md](releases.md) for the full release strategy, tag conventions, and per-ecosystem install instructions.
