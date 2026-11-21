# Standard library

Some coordination that once needed a pattern is now part of Go's standard library. These guides show how to use it
correctly: what it provides, the code that uses it, and its pitfalls.

<!-- catalog: stdlib -->
| Page | Summary |
|---|---|
| [Condition Variable](condition-variable.md) | Let goroutines wait until a condition on shared state holds, and wake them when it may have changed. |
| [Context Propagation](context-propagation.md) | Stop all the work done for a request together, and give it the request's deadline and scoped values. |
| [Deadline](deadline.md) | Bound how long an operation may take. |
| [Lazy Initialization](lazy-initialization.md) | Build a value on first use, exactly once, safely from many goroutines. |
| [Synctest](synctest.md) | Test concurrent and time-dependent code deterministically, with fake time and a way to wait until every goroutine is blocked. |
<!-- /catalog -->
