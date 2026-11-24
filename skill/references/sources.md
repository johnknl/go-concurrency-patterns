# Bibliography

Every pattern page ends with the places where the pattern is described, cited by author and year. The
references below are in alphabetical order, in APA style, and each one lists the pages that cite it.

A source is listed once at least one page cites it.

## References

Ajmani, S. (2014a). Go concurrency patterns: Context. *The Go Blog*. <https://go.dev/blog/context>  
    Cited in: [Context Propagation](stdlib/context-propagation.md); [Cooperative Cancellation](concepts/cooperative-cancellation.md); [Ignored Context](anti-patterns/ignored-context.md)

Ajmani, S. (2014b). Go concurrency patterns: Pipelines and cancellation. *The Go Blog*. <https://go.dev/blog/pipelines>  
    Cited in: [Channel Ownership](concepts/channel-ownership.md); [Cooperative Cancellation](concepts/cooperative-cancellation.md) ("Explicit cancellation"); [Fan-Out/Fan-In](patterns/fan-out-fan-in.md) ("Fan-out, fan-in"); [Goroutine Leak Prevention](concepts/goroutine-leak-prevention.md) ("Explicit cancellation"); [Pipeline](patterns/pipeline.md) ("What is a pipeline?")

Akidau, T., Balikov, A., Bekiroğlu, K., Chernyak, S., Haberman, J., Lax, R., McVeety, S., Mills, D., Nordstrom, P., & Whittle, S. (2013). MillWheel: Fault-tolerant stream processing at internet scale. *Proceedings of the VLDB Endowment*, 6(11), 1033–1044. <https://doi.org/10.14778/2536222.2536229>  
    Cited in: [Watermark](patterns/watermark.md)

Akidau, T., Bradshaw, R., Chambers, C., Chernyak, S., Fernández-Moctezuma, R. J., Lax, R., McVeety, S., Mills, D., Perry, F., Schmidt, E., & Whittle, S. (2015). The dataflow model: A practical approach to balancing correctness, latency, and cost in massive-scale, unbounded, out-of-order data processing. *Proceedings of the VLDB Endowment*, 8(12), 1792–1803. <https://doi.org/10.14778/2824032.2824076>  
    Cited in: [Watermark](patterns/watermark.md); [Windowing](patterns/windowing.md)

Akidau, T., Chernyak, S., & Lax, R. (2018). *Streaming systems: The what, where, when, and how of large-scale data processing*. O'Reilly Media.  
    Cited in: [Watermark](patterns/watermark.md)

Baker, H. C., & Hewitt, C. (1977). The incremental garbage collection of processes. *ACM SIGPLAN Notices*, 12(8), 55–59. <https://doi.org/10.1145/872734.806932>  
    Cited in: [Future](patterns/future.md)

Buschmann, F., Meunier, R., Rohnert, H., Sommerlad, P., & Stal, M. (1996). *Pattern-oriented software architecture, volume 1: A system of patterns*. John Wiley & Sons.  
    Cited in: [PubSub Hub](patterns/pubsub-hub.md) (Publisher-Subscriber)

Coffman, E. G., Elphick, M. J., & Shoshani, A. (1971). System deadlocks. *ACM Computing Surveys*, 3(2), 67–78. <https://doi.org/10.1145/356586.356588>  
    Cited in: [Inconsistent Lock Order](anti-patterns/inconsistent-lock-order.md)

Cox-Buday, K. (2017). *Concurrency in Go: Tools and techniques for developers*. O'Reilly Media. <https://www.oreilly.com/library/view/concurrency-in-go/9781491941294/>  
    Cited in: [Bounded Queue](patterns/bounded-queue.md) (Ch. 4, "Queuing"); [Channel Generator](anti-patterns/channel-generator.md) (Ch. 4, "Some Handy Generators"); [Channel Ownership](concepts/channel-ownership.md) (Ch. 4, "Confinement"); [Confinement](concepts/confinement.md) (Ch. 4, "Confinement"); [Context Propagation](stdlib/context-propagation.md) (Ch. 4, "The context Package"); [Cooperative Cancellation](concepts/cooperative-cancellation.md) (Ch. 5, "Timeouts and Cancellation"); [Deadline](stdlib/deadline.md) (Ch. 5, "Timeouts and Cancellation"); [Fan-Out/Fan-In](patterns/fan-out-fan-in.md) (Ch. 4, "Fan-Out, Fan-In"); [For-Select Loop](patterns/for-select-loop.md) (Ch. 4, "The for-select Loop"); [Goroutine Leak Prevention](concepts/goroutine-leak-prevention.md) (Ch. 4, "Preventing Goroutine Leaks"); [Healing Goroutines](patterns/healing-goroutines.md) (Ch. 5, "Healing Unhealthy Goroutines"); [Heartbeat](patterns/heartbeat.md) (Ch. 5, "Heartbeats"); [Object Cache](patterns/object-cache.md) (Ch. 3, "Pool"); [Or-Done](patterns/or-done.md) (Ch. 4, "The or-done-channel"); [Pipeline](patterns/pipeline.md) (Ch. 4, "Pipelines"); [Request Racing](patterns/request-racing.md) (Ch. 5, "Replicated Requests"); [Result with Error](patterns/result-with-error.md) (Ch. 4, "Error Handling"); [Tee](patterns/tee.md) (Ch. 4, "The tee-channel"); [Token Bucket](patterns/token-bucket.md) (Ch. 5, "Rate Limiting")

Cuervo, A. F. (2016). Handling overload. *Site reliability engineering: How Google runs production systems*. <https://sre.google/sre-book/handling-overload/>  
    Cited in: [Buffer as a Band-Aid](anti-patterns/oversized-buffer.md); [Load Shedding](patterns/load-shedding.md); [Overload Policy](concepts/overload-policy.md)

Dean, J., & Barroso, L. A. (2013). The tail at scale. *Communications of the ACM*, 56(2), 74–80. <https://doi.org/10.1145/2408776.2408794>  
    Cited in: [Request Racing](patterns/request-racing.md)

DeWitt, D. J., Katz, R. H., Olken, F., Shapiro, L. D., Stonebraker, M. R., & Wood, D. A. (1984). Implementation techniques for main memory database systems. *Proceedings of the 1984 ACM SIGMOD International Conference on Management of Data*, 1–8. <https://doi.org/10.1145/602259.602261>  
    Cited in: [Group Commit](patterns/group-commit.md) (Section 5.2, "Limits to Transaction Throughput")

Dijkstra, E. W. (1965). *Cooperating sequential processes* (EWD 123). Technological University Eindhoven. <https://www.cs.utexas.edu/~EWD/transcriptions/EWD01xx/EWD123.html>  
    Cited in: [Semaphore](patterns/semaphore.md)

Gerrand, A. (2010a). Go concurrency patterns: Timing out, moving on. *The Go Blog*. <https://go.dev/blog/concurrency-timeouts>  
    Cited in: [Deadline](stdlib/deadline.md); [Timed Send](patterns/timed-send.md)

Gerrand, A. (2010b). Share memory by communicating. *The Go Blog*. <https://go.dev/blog/codelab-share>  
    Cited in: [Confinement](concepts/confinement.md); [Pointer over a Channel](anti-patterns/pointer-over-channel.md); [Snapshot via Owner](patterns/snapshot-via-owner.md)

Harsanyi, T. (2022). *100 Go mistakes and how to avoid them*. Manning. <https://www.manning.com/books/100-go-mistakes-and-how-to-avoid-them>  
    Cited in: [Fire-and-Forget Goroutine](anti-patterns/fire-and-forget-goroutine.md); [Sleeping to Synchronize](anti-patterns/sleep-synchronization.md); [Unbounded Goroutines](anti-patterns/unbounded-goroutines.md)

Hendler, D., Incze, I., Shavit, N., & Tzafrir, M. (2010). Flat combining and the synchronization-parallelism tradeoff. *Proceedings of the 22nd Annual ACM Symposium on Parallelism in Algorithms and Architectures*, 355–364. <https://doi.org/10.1145/1810479.1810540>  
    Cited in: [Group Commit](patterns/group-commit.md)

Herlihy, M., & Shavit, N. (2012). *The art of multiprocessor programming (Rev. 1st ed.)*. Morgan Kaufmann.  
    Cited in: [Compare-and-Swap Loop](patterns/compare-and-swap-loop.md); [Copy-on-Write Snapshot](patterns/copy-on-write-snapshot.md); [Memory Visibility](concepts/memory-visibility.md)

Hewitt, C., Bishop, P., & Steiger, R. (1973). A universal modular ACTOR formalism for artificial intelligence. *Proceedings of the 3rd International Joint Conference on Artificial Intelligence*, 235–245. <https://www.ijcai.org/Proceedings/73/Papers/027B.pdf>  
    Cited in: [Active Object](patterns/active-object.md)

Hoare, C. A. R. (1974). Monitors: An operating system structuring concept. *Communications of the ACM*, 17(10), 549–557. <https://doi.org/10.1145/355620.361161>  
    Cited in: [Condition Variable](stdlib/condition-variable.md); [Monitor](patterns/monitor.md)

Hohpe, G., & Woolf, B. (2003). *Enterprise integration patterns: Designing, building, and deploying messaging solutions*. Addison-Wesley. <https://www.enterpriseintegrationpatterns.com/patterns/messaging/Resequencer.html>  
    Cited in: [Ordering](concepts/ordering.md); [Resequencer](patterns/resequencer.md) (Resequencer)

Lampson, B. W., & Redell, D. D. (1980). Experience with processes and monitors in Mesa. *Communications of the ACM*, 23(2), 105–117. <https://doi.org/10.1145/358818.358824>  
    Cited in: [Condition Variable](stdlib/condition-variable.md); [Monitor](patterns/monitor.md)

McGranaghan, M., & Bendersky, E. (n.d.). Worker pools. *Go by Example*. <https://gobyexample.com/worker-pools>  
    Cited in: [Worker Pool](patterns/worker-pool.md)

Metz, S. (2016). The wrong abstraction. *sandimetz.com*. <https://sandimetz.com/blog/2016/1/20/the-wrong-abstraction>  
    Cited in: [Premature Abstraction](anti-patterns/premature-abstraction.md)

Meyers, S., & Alexandrescu, A. (2004). C++ and the perils of double-checked locking. *Dr. Dobb's Journal*, . <https://www.aristeia.com/Papers/DDJ_Jul_Aug_2004_revised.pdf>  
    Cited in: [Double-Checked Locking](anti-patterns/double-checked-locking.md)

Nygard, M. T. (2018). *Release it! Design and deploy production-ready software (2nd ed.)*. Pragmatic Bookshelf. <https://pragprog.com/titles/mnee2/release-it-second-edition/>  
    Cited in: [Backpressure](concepts/backpressure.md) (Ch. 5, "Create Back Pressure"); [Load Shedding](patterns/load-shedding.md) (Ch. 5, "Shed Load"); [Overload Policy](concepts/overload-policy.md) (Ch. 5, "Shed Load" and "Create Back Pressure")

Pike, R. (2012). *Go concurrency patterns* [Conference presentation]. Google I/O. <https://go.dev/talks/2012/concurrency.slide>  
    Cited in: [Channel Generator](anti-patterns/channel-generator.md) (Generator: function that returns a channel); [Fan-Out/Fan-In](patterns/fan-out-fan-in.md) (Fan-in)

Pike, R. (2015). *Go proverbs* [Conference presentation]. Gopherfest SV. <https://go-proverbs.github.io/>  
    Cited in: [Premature Abstraction](anti-patterns/premature-abstraction.md) ("A little copying is better than a little dependency")

Reactive Streams. (n.d.). Reactive streams. *reactive-streams.org*. <https://www.reactive-streams.org/>  
    Cited in: [Backpressure](concepts/backpressure.md); [Overload Policy](concepts/overload-policy.md)

ReactiveX. (n.d.). Subject. *ReactiveX documentation*. <https://reactivex.io/documentation/subject.html>  
    Cited in: [Replay Buffer](patterns/replay-buffer.md) (ReplaySubject)

Schmidt, D. C., Stal, M., Rohnert, H., & Buschmann, F. (2000). *Pattern-oriented software architecture, volume 2: Patterns for concurrent and networked objects*. John Wiley & Sons.  
    Cited in: [Active Object](patterns/active-object.md) (Active Object); [Monitor](patterns/monitor.md) (Monitor Object)

Schneider, F. B. (1990). Implementing fault-tolerant services using the state machine approach: A tutorial. *ACM Computing Surveys*, 22(4), 299–319. <https://doi.org/10.1145/98163.98167>  
    Cited in: [Determinism](concepts/determinism.md); [Non-Deterministic State Machine](anti-patterns/non-deterministic-state-machine.md); [State Machine](patterns/state-machine.md)

The Go Authors. (2022). The Go memory model. *go.dev*. <https://go.dev/ref/mem>  
    Cited in: [Channel Ownership](concepts/channel-ownership.md); [Double-Checked Locking](anti-patterns/double-checked-locking.md); [Lazy Initialization](stdlib/lazy-initialization.md); [Memory Visibility](concepts/memory-visibility.md); [Ordering](concepts/ordering.md)

The Go Authors. (n.d.j). bufio: Scanner.Err. *pkg.go.dev*. <https://pkg.go.dev/bufio#Scanner.Err>  
    Cited in: [Close with Cause](patterns/close-with-cause.md)

The Go Authors. (n.d.a). Effective Go. *go.dev*. <https://go.dev/doc/effective_go>  
    Cited in: [Semaphore](patterns/semaphore.md) ("Channels")

The Go Authors. (n.d.b). Go code review comments. *Go wiki*. <https://go.dev/wiki/CodeReviewComments>  
    Cited in: [Needless Asynchrony](anti-patterns/needless-asynchrony.md) (Synchronous Functions)

The Go Authors. (n.d.c). Package copylock. *pkg.go.dev*. <https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/copylock>  
    Cited in: [Copied Mutex](anti-patterns/copied-mutex.md)

The Go Authors. (n.d.d). Package rate. *pkg.go.dev*. <https://pkg.go.dev/golang.org/x/time/rate>  
    Cited in: [Token Bucket](patterns/token-bucket.md)

The Go Authors. (n.d.e). Package sync. *pkg.go.dev*. <https://pkg.go.dev/sync>  
    Cited in: [Condition Variable](stdlib/condition-variable.md); [Copied Mutex](anti-patterns/copied-mutex.md); [Double-Checked Locking](anti-patterns/double-checked-locking.md); [Lazy Initialization](stdlib/lazy-initialization.md); [Memory Visibility](concepts/memory-visibility.md); [Monitor](patterns/monitor.md); [Object Cache](patterns/object-cache.md)

The Go Authors. (n.d.f). Package sync/atomic. *pkg.go.dev*. <https://pkg.go.dev/sync/atomic>  
    Cited in: [Compare-and-Swap Loop](patterns/compare-and-swap-loop.md); [Copy-on-Write Snapshot](patterns/copy-on-write-snapshot.md) (Value, examples "Config" and "ReadMostly"); [Memory Visibility](concepts/memory-visibility.md)

The Go Authors. (n.d.g). Package testing/synctest. *pkg.go.dev*. <https://pkg.go.dev/testing/synctest>  
    Cited in: [Determinism](concepts/determinism.md); [Synctest](stdlib/synctest.md); [Timers and Tickers](concepts/timers.md)

The Go Authors. (n.d.h). Package time. *pkg.go.dev*. <https://pkg.go.dev/time>  
    Cited in: [Timers and Tickers](concepts/timers.md)

The Go Authors. (n.d.i). Use a sync.Mutex or a channel? *Go wiki*. <https://go.dev/wiki/MutexOrChannel>  
    Cited in: [Channel Where a Mutex Would Do](anti-patterns/channel-instead-of-mutex.md); [Confinement](concepts/confinement.md)

The Gorilla WebSocket Authors. (2013). Chat example. *gorilla/websocket*. <https://github.com/gorilla/websocket/tree/main/examples/chat>  
    Cited in: [Online Attach](patterns/online-attach.md)

Tu, T., Liu, X., Song, L., & Zhang, Y. (2019). Understanding real-world concurrency bugs in Go. *Proceedings of the 24th International Conference on Architectural Support for Programming Languages and Operating Systems*, 865–878. <https://doi.org/10.1145/3297858.3304069>  
    Cited in: [Closing from the Wrong Side](anti-patterns/wrong-side-close.md); [Fire-and-Forget Goroutine](anti-patterns/fire-and-forget-goroutine.md); [Lock Held While Blocking](anti-patterns/lock-held-while-blocking.md); [Sleeping to Synchronize](anti-patterns/sleep-synchronization.md)

Tucker, P. A., Maier, D., Sheard, T., & Fegaras, L. (2003). Exploiting punctuation semantics in continuous data streams. *IEEE Transactions on Knowledge and Data Engineering*, 15(3), 555–568. <https://doi.org/10.1109/TKDE.2003.1198390>  
    Cited in: [Watermark](patterns/watermark.md)
