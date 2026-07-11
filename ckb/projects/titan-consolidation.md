| Metadata | Value |
| :--- | :--- |
| **Schema Version** | 1.0 |
| **ID** | proj:titan-consolidation |
| **Type** | Project |
| **Status** | Active |
| **Verification Level** | Independently-Verified |
| **Confidence** | 0.95 |
| **Visibility** | Public |
| **Source** | Architecture Specification v2.4 |
| **Last Updated** | 2026-07-10 |
| **Lifecycle State** | Completed |
| **Related Documents** | timeline:main |
| **Related Experience** | exp:stark-devops |
| **Related Evidence** | ev:git-titan-repo, ev:eval-stark-2025, ev:adr-012 |
| **Tags** | cloud-infrastructure, kubernetes, migration |

---

## 1. Project Specifications
*   **Objective**: Consolidate four legacy Kubernetes clusters scattered across three geographical cloud regions into a single, unified, multi-tenant cluster.
*   **Problem**: Development teams were managing isolated deployment configurations, leading to high maintenance costs, security vulnerabilities, and platform drift. Annual running cost exceeded $3.2M due to idle resource allocations.
*   **Solution**: Architected a central multi-tenant EKS cluster utilizing Kubernetes Namespaces, network policies, and virtual clusters (vcluster). Built automated terraform modules to deploy namespaces on demand.

---

## 2. Architecture & Design Decisions
*   **Architecture Detail**: VPC peering and transit gateways allow secure cross-region microservices routing. Integrated Istio service mesh for mutual TLS (mTLS) enforcement. Standardized GitOps deployment workflow using ArgoCD.
*   **Design Decision**: Decided to use a single shared cluster partitioned by Namespaces and NetworkPolicies instead of separate clusters. This drastically reduced AWS control plane costs and simplified maintenance, though it required strict multi-tenant network rules. Shifted deployments to ArgoCD pull-based model rather than Jenkins push-based model to ensure clusters automatically reconcile configuration drift.

---

## 3. Implementation Details
*   **Challenges**: IP Address exhaustion within VPC subnets during mass container restarts. Mitigated by re-architecting AWS VPC CNI plugin settings to allocate secondary CIDR blocks, opening an additional 60,000 container IPs.
*   **Leadership**: Coordinated with seven product development team leads to schedule and execute migrations without database write interruptions. Conducted hands-on training sessions for developers on Kubernetes manifest configurations and logging systems.
*   **Technologies**: Kubernetes, Terraform, ArgoCD, AWS, Go, Istio.

---

## 4. Outcomes & Metrics
*   **Business Impact**: Consolidated scattered resources under single-pane governance, reducing billing fragmentation.
*   **Metrics**:
    *   Saved **$950,000 annually** in infrastructure footprint cost.
    *   Increased overall CPU/Memory utilization from **15% to 62%**.
    *   Automatically resolved 100% of unauthorized cluster configuration changes within 5 minutes.
*   **Future Work**: Transitioning workload scheduling to Karpenter-based dynamic provisioners.
*   **Lessons Learned**: Early standardization of application resource limits (CPU request/limit values) is essential before cluster consolidation. Lack of limits initially caused resource starvation (noisy neighbor issues) for smaller services.
