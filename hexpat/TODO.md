# Generator TODO

## Tier 2 — Extended Type Support

- [ ] `while`-terminated arrays
- [ ] Relative pointers (`[[pointer_base()]]`)
- [ ] Nullable pointers
- [ ] `$` (cursor) reads

## Codegen Enhancements

- [x] Lazy reader generation (`*FooReader` with per-field accessors, zero-I/O sub-readers for nested structs, lazy pointer following) — static-offset structs only
- [ ] Lazy reader support for dynamic-offset structs (conditionals, expression-length arrays)
- [ ] `sizeof()` / `addressof()` support in generated code
