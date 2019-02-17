# lockerd

lockerd, not-so-cheekily named in honor of the rather great [linkerd](https://linkerd.io), is a distributed locking service designed to expose a simple interface easily consumed in a language-agnostic manner.


## Rationale

In any sufficiently complex system, locking to protect resource access is a common solution to a number of problems. However, currently available solutions suffer from a number of issues, that make development using this construct difficult.

Firstly, most available solutions require a specialized pattern of use of some more generalized primitives to provide save locking, such as is the case with Redis, ZooKeeper, and, to some extent, etcd – and the same holds true if using even more general solutions like a relational database to solve the issue. These patterns must be correctly implemented across all participating clients, which complicates implementation.

Secondarily, a number of these solutions require the use of specialized protocols and often clients that are difficult to easily implement in a number of languages, ZooKeeper being the main offender in this area, while others de facto require reliance on clients for which the behavior is fairly opaque in failure scenarios. In extension hereof, many of these protocols require persistent connections or sessions to be maintained, which attenuates the aforementioned problem.

Lastly, a number of these solutions by default share resources with a set of other primitives, causing either a much greater degree of production complexity in both observing and separating these concerns. To be fair, this is innately true of any system of growing complexity no matter the level of separation, so this argument is not as strong as the overall arguments in favor of development simplicity outlined above.

In summary, in an effort to build a service architecture in which fairly common building blocks are readily available in an easily accessible manner no matter the language and environment, lockerd is built from the perspective of desiring a much simpler interface and set of constructs upon which to reliably build distributed applications.


## Goals

Partially from the rationale above falls the following set of goals for lockerd:

* **Simple.** lockerd should be simple to understand and use with as few caveats as possible. Whenever feature complexity is to be weighed against increasing complexity in understanding, the design should err on the side of simplicity in lieu of overwhelming arguments to the opposite.
* **Stateless API using well-supported protocols.** lockerd should provide its entire spectrum of functionality using well-supported, stateless protocols (initially HTTP), so that no complex logic is required to initiate and maintain locks.
* **Observable.** lockerd should provide an easy interface for observing the current state of locks so as to help diagnose especially
* **Operationally easy.** lockerd should be incredibly easy to deploy and require little to no ongoing maintenance beyond scaling concerns in order to function. The name inspiration, linkerd, is a great inspiration in this regard.


## Roadmap

* **Durability**. In its current early state, lockerd does not persist nor replicate its state, making it fairly fragile in the face of operational disruption. The future plans are to add both disk persistence and a truly distributed replication system.
* **Performance**. The performance of lockerd is as of right now fully untested, and there are clear avenues of scalability challenges with regards to both the total number of locks outstanding as well as the contention around each lock that are to be
* **Adding more interfaces.** lockerd currently only exposes a simple REST-like HTTP API interface, but it is conceivable that other interfaces could be useful
* **Adding more complex locking constructs.** Readers-writer locks and semaphores are very useful constructs and could easily be supported and exposed by lockerd in the future.
