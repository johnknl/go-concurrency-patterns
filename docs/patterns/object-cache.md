---
title: Object Cache
aliases: ['Free list']
intent: Avoid allocating and garbage-collecting the same kind of temporary object over and over in a hot path.
purpose: coordination
scale: idiom
package: patterns/objectcache
mechanisms: [sync]
concerns: [throughput]
sources:
  - id: cox-buday
    at: 'Ch. 3, "Pool"'
  - id: go-sync
---
# Object Cache

<!-- meta -->
Avoid allocating and garbage-collecting the same kind of temporary object over and over in a hot path.

<div class="cp-props" markdown>

| | |
|---|---|
| Also known as | Free list |
| Purpose | [Coordination](../catalog/purpose.md#purpose-coordination) |
| Scale | [Idiom](../catalog/scale.md#scale-idiom) |
| Mechanisms | [Sync](../catalog/mechanism.md#mechanism-sync) |
| Concerns | [Throughput](../catalog/concern.md#concern-throughput) |
| Code | [`patterns/objectcache`](https://github.com/johnknl/go-concurrency-patterns/tree/main/patterns/objectcache) |

</div>
<!-- /meta -->

## Problem

Short-lived objects that are created over and over in a hot path create garbage and allocator work. The objects
are interchangeable, so keeping used ones for the next caller saves the allocation and the later collection.

## Mechanism

`sync.Pool` holds temporary objects. `Get` returns an item, or calls `New` when there is none; `Put` returns one.
The user resets the object before `Put`. A `sync.Pool` is safe for concurrent use and must not be copied after
first use.

<!-- code: patterns/objectcache#buffers patterns/objectcache#render patterns/objectcache#Example_syncPool:main -->
```go
var buffers = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

func render(name string) string {
	buf := buffers.Get().(*bytes.Buffer) //nolint:errcheck // New only creates buffers
	defer func() {
		buf.Reset()
		buffers.Put(buf)
	}()

	fmt.Fprintf(buf, "hello, %s", name)

	return buf.String()
}

func main() {
	for _, name := range []string{"ann", "bob", "cy"} {
		buf := buffers.Get().(*bytes.Buffer) //nolint:errcheck // New only creates buffers
		fmt.Println("empty:", buf.Len() == 0)
		buffers.Put(buf)

		fmt.Println(render(name))
	}

	// Output:
	// empty: true
	// hello, ann
	// empty: true
	// hello, bob
	// empty: true
	// hello, cy
}
```
<!-- /code -->

`sync.Pool` makes no promise about what it holds: the runtime may drop items at any garbage collection, under the
race detector `Put` randomly discards some, there is no size limit, and `Get` may return any item that was put or a
new one. Correctness must never depend on reuse; the example prints the same output either way.

## Applicability

Use it when the objects are temporary, interchangeable, and costly to allocate or collect, and a profile shows that
the allocation matters. Cheap objects gain nothing. For resources that must be bounded in number, such as
connections, use a [Resource Pool](resource-pool.md): a `sync.Pool` may drop them at any time without closing them.

## Consequences

- A forgotten reset leaks state from one user to the next.
- An object must not be used after `Put`; the next user may already have it.
- Store pointers: putting a non-pointer value, such as a slice header, converts it to `any`, which allocates.
- A pool can retain unusually large objects (a buffer that grew once) unless the user discards them instead of
  calling `Put`.

## Related patterns

- [Resource Pool](resource-pool.md) hands out a fixed set of resources and bounds their number.
- [Worker Pool](worker-pool.md) keeps goroutines, not objects, for reuse.

## Sources

<!-- sources -->
- [Cox-Buday (2017)](../sources.md#source-cox-buday), Ch. 3, "Pool"
- [The Go Authors (n.d.e)](../sources.md#source-go-sync)
<!-- /sources -->
