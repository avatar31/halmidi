## SPDK Usage

## 1.1 Purpose
This document defines the functional and non-functional requirements for a cloud-native, high-performance Disaggregated Shared-Everything (DASE) storage engine written in pure Go. The architecture must natively abstract the underlying physical location of storage, supporting both local shared-memory transfers and remote network fabric routing under a unified application interface.
## 1.2 Business Objectives

* Unified Codebase Execution: Ensure developers can build, run, and scale the application locally on a single development machine using shared memory, or deploy it across a vast distributed network without altering the core application engine.
* Sub-Millisecond Performance Consistency: Guarantee deterministic, microsecond-level P99 latency bounds across both topologies by isolating data path interactions from Go’s runtime memory management.
* Zero Cgo Complexity: Maintain a strict no-cgo policy on the compute side to ensure standard compilation chains, clean stack traces, and complete environment portability.

------------------------------
## 2. Dual-Topology System Architecture Overview## Topology A: Single-Node (Shared Memory / Local Deployment)

+-------------------------------------------------------------------------+

| SINGLE PHYSICAL BARE-METAL SERVER / KUBERNETES POD                      |
|                                                                         |
|  +--------------------------+             +--------------------------+  |
|  |  GO COMPUTE APP (NO CGO) |             |       SPDK PROCESS       |  |
|  |  - Control: Unix Socket  |             |  - SMA Daemon Server     |  |
|  |  - Data: Ring Buffer /   |             |  - Local NVMe Controller |  |
|  +--------------+-----------+             +-----------+--------------+  |
|                 |                                     |                 |
|                 +-----------------+-------------------+                 |
|                                   |                                     |
|                                   v                                     |
|                     [ ZERO-COPY LINUX HUGEPAGES ]                       |
+-------------------------------------------------------------------------+

## Topology B: Multi-Node (Disaggregated Fabric / Distributed Deployment)

+------------------------------------------+

| COMPUTE NODE A (Pure Go Client Node)     |
|  - Control: Remote gRPC Client           |
|  - Data: Kernel NVMe-oF/TCP Initiator    |
+-------------------+----------------------+
                    |
                    | (Control: gRPC over WAN / Data: NVMe-oF over TCP Fabric)
                    v
+------------------------------------------+

| STORAGE NODE B (Disaggregated Pool)      |
|  - Control: SPDK SMA Daemon (gRPC)       |
|  - Data: SPDK NVMe-oF Target (Poll Mode) |
+------------------------------------------+

------------------------------
## 3. Detailed Component Requirements## 3.1 Control Plane & Deployment Abstraction
The control plane must hide whether storage is local or remote by introducing a clean Storage Driver Abstraction Interface.

* CP-001: Automated Transport Discovery
* Requirement: The Go application must evaluate environmental flags or a local YAML configuration at startup to determine its operational topology (LOCAL_SHM or REMOTE_FABRIC).
* CP-002: Topology A Setup (Local Shared Memory)
* Requirement: Under LOCAL_SHM mode, the Go application must issue configuration steps via local UNIX domain sockets to tell the SPDK instance to map a slice of local Linux Hugepages (2MB or 1GB configurations) shared by both processes.
* CP-003: Topology B Setup (Remote Fabric)
* Requirement: Under REMOTE_FABRIC mode, the Go application must use standard gRPC to connect to the remote SPDK Storage Management Agent (SMA) daemon, authenticate, provision a logical storage segment, and execute an OS-level nvme connect command to map the network subsystem locally as a block device (e.g., /dev/nvme0n1).

------------------------------
## 3.2 Data Plane (The Hot Path)
The data path layer must move data at raw hardware speeds without allocating objects on the Go heap, protecting the application from GC latency pauses.

* DP-001: Zero-Heap Allocation Pipeline (sync.Pool)
* Requirement: For both local and remote topologies, the hot path read/write loops must never use raw, un-pooled allocations (like make([]byte, 4096) inside a function body). All 4KB blocks must be checked out from a reusable sync.Pool object pool and immediately returned upon completion.
   * Rationale: Bypassing active Go heap allocations keeps the heap size static, completely preventing the Go runtime from triggering Stop-The-World (STW) garbage collection pauses mid-flight.
* DP-002: Topology A Data Flow (Local Lockless Ring Buffer)
* Requirement: In local mode, writing 4KB of data must skip network layers entirely. The Go worker must drop its pre-allocated 4KB memory buffer directly into the shared Hugepage memory segment and update a lockless Ring Buffer pointer index using fast atomic CPU operations. The polling SPDK thread reads it instantly.
* DP-003: Topology B Data Flow (Direct I/O)
* Requirement: In remote fabric mode, the Go application must read/write to the kernel-mapped network NVMe block device using the os.O_DIRECT file execution flag.
   * Rationale: Bypassing the Linux page cache avoids expensive OS data copying and context-switching overhead, streaming the 4KB data blocks directly onto the physical 25/100 GbE network card.
* DP-004: Global Metadata & Locking Coordination
* Requirement: In a multi-node shared-everything cluster, block mapping configurations (which node owns which physical 4KB block ranges) cannot sit inside local memory. The application must interact with a centralized, ultra-low-latency distributed coordination tier (e.g., a clustered distributed Key-Value registry or non-volatile Storage Class Memory) to execute global atomics and handle concurrency conflicts.

------------------------------
## 4. Non-Functional Requirements## 4.1 Latency & Performance SLA Metrics

| Operational Topology | Performance Metric | P95 Latency Bound | P99 Latency Bound |
|---|---|---|---|
| Topology A (Shared Memory) | Local 4KB Block Read/Write | < 15 Microseconds | < 25 Microseconds |
| Topology B (NVMe-oF/TCP) | Network 4KB Block Read/Write | < 120 Microseconds | < 150 Microseconds |


* NFR-001 Go GC Isolation: Across both deployment models, Go runtime garbage collection runs must not introduce execution pauses or background caching interference that skews storage data plane latency by more than 10% of basic hardware line-rate specs.

## 4.2 Portability & Elastic Scaling

* NFR-002 Cattle Architecture: All compute applications must remain completely stateless. If a Compute Node running in a multi-node fabric cluster crashes, any secondary compute node must be able to instantly mount and claim the exact same block coordinate mappings from the shared registry without triggering a data-rebuild phase.
* NFR-003 Environment Portability: The Go client application must build cleanly inside containerization environments (like Docker or standard Linux Alpine targets) using CGO_ENABLED=0, with no dependency on local C cross-compilers or system header architectures.

------------------------------
## 5. Deployment & Runtime Constraints

* ENV-001 Configuration Requirements: Hosts running the local shared memory topology must have a minimum allocation of 2GB of Linux Hugepages pre-configured and mounted at system startup (typically via /dev/hugepages).
* ENV-002 Isolation Mapping: In production data center networks running the Multi-Node model, high-speed NVMe-oF TCP block networks must run on dedicated network interfaces and isolated storage VLANs to prevent management gRPC control signals or general web traffic from introducing performance jitters on the data plane.
