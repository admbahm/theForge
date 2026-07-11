| Metadata | Value |
| :--- | :--- |
| **Schema Version** | 1.0 |
| **ID** | exp:acme-lead |
| **Type** | Experience |
| **Status** | Active |
| **Verification Level** | Independently-Verified |
| **Confidence** | 1.00 |
| **Visibility** | Public |
| **Source** | HR Records & Exit Interview |
| **Last Updated** | 2026-07-10 |
| **Lifecycle State** | Completed |
| **Related Documents** | timeline:main |
| **Related Projects** | proj:phoenix-gateway |
| **Related Evidence** | ev:award-acme-excellence, ev:git-phoenix-repo, ev:git-acme-repo |
| **Tags** | cloud-systems, api-gateway, microservices |

---

## 1. Role Context
*   **Role**: Lead Infrastructure Engineer
*   **Duration**: 2022-01 – 2024-08
*   **Location**: Remote
*   **Mission**: Scale Acme's microservices API gateway layer to handle a 300% increase in API transaction volume while stabilizing latency metrics.
*   **Responsibilities**:
    *   Maintain ownership of Acme's API routing and proxy services.
    *   Establish SLO/SLA monitoring dashboards for core routes.
    *   Review and approve team code submissions in Go and Python.

---

## 2. Key Achievements

### Technical & Architectural
*   Rewrote the core routing proxy in Go, substituting the legacy Python service. This reduced memory footprints by **75%** and cut p99 response times from **340ms to 12ms**.
*   Implemented rate-limiting filters (Token Bucket algorithm) protecting database engines from surge traffic spikes.

### Business Impact & Metrics
*   **Scale**: Safely scaled operations from **5,000 to 22,000 requests per second** at peak without degradation.
*   **Stability**: Achieved **99.99% uptime** across core gateways over a 12-month trailing window.

---

## 3. Technologies & Methodologies
*   **Technologies**: Go, Redis, Docker, Envoy, Nginx, PostgreSQL, Google Cloud Platform (GCP).
*   **Methodologies**: TDD, SRE, SLO/SLA Management.
