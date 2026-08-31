# Agent instructions

- Add comments only when they explain non-obvious behavior, a safety constraint,
  or an important architectural decision.
- Do not add comments that merely restate a name, type, or function signature.
- Use Conventional Commits without scopes: `feat: description`,
  `fix: description`, `refactor: description`, `test: description`, or
  `docs: description`.
- Run `gofmt` and `go test ./...` before committing Go changes.
