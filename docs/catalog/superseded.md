# Superseded

Patterns that later features of the language or the standard library have largely replaced. Their pages stay,
because the patterns still appear in code and in the literature, and each says what to use instead. See the
[other axes](index.md).

<!-- catalog: superseded -->
| Pattern | Instead |
|---|---|
| [Or-Done](../patterns/or-done.md) | Done channels have been largely superseded by `context.Context`. The wrapper works the same with `ctx.Done()` in place of `done`; only code that still hands out bare done channels needs the done-channel form. |
<!-- /catalog -->
