| Metadata | Value |
| :--- | :--- |
| **Schema Version** | 1.0 |
| **ID** | exp:stark-devops |
| **Type** | Experience |
| **Status** | Active |
| **Verification Level** | Independently-Verified |
| **Confidence** | 0.95 |
| **Visibility** | Public |
| **Source** | Manager Appraisals & Git Logs |
| **Last Updated** | 2026-07-10 |
| **Lifecycle State** | Active |
| **Related Documents** | timeline:main |
| **Related Projects** | proj:titan-consolidation |
| **Related Evidence** | ev:eval-stark-2025, ev:adr-012, ev:inc-122, ev:soc2-audit-log |
| **Tags** | cloud-infrastructure, kubernetes, engineering-leadership |

---

## 1. Role Context
*   **Role**: Principal DevOps Architect
*   **Duration**: 2025-06 – Present
*   **Location**: Los Angeles, CA (Hybrid)
*   **Mission**: Standardize and consolidate scattered compute platforms across Stark's commercial tech stack onto a unified, multi-region Kubernetes platform while maintaining SOC2 compliance and lowering OpEx.
*   **Responsibilities**:
    *   Direct the architecture of the Kubernetes Core platform.
    *   Govern IAM policies and infrastructure-as-code automation templates.
    *   Mentor 15+ senior infrastructure and DevOps engineers.
    *   Define incident response guidelines and SLA targets.

---

## 2. Key Achievements

### Technical & Architectural
*   Designed a multi-region VPC networking model with global load balancing, ensuring failover times under 15 seconds.
*   Pioneered GitOps infrastructure deployments utilizing ArgoCD, reducing configuration drift incidents by 92%.
*   Refined microservices telemetry pipelines, introducing eBPF-based agent monitoring for near-zero runtime overhead.

### Leadership & Initiatives
*   Formed the Stark Platform Guild, a bi-weekly architecture review board, improving alignment across five separate business units.
*   Instituted blameless post-mortems, shifting team culture toward preventive engineering.

### Business Impact & Metrics
*   **Cost Reduction**: Saved **$1.2M in annual cloud spend** by implementing autoscaling configurations and spot-instance node groups.
*   **Developer Velocity**: Reduced developer onboarding times from **4 days to 30 minutes** through developer self-service tooling.

---

## 3. STAR Stories

### STAR: Resolving Multi-Region Failover Latency
*   **Situation**: During a peak load simulator run, the platform experienced a regional outage. Dynamic traffic routing failed, resulting in 4 minutes of service interruption.
*   **Task**: Re-engineer the global load balancer configuration to automatically detect unhealthy regions and reroute traffic within Stark's strict 30-second service-level objective.
*   **Action**: Designed a custom health checking script that reports real-time database replication lag and system load. Wired this query directly to global routing profiles and implemented DNS-based fallback routing rules.
*   **Result**: Reduced maximum client failover latency to **12 seconds** during subsequent stress runs. Zero downtime recorded during the next real region failover.
*   **Evidence Ref**: [ev:inc-122](../evidence.md)

---

## 4. Technologies & Methodologies
*   **Technologies**: Kubernetes, Terraform, Go, Prometheus, Grafana, AWS (EKS, Route53, VPC), ArgoCD.
*   **Methodologies**: GitOps, Zero-Trust Access, Blameless Post-Mortems, Agile Infrastructure.
