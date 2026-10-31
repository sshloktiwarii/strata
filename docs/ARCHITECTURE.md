# 🏛️ Fleet Systems Architecture Blueprint
**Projects Covered**: `DETENT` (Chrono) | `ASTROLABE` (3D Spatial AI) | `LATTICE` (CRDT Sync) | `STRATA` (LSM Engine)  
**Classification**: Internal Technical Specification (Local Workspace Only — Do Not Commit / Push)

---

## 🗺️ Fleet System Topology & Port Map

```
┌────────────────────────────────────────────────────────────────────────────────────────┐
│                               FLEET RUNTIME TOPOLOGY                                    │
│                                                                                        │
│   [ DETENT ]            [ ASTROLABE ]             [ LATTICE ]            [ STRATA ]    │
│   Port :3000            Port :3001                Port :4001             Port :5001    │
│   Next.js 15 App        Next.js 15 R3F            Go Distributed         Go Storage    │
│   Worker Delta Clock    3D Spatial Cockpit        CRDT Peer Mesh         LSM-Tree DB   │
│         │                     │                         │                      │       │
│         ▼                     ▼                         ▼                      ▼       │
│   Web Audio Engine      Three.js Orbit Rig        SQLite WAL Storage     CRC32 WAL Log │
│   Knurled Titanium      Telemetry HUD             Vector Clock Mesh      Leveled SSTs  │
│   4-Mode Canvas Shaders Node Reticles             P2P WebSockets         Bloom Filters │
└────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 1. DETENT // Tactile Swiss Chronograph (`fleet-projects/detent`)

### 1.1 Architecture & Pipeline

```
┌────────────────────────────────────────────────────────────────────────┐
│                        DETENT ARCHITECTURE                             │
│                                                                        │
│   +--------------------+       PostMessage        +------------------+ │
│   |   Dedicated Web    | ───────────────────────► |  Main Thread UI  | │
│   |   Worker Thread    |   { type: 'TICK',        |  - React DOM     | │
│   |   (High-Res Delta) |     elapsed, drift }     |  - SVG Bezel     | │
│   +--------------------+                          +------------------+ │
│             │                                               │          │
│             │ performance.now()                             │ Pointer  │
│             ▼                                               ▼ Events   │
│   [ Precision Clock ]                             [ 120-Tick Bezel ]   │
│   - Drift < 5.000ms                               - 6° Detent Math     │
│   - Tab Background Isolation                      - Audio Haptics      │
│                                                             │          │
│                                                             ▼          │
│                                                   [ Web Audio Synth ]  │
│                                                   - Solenoid Clicks    │
│                                                   - Ratchet Detents    │
└────────────────────────────────────────────────────────────────────────┘
```

### 1.2 Core Subsystem Invariants
1. **Delta-Clock Worker (`src/workers/timer.worker.ts`)**:
   - The timing loop runs inside an isolated dedicated Web Worker.
   - Computes elapsed time using differential timestamps:
     $$\Delta t = \text{performance.now()} - t_{\text{last}}$$
   - Prevents timing drift when the operating system or browser throttles background tab execution. Drift ceiling verified under $5.000\text{ ms}$ over 1,000 synthetic ticks.
2. **Knurled Titanium Rotary Dial (`src/components/PrecisionDial.tsx`)**:
   - Captures continuous pointer coordinates via `PointerCapture`.
   - Computes angular deflection relative to dial center:
     $$\theta = \left(\text{atan2}(dy, dx) \cdot \frac{180}{\pi} + 90\right) \pmod{360}$$
   - Quantizes rotation into 60 physical detents of 6° each ($1\text{ detent} = 60\text{ seconds}$ scrub).
   - Generates procedural audio clicks on each detent crossing.
3. **Multi-Engine Generative Canvas (`src/components/EpochAuroraBackground.tsx`)**:
   - 60 FPS HTML5 Canvas engine dynamically running 4 distinct mathematical visualizers:
     - **Mode 1 (Star Atlas Hyperspace)**: 450 depth-projected stars ($x, y, z$) with relativistic warp streaks.
     - **Mode 2 (Lando Norris F1 Telemetry)**: 80 horizontal velocity streaks with volt gradient `#D2FF00` and Silverstone telemetry HUD.
     - **Mode 3 (Make Me Pulse 3D Sphere)**: 180-vertex 3D geodesic sphere with harmonic sine displacement and Champagne Gold links.
     - **Mode 4 (USAF ECHO Sonar Radar)**: 360° rotating radar sweep beam with phosphorescent decay and acoustic oscillogram.
4. **Procedural Web Audio Engine (`src/lib/audio.ts`)**:
   - Generates mechanical sounds synthetically via `AudioContext` without external asset loading.
   - Utilizes custom oscillator chains (square, triangle, noise buffers) and biquad bandpass filtering.

---

## 2. ASTROLABE // 3D Spatial Fleet Orchestrator (`fleet-projects/astrolabe`)

### 2.1 Architecture & Scene Graph

```
┌────────────────────────────────────────────────────────────────────────┐
│                        ASTROLABE ARCHITECTURE                          │
│                                                                        │
│   +----------------------------------------------------------------+   │
│   |              React Three Fiber / Three.js Canvas               |   │
│   |                                                                |   │
│   |   +-------------------+    +-------------------------------+   |   │
│   |   |   Camera Rig      |    |   3D Spatial Node Cluster     |   |   │
│   |   |   - Orbit Controls|    |   - Ollama (Inference Core)   |   |   │
│   |   |   - Dolly Presets |    |   - SemCache (Vector Proxy)   |   |   │
│   |   |   - Damped Inertia|    |   - Sentinel (Telemetry Node) |   |   │
│   |   +-------------------+    |   - AgentVault (Ledger Node)  |   |   │
│   |             │              +-------------------------------+   |   │
│   +-------------┼──────────────────────────────┼───────────────────+   │
│                 │                              │                       │
│                 ▼                              ▼                       │
│   +---------------------------+  Vector3.project   +---------------+   │
│   | Aerospace Cockpit HUD     | ◄───────────────── | Particle Flow |   │
│   | - Target Reticles         |                    | - Real-time   │   │
│   | - Real-time Token Rates   |                    |   Stream Lines|   │
│   +---------------------------+                    +---------------+   │
└────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Core Subsystem Invariants
1. **Scene Graph & Orbital Rig (`src/components/canvas/`)**:
   - Built on React Three Fiber (`@react-three/fiber`) and Three.js.
   - Camera kinematics managed by `CameraRig.tsx` with smooth spherical coordinate interpolation ($r, \theta, \phi$) between 4 tactical presets (Overview, Inference Cluster, Cache Subsystem, Telemetry Matrix).
2. **Dynamic 3D Particle Streams (`ParticleStream.tsx`)**:
   - Uses instanced buffer geometries to animate packet flows along quadratic Bezier curves between interconnected cluster nodes.
   - Flow velocity and particle density scale dynamically based on measured token generation rates.
3. **Aerospace HUD Projection (`src/components/hud/`)**:
   - Projects 3D node coordinates onto the 2D viewport plane using Three.js projection:
     $$\mathbf{p}_{\text{screen}} = \mathbf{p}_{\text{world}} \cdot \mathbf{M}_{\text{view}} \cdot \mathbf{M}_{\text{proj}}$$
   - Draws target-lock brackets, azimuth angles, and hardware telemetry directly aligned with the 3D meshes.
4. **Local Cluster Adapters (`src/lib/constants.ts`)**:
   - Integrates with local runtime ports: Ollama (`:11434`), SemCache (`:8080`), Sentinel (`:9090`), and AgentVault (`:8000`).

---

## 3. LATTICE // Conflict-Free Replication Engine (`fleet-projects/lattice`)

### 3.1 Architecture & Synchronization Flow

```
┌────────────────────────────────────────────────────────────────────────┐
│                         LATTICE ARCHITECTURE                           │
│                                                                        │
│   +--------------------+                   +-----------------------+   │
│   |  CRDT Memory State | ◄─── CDC Hook ──► |  Pure Go SQLite WAL   |   │
│   |  - LWW-Map         |                   |  - Append-Only Log    |   │
│   |  - PNCounter       |                   |  - PRAGMA journal=WAL |   │
│   |  - ORSet           |                   |  - Atomic Mutation Tx |   │
│   +--------------------+                   +-----------------------+   │
│             │                                          │               │
│             ▼                                          ▼               │
│   +--------------------+                   +-----------------------+   │
│   | Vector Clock Engine|                   | P2P WebSocket Mesh    |   │
│   | - Causality Check  | ◄───────────────► | - Peer Discovery      |   │
│   | - Lamport Tiebreak |                   | - Delta Sync Stream   |   │
│   +--------------------+                   | - Split-Brain Chaos   |   │
│                                            +-----------------------+   │
└────────────────────────────────────────────────────────────────────────┘
```

### 3.2 Core Subsystem Invariants
1. **Mathematical Join-Semilattice Model**:
   - Every CRDT state mutation forms a join-semilattice ordered by $\le$ with a least upper bound merge operation $\sqcup$:
     $$\text{Commutativity: } A \sqcup B = B \sqcup A$$
     $$\text{Associativity: } (A \sqcup B) \sqcup C = A \sqcup (B \sqcup C)$$
     $$\text{Idempotence: } A \sqcup A = A$$
2. **Causal Ordering & Vector Clocks (`pkg/crdt/`)**:
   - Each node maintains a vector clock $V$. Given vector clocks $V_A$ and $V_B$:
     $$V_A \le V_B \iff \forall k, V_A[k] \le V_B[k]$$
   - If neither $V_A \le V_B$ nor $V_B \le V_A$, the updates are concurrent, and conflict resolution defaults to Lamport physical timestamps with deterministic node UUID comparison.
3. **Pure Go SQLite Storage (`pkg/storage/`)**:
   - Powered by `modernc.org/sqlite` without CGO dependencies.
   - Operates in Write-Ahead Logging mode (`PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL;`).
4. **P2P WebSocket Mesh & Partition Simulator (`pkg/network/`, `pkg/simulation/`)**:
   - Maintains full-duplex WebSocket peer connections across arbitrary ports (`:4001`, `:4002`, etc.).
   - Includes a deterministic split-brain chaos manager capable of isolating node clusters and verifying state convergence upon re-connection.

---

## 4. STRATA // Leveled Storage Engine (`fleet-projects/strata`)

### 4.1 Architecture & Compaction Hierarchy

```
┌────────────────────────────────────────────────────────────────────────┐
│                          STRATA ARCHITECTURE                           │
│                                                                        │
│                       [ Incoming Write Request ]                       │
│                                    │                                   │
│                    ┌───────────────┴───────────────┐                   │
│                    ▼                               ▼                   │
│          +-------------------+           +-------------------+         │
│          |  Write-Ahead Log  |           | In-Memory MemTable|         │
│          |  (CRC32 Checksum) |           |  (SkipList Index) |         │
│          +-------------------+           +-------------------+         │
│                                                    │                   │
│                                                    ▼ 2MB Flush         │
│                                          +-------------------+         │
│                                          |  Level 0 SSTables |         │
│                                          | (Overlapping Keys)|         │
│                                          +-------------------+         │
│                                                    │                   │
│                                                    ▼ Leveled Merge     │
│                                          +-------------------+         │
│                                          |  Level 1 SSTables |         │
│                                          | (Non-overlapping) |         │
│                                          +-------------------+         │
│                                                    │                   │
│                                                    ▼ Multi-Way K-Merge │
│                                          +-------------------+         │
│                                          |  Level 2+ Storage |         │
│                                          | (Archived Blocks) |         │
│                                          +-------------------+         │
└────────────────────────────────────────────────────────────────────────┘
```

### 4.2 Core Subsystem Invariants
1. **In-Memory SkipList MemTable (`pkg/skiplist/`)**:
   - Probabilistic SkipList providing $O(\log n)$ search, insert, and ordered iteration.
   - Synchronized with `sync.RWMutex` for concurrent multi-reader access.
   - Automatic flush triggered when active memory reaches the byte threshold ($2\text{MB}$).
2. **CRC32-Castagnoli WAL Frame (`pkg/wal/`)**:
   - Every write is serialized to an append-only WAL before MemTable insertion:
     $$\text{Frame} = [\text{CRC32}: 4\text{B}][\text{Type}: 1\text{B}][\text{KeyLen}: 4\text{B}][\text{ValLen}: 4\text{B}][\text{Key}][\text{Value}]$$
   - Uses Castagnoli polynomial `0x82F63B78`. On startup recovery, scans sequentially and automatically truncates the log at the first corrupted or partial byte offset.
3. **Immutable Binary SSTable Format (`pkg/sstable/`)**:
   - `[Data Block 0] ... [Data Block N] [Sparse Index Block] [Bloom Filter Block] [Trailer Footer: 48B]`
   - Sparse index entries are written every $4\text{KB}$, enabling single-seek binary search.
4. **Murmur3 Bit-Vector Bloom Filter (`pkg/bloom/`)**:
   - Configured with 7 optimal hash functions, yielding a $<1\%$ false positive rate.
   - Eliminates 99% of unnecessary disk seeks on key misses.
5. **Leveled Compaction Worker (`pkg/compactor/`)**:
   - Background worker merges overlapping $L0$ tables into partitioned, non-overlapping $L1$ tables.
   - Purges tombstoned deletions once older generation references are reconciled.

