# HALMIDI SDS — MASTER PRD
## Product Requirements Document — Enterprise Multi-Protocol DASE Storage Architecture

* **Document Baseline:** Master Architecture Specification
* **Data Model:** Disaggregated Shared-Everything (DASE)
* **Compute Runtime:** Pure Go (`CGO_ENABLED=0`)
* **Fabric Acceleration:** SPDK SMA + NVMe-oF / Hugepages

---

## 1. Executive Summary & Vision

**Halmidi SDS** is an enterprise-grade Software-Defined Storage platform engineered for unstructured data workloads. Named after the historic Halmidi rock inscription—symbolizing permanent, immutable persistence—the system eliminates the historical divide between object storage (AWS S3) and POSIX file protocols (NFSv3/v4 and SMB2/v3).

Unlike traditional shared-nothing architectures that scale storage and compute in locked steps, Halmidi natively implements a **Disaggregated Shared-Everything (DASE)** architecture. Stateless compute nodes handle protocol termination, Reed-Solomon Erasure Coding, and caching, while talking to high-performance media via SPDK and NVMe-oF. To ensure portable container builds and deterministic tail latencies, the entire compute tier adheres to a strict **pure Go runtime (`CGO_ENABLED=0`)**, isolating data movement from Go GC interference via aligned memory pools and lockless memory pipelines.

---

## 2. Dual-Topology System Architecture

Halmidi operates under a unified codebase that transparently discovers and adapts to two core topologies: a localized single-node shared-memory mode for development and edge pods, and a distributed multi-node DASE fabric for enterprise datacenters.

### 2.1 Topology A: Single-Node (Shared Memory / Local Dev / Pod)

In Topology A, all services reside on a single physical host or Kubernetes pod. Data transfer between the Go compute process and SPDK bypasses network stacks and kernel context switches using Linux Hugepages and lockless ring buffers.


```

┌─────────────────────────────────────────────────────────────────────────────┐
│ SINGLE PHYSICAL SERVER / LOCAL KUBERNETES POD RUNTIME                       │
│                                                                             │
│  [ Clients: S3 REST / NFS Mount / SMB Share ]                               │
│         │                  │               │                                │
│         ▼                  ▼               ▼                                │
│  ┌──────────────┐   ┌─────────────┐ ┌─────────────┐                         │
│  │ Halmidi S3   │   │ NFS Ganesha │ │    Samba    │                         │
│  │ (Native Go)  │   │  (FSAL C)   │ │  (VFS C)    │                         │
│  └──────┬───────┘   └──────┬──────┘ └──────┬──────┘                         │
│         │                  │ (nfs.sock)    │ (smb.sock)                     │
│         ▼                  ▼               ▼                                │
│  ┌────────────────────────────────────────────────┐                         │
│  │        HALMIDI CORE COMPUTE PROCESS (Go)       │                         │
│  │  • Pure Go (CGO_ENABLED=0)                     │                         │
│  │  • Omashu (Embedded Raft + BadgerDB Metadata)  │                         │
│  │  • dotfs Engine (Stripe chunking & RS EC)      │                         │
│  │  • Zero-Heap sync.Pool (4KB-aligned slices)    │                         │
│  └───────────────────────┬────────────────────────┘                         │
│                          │                                                  │
│       Control: UDS gRPC  │   Data: Lockless Ring Buffer (Atomic Head/Tail)  │
│       to SPDK SMA        │   via /dev/hugepages (2MB/1GB pages)             │
│                          ▼                                                  │
│  ┌────────────────────────────────────────────────┐                         │
│  │            SPDK STORAGE PROCESS (C)            │                         │
│  │  • Storage Management Agent (SMA) over UDS     │                         │
│  │  • User-Space Polling Mode Driver (PMD)        │                         │
│  │  • Direct PCIe NVMe Driver (Kernel Bypass)     │                         │
│  └───────────────────────┬────────────────────────┘                         │
│                          ▼                                                  │
│              [ LOCAL NVMe SSD DRIVES ]                                      │
└─────────────────────────────────────────────────────────────────────────────┘

```

### 2.2 Topology B: Multi-Node (Disaggregated Shared-Everything / DASE)

In Topology B, stateless compute nodes scale independently from shared JBOF storage pools. Compute nodes talk to storage nodes over 100G/200G NVMe-oF (TCP or RoCEv2), orchestrating target lifecycle via SPDK SMA gRPC interfaces.


```

┌────────────────────────────────────────────────────────────────────────────────────────┐
│ ENTERPRISE CLUSTER TIER (High-Availability L4/L7 Load Balancer / VIP via Keepalived)   │
└───────────────────────┬───────────────────────────────┬────────────────────────────────┘
│                               │
┌───────────────────────▼───────────────────────────────▼────────────────────────────────┐
│ STATELESS COMPUTE TIER (Pure Go Cattle Nodes - Scale Compute & Protocol IOPS)          │
│                                                                                        │
│  ┌─────────────────────────────────────────┐  ┌─────────────────────────────────────┐  │
│  │             COMPUTE NODE 1              │  │           COMPUTE NODE 2            │  │
│  │  • Halmidi Core (S3 Gateway, DLM, ACL)  │  │  • Halmidi Core (S3, DLM, ACL)      │  │
│  │  • NFS-Ganesha & Samba Gateways         │  │  • NFS-Ganesha & Samba Gateways     │  │
│  │  • Omashu Raft Consensus Member         │  │  • Omashu Raft Consensus Member     │  │
│  │  • Control: Remote SPDK SMA gRPC Client │  │  • Control: Remote SMA gRPC Client  │  │
│  │  • Data: Kernel NVMe-oF Initiator       │  │  • Data: Kernel NVMe-oF Initiator   │  │
│  │    (Direct I/O via os.O_DIRECT)         │  │    (Direct I/O via os.O_DIRECT)     │  │
│  └────────────────────┬────────────────────┘  └──────────────────┬──────────────────┘  │
└───────────────────────┼──────────────────────────────────────────┼─────────────────────┘
│                                          │
▼                                          ▼
════════════════════════════════════════════════════════════════════════════════════
HIGH-SPEED STORAGE FABRIC (100G / 200G NVMe-oF TCP / RoCEv2)
════════════════════════════════════════════════════════════════════════════════════
│                                          │
┌───────────────────────┼──────────────────────────────────────────┼─────────────────────┐
│ SHARED STORAGE TIER (JBOF Storage Nodes - Scale Media & Raw Capacity)                  │
│                                                                                        │
│  ┌─────────────────────────────────────────┐  ┌─────────────────────────────────────┐  │
│  │          STORAGE JBOF NODE 1            │  │         STORAGE JBOF NODE 2         │  │
│  │  • SPDK SMA Daemon (gRPC Server)        │  │  • SPDK SMA Daemon (gRPC Server)    │  │
│  │  • SPDK NVMe-oF Target (Poll Mode)      │  │  • SPDK NVMe-oF Target (Poll Mode)  │  │
│  │  • Zero-Copy Direct NVMe PCIe Access    │  │  • Zero-Copy Direct NVMe PCIe Access│  │
│  └────────────────────┬────────────────────┘  └──────────────────┬──────────────────┘  │
│                       ▼                                          ▼                     │
│           [ DUAL-PORT NVMe FLASH POOL ]              [ DUAL-PORT NVMe FLASH POOL ]     │
└────────────────────────────────────────────────────────────────────────────────────────┘

```

---

## 3. Component Interface & Integration Matrix

| Subsystem | Language | Protocol / IPC Interface | Architectural Responsibility |
| :--- | :--- | :--- | :--- |
| **Halmidi Core** | Pure Go (no cgo) | REST (S3), UDS, SMA gRPC, Direct I/O | S3 REST engine, protocol coordination, Distributed Lock Manager (DLM), Canonical ACL engine, aligned buffer pools, background GC/scrubbers. |
| **Omashu DB** | Go (Embedded) | In-Memory BadgerDB + Raft RPC | Linearizable distributed metadata engine. Manages unified inode tables, parent-child hierarchies, S3 bucket/key catalogs, lock leases, and chunk allocation maps. |
| **dotfs Engine** | Pure Go | Internal Storage Driver Interface | Reed-Solomon Erasure Coding ($K+M$), stripe slicing, BLAKE3/CRC32C checksums, block coordinate mapping across NVMe-oF namespaces. |
| **NFS-Ganesha** | C | NFSv3/v4 / Framed UDS (`nfs.sock`) | User-space NFS server delegating filesystem lookups and reads/writes to Halmidi via a low-overhead binary IPC framing protocol over UDS. |
| **Samba** | C | SMB2/SMB3 / Framed UDS (`smb.sock`) | User-space SMB server intercepting Windows handles, oplocks, and byte-range locks, forwarding I/O and lease validations to Halmidi via UDS. |
| **SPDK SMA** | Python / C / Go | gRPC over UDS / WAN (`spdk/sma-goapi`) | Storage Management Agent orchestrating bdev creation, NVMe-oF controller discovery, namespace attachments, QoS caps, and crypto offload. |
| **SPDK Target** | C (User-space) | NVMe-oF (TCP / RoCEv2) | High-performance user-space target polling NVMe queues directly via kernel-bypass drivers, serving remote block requests to compute nodes. |

---

## 4. Functional Requirements

### 4.1 Multi-Protocol Unified Namespace & Operations
* **FR-001 (Unified Namespace):** Any file ingested via NFS or SMB must be instantly queryable and downloadable via S3 without batch reconciliation or re-indexing.
* **FR-002 (AWS S3 REST Compliance):** Comprehensive support for S3 Core APIs: `PutObject`, `GetObject`, `DeleteObject`, `HeadObject`, `ListObjectsV2`, `CopyObject`, and AWS Signature Version 4 (SigV4).
* **FR-003 (S3 Multipart Atomicity):** Parts uploaded concurrently are staged under isolated Omashu metadata prefixes (`.halmidi/multipart/<upload_id>`). Parts commit to the POSIX namespace via a single atomic Raft commit upon `CompleteMultipartUpload`.
* **FR-004 (Thin-Gateway Architecture):** C-based gateways (Ganesha `FSAL_DOTFS` and Samba `vfs_dotfs`) remain thin protocol adaptors, forwarding I/O via framed binary packets across UDS sockets to the Go runtime.

### 4.2 Storage Driver Abstraction & DASE Data Path
* **FR-005 (Storage Driver Interface):** Compute data path must expose a unified Go interface (`ReadBlock`, `WriteBlock`, `Flush`) abstracting Topology A (Hugepage SHM) and Topology B (NVMe-oF Direct I/O).
* **FR-006 (Zero-Heap Allocation Pipeline):** Read/write loops must not allocate on the Go heap. All block slices are checked out from a reusable `sync.Pool` and returned immediately upon I/O completion.
* **FR-007 (4KB Memory Alignment):** Buffer pools must allocate memory via `unix.Mmap` (anonymous private mappings) to guarantee 4096-byte memory alignment required by Linux `O_DIRECT`.
* **FR-008 (Configurable Chunk Striping):** Engine supports configurable block/chunk sizes (4KB for fine-grained random I/O; 64KB to 1MB for high-throughput S3 streaming) mapped to Reed-Solomon stripes.
* **FR-009 (SPDK SMA Orchestration):** Halmidi manages remote storage attachments using pure Go gRPC bindings (`github.com/spdk/sma-goapi`) to dynamically discover, attach, and monitor NVMe namespaces.

### 4.3 Distributed Lock Manager (DLM) & Consistency
* **FR-010 (Raft-Backed Lock Table):** Omashu manages centralized lock states replicated across nodes to prevent split-brain and write collisions.
* **FR-011 (SMB Lease Break Coordination):** If an S3 PUT or NFS write targets a file held by an SMB client with an active write lease, Halmidi dispatches a lease break signal over `smb.sock` and awaits lease downgrade before granting write access.
* **FR-012 (POSIX fcntl Support):** Full support for POSIX advisory byte-range locks across NFS clients.

### 4.4 Data Resiliency, Self-Healing & Cleanup
* **FR-013 (Reed-Solomon Erasure Coding):** `dotfs` enforces configurable $K+M$ profiles (e.g., $4+2$, $8+4$) across distinct physical drives and failure domains.
* **FR-014 (Continuous Bitrot Scrubber):** Background low-priority worker verifies chunk checksums (BLAKE3) at rest and auto-reconstructs corrupted blocks via peer parity chunks.
* **FR-015 (Garbage Collector):** Automated reclamation worker cleans up orphaned multipart uploads past TTL and deletes physical chunks once metadata refcounts reach zero.

---

## 5. Non-Functional Requirements (NFR)

### 5.1 High Availability & Fault Tolerance
* **NFR-001 (Stateless Cattle Failover):** Compute nodes hold zero private persistent data. If a compute node crashes, secondary nodes claim identical block coordinates from Omashu with zero rebuild delay.
* **NFR-002 (Virtual IP Failover):** Compute nodes participate in a Keepalived/VRRP cluster or sit behind an active L4/L7 load balancer. Client reconnection failover must complete in <3 seconds.
* **NFR-003 (Fabric Multipathing):** In Topology B, NVMe-oF initiator must configure multi-path I/O (ANA - Asymmetric Namespace Access) across redundant network fabrics.
* **NFR-004 (Raft Quorum Resiliency):** Omashu maintains linearizable read/write consistency across $2N+1$ consensus nodes, surviving up to $N$ node failures without service disruption.

### 5.2 Performance & Latency SLA Bounds

| Operational Metric | Topology | P95 Bound | P99 Bound |
| :--- | :--- | :--- | :--- |
| **Local 4KB Block Read/Write** | Topology A (Hugepage SHM) | < 15 µs | < 25 µs |
| **Network 4KB Block Read/Write** | Topology B (NVMe-oF TCP) | < 120 µs | < 150 µs |
| **S3 1MB Streaming Throughput** | Topology B (100GbE Fabric) | > 9.2 GB/s per node | Line-rate saturation |
| **Go GC STW Latency Impact** | Universal (`sync.Pool`) | 0% line-rate jitter | < 500 µs max GC pause |

### 5.3 Portability, Security & Observability
* **NFR-005 (Zero-Cgo Build Portability):** Compute binaries must compile with `CGO_ENABLED=0`, deploying cleanly on standard Alpine/Debian containers without C toolchain dependencies.
* **NFR-006 (Network Isolation):** NVMe-oF data traffic must run on dedicated VLANs with MTU 9000 (Jumbo Frames) isolated from management and public gateway networks.
* **NFR-007 (Audit & Telemetry):** Prometheus exporter publishing real-time IOPS, queue depths, EC encode latency, and CEF/JSON audit logs for all file/object operations.

---

## 6. Priority Roadmap & Phased Execution

| Phase | Priority | Core Architectural Deliverables | Exit Criteria & Validation |
| :--- | :--- | :--- | :--- |
| **Phase 1** | **P0** | • Pure Go compute daemon (`CGO_ENABLED=0`).<br>• Omashu Raft + BadgerDB metadata schema.<br>• 4KB-aligned `sync.Pool` memory allocator.<br>• Topology A Hugepage lockless ring buffer.<br>• S3 Core API (PUT, GET, DELETE, List). | Pass AWS S3 Conformance test suite; verify zero-heap allocations via Go pprof; achieve <25µs P99 on local SHM. |
| **Phase 2** | **P0** | • NFS-Ganesha `FSAL_DOTFS` & Samba `vfs_dotfs` over UDS.<br>• SPDK SMA gRPC client integration (`sma-goapi`).<br>• Topology B NVMe-oF Kernel Initiator (`O_DIRECT`).<br>• S3 Multipart Upload staging & atomic commit. | File written via S3 immediately mountable and readable via NFS/SMB; POSIX `pjdfstest` compliance; remote NVMe-oF writes verified. |
| **Phase 3** | **P1** | • DLM Raft lease engine & SMB break dispatch.<br>• Bitrot Scrubber & non-blocking EC rebuild worker.<br>• GC engine (Orphaned multipart & chunk pruning).<br>• High-Availability failover (Keepalived VIP). | Node crash during active multi-protocol write stream with zero data loss; automatic client reconnection in <3s. |
| **Phase 4** | **P2** | • Point-in-Time CoW Snapshots (Windows VSS / `.snapshot`).<br>• S3 Object Versioning.<br>• SMB2 Change Notify & S3 Event Webhooks.<br>• Multi-tenant Quotas, Lifecycle & Prometheus metrics. | Benchmark cluster at 100k+ IOPS; multi-node expansion without client I/O interruption. |
