| Metadata | Value |
| :--- | :--- |
| **Schema Version** | 1.0 |
| **ID** | proj:phoenix-gateway |
| **Type** | Project |
| **Status** | Active |
| **Verification Level** | Independently-Verified |
| **Confidence** | 1.00 |
| **Visibility** | Public |
| **Source** | Design Document v1.0 |
| **Last Updated** | 2026-07-10 |
| **Lifecycle State** | Completed |
| **Related Documents** | timeline:main |
| **Related Experience** | exp:acme-lead |
| **Related Evidence** | ev:git-phoenix-repo, ev:award-acme-excellence |
| **Tags** | systems-engineering, go, api-gateway |

---

## 1. Project Specifications
*   **Objective**: Replace a legacy Python-based API routing proxy with a high-throughput, low-latency API gateway to prevent outages during flash sale events.
*   **Problem**: The legacy gateway regularly encountered thread-pool exhaustion when handling concurrency spikes above 5,000 requests per second, causing database connection failures downstream.
*   **Solution**: Re-engineered the gateway routing logic in Go, leveraging a lock-free routing tree and Redis-backed rate limiting.

---

## 2. Architecture & Design Decisions
*   **Architecture Detail**: Locking structures were eliminated in favor of concurrency-friendly Go primitives and a trie-based search routing map. Distributed rate limit counters are shared via a local cluster Redis store.
*   **Design Decision**: Refused third-party heavyweight gateway products to maintain low memory footprints and direct code-level control over request filtering protocols.

---

## 3. Implementation Details
*   **Challenges**: Memory allocations under extreme GC pressure. Avoided allocating temporary buffers by recycling byte slices in pools.
*   **Leadership**: Technical lead for a 3-person infrastructure sub-team, coordinating deployment pipelines and metrics.
*   **Technologies**: Go, Redis, Docker, Envoy, Nginx.

---

## 4. Outcomes & Metrics
*   **Business Impact**: Handled flash-sale traffic with zero customer-facing route degradation.
*   **Metrics**:
    *   Successfully handled peak loads of **30,000 requests per second** (500% improvement).
    *   p99 response latency reduced from **340ms to 8ms**.
    *   Compute server instances required reduced from **24 down to 3** (saved $150,000/yr).
*   **Lessons Learned**: Avoid adding heavy logging statements on high-throughput routing paths, as disk I/O bottlenecks can negate algorithm performance gains.
