# ⚖️ Fleet Architectural Decision Records (ADR)
**Projects Covered**: `DETENT` (Chrono) | `ASTROLABE` (3D Spatial AI) | `LATTICE` (CRDT Sync) | `STRATA` (LSM Engine)  
**Classification**: Internal Technical Specification (Local Workspace Only — Do Not Commit / Push)

---

## Index of Architectural Decisions

| ADR ID | Decision Title | Subsystem Impacted | Status |
| :--- | :--- | :--- | :---: |
| **ADR-001** | Dedicated Web Worker Delta Clock vs. `setInterval` | `DETENT` | **ACCEPTED** |
| **ADR-002** | Synthetic Procedural Web Audio vs. Static Audio Assets | `DETENT` | **ACCEPTED** |
| **ADR-003** | 60 FPS Mathematical Canvas Engines vs. 2D CSS Blur Gradients | `DETENT` | **ACCEPTED** |
| **ADR-004** | React Three Fiber / WebGL 2.0 vs. 2D DOM Node Graph | `ASTROLABE` | **ACCEPTED** |
| **ADR-005** | Viewport Projection HUD Reticle vs. DOM Overlay Cards | `ASTROLABE` | **ACCEPTED** |
| **ADR-006** | State-Based & Delta CRDT vs. Operation-Based Replication | `LATTICE` | **ACCEPTED** |
| **ADR-007** | Pure Go SQLite in WAL Mode vs. Embedded BoltDB/Badger | `LATTICE` | **ACCEPTED** |
| **ADR-008** | Probabilistic SkipList vs. Red-Black Tree for MemTable | `STRATA` | **ACCEPTED** |
| **ADR-009** | CRC32-Castagnoli Polynomial vs. Standard IEEE 802.3 for WAL | `STRATA` | **ACCEPTED** |
| **ADR-010** | Leveled Compaction ($L0 \rightarrow L1 \rightarrow L2$) vs. Size-Tiered | `STRATA` | **ACCEPTED** |

---

## ADR-001: Dedicated Web Worker Delta Clock vs. `setInterval`

### Context
Browsers throttle `setTimeout` and `setInterval` on inactive or backgrounded tabs down to 1 execution per second (or suspend execution completely) to preserve battery. Traditional focus timers drift by minutes or hours when backgrounded.

### Decision
Run the core timing clock on a dedicated Web Worker thread. Compute elapsed time strictly using differential high-resolution timestamps against `performance.now()`. Dispatch tick updates to the React UI via `postMessage`.

### Consequences
* **Positive**: Clock drift is mathematically capped under $5.000\text{ ms}$ over 1,000 synthetic ticks. Timer remains accurate regardless of browser tab backgrounding.
* **Negative**: Requires serializing timer state through the Web Worker message boundary.

---

## ADR-002: Synthetic Procedural Web Audio vs. Static Audio Assets

### Context
Audio sound effects (clicks, tick detents, relay chimes) normally require fetching `.mp3` or `.wav` files over HTTP. This introduces network latency, CORS security errors, and offline playback failures.

### Decision
Implement an in-memory procedural synthesizer using the native `AudioContext` Web Audio API. Construct square, saw, and noise oscillator graphs with envelope gain curves and biquad filtering to synthesize mechanical relay and ratchet clicks on demand.

### Consequences
* **Positive**: Zero external network requests, zero CORS errors, sub-10ms response time, 100% offline reliability.
* **Negative**: Sound design must be coded mathematically rather than recorded from physical microphones.

---

## ADR-003: 60 FPS Mathematical Canvas Engines vs. 2D CSS Blur Gradients

### Context
Initial visual tests using three blurry radial-gradient circles (`filter: blur(60px)`) produced a generic, low-effort aesthetic that failed the luxury agency benchmark.

### Decision
Eliminate all 2D blur filters. Implement a 60 FPS Canvas rendering loop executing 4 distinct mathematical graphics engines derived from award-winning reference sites:
1. Star Atlas 3D Relativistic Starfield ($z$-projection).
2. Lando Norris F1 Telemetry & Velocity Stream.
3. Make Me Pulse 3D Geodesic Spherical Mesh.
4. USAF E.C.H.O. Acoustic Sonar Radar Sweep.

### Consequences
* **Positive**: Crisp vector lines, high-contrast HUD coordinates, true 3D spatial depth, and zero aesthetic "vibe-coding".
* **Negative**: Requires managing frame animation loops and canvas resize coordinate transforms.

---

## ADR-004: React Three Fiber / WebGL 2.0 vs. 2D DOM Node Graph

### Context
Visualizing local inference cluster microservices (Ollama, SemCache, Sentinel, AgentVault) using standard HTML/SVG nodes limits visual feedback and cannot effectively display high-density concurrent token flows.

### Decision
Build the topology workbench in WebGL 2.0 using React Three Fiber (`@react-three/fiber`) and Three.js. Render nodes as 3D instanced meshes in an orbital coordinate space with volumetric lighting, fog, and particle curves.

### Consequences
* **Positive**: Hardware-accelerated 60 FPS performance, arbitrary camera pitch/yaw orbital choreography, cinematic lighting.
* **Negative**: Increases initial bundle size and requires WebGL 2.0 hardware context support.

---

## ADR-005: Viewport Projection HUD Reticle vs. DOM Overlay Cards

### Context
Displaying node labels and telemetry metrics in a fixed HTML sidebar disconnects the user's attention from the 3D spatial model.

### Decision
Project 3D world coordinates of selected nodes to 2D screen coordinates using `Vector3.project(camera)`. Render dynamic tactical corner brackets, azimuth angles, and hardware metrics that track directly over the 3D meshes as the camera moves.

### Consequences
* **Positive**: Delivers an authentic aerospace heads-up display (HUD) feel with high spatial cohesion.
* **Negative**: HUD coordinates must be recomputed on every animation frame during camera motion.

---

## ADR-006: State-Based & Delta CRDT vs. Operation-Based Replication

### Context
Distributed peer-to-peer synchronization must support long offline periods, arbitrary network latency, and out-of-order packet delivery without requiring a central consensus coordinator (Raft/Paxos).

### Decision
Implement State-Based and Delta-State Conflict-Free Replicated Data Types (CRDTs). Utilize a join-semilattice algebraic contract where states merge via a least-upper-bound operation $\sqcup$. Order operations using component-wise Vector Clocks and Lamport tie-breaking.

### Consequences
* **Positive**: Mathematically guaranteed eventual convergence (Commutative, Associative, Idempotent). Zero split-brain data corruption.
* **Negative**: State payload sizes can grow over time, necessitating tombstone compaction and delta-state diffing.

---

## ADR-007: Pure Go SQLite in WAL Mode vs. Embedded BoltDB/Badger

### Context
The CRDT sync daemon requires persistent local storage. Standard CGO SQLite bindings require a C compiler, complicating cross-compilation on macOS/Linux. Key-value stores like BoltDB lack rich query capabilities for complex document schemas.

### Decision
Use pure Go SQLite (`modernc.org/sqlite`) configured in Write-Ahead Logging mode (`PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL;`).

### Consequences
* **Positive**: Zero CGO dependencies, single static binary compilation, high-throughput concurrent reads during writes, robust transaction semantics.
* **Negative**: Pure Go SQLite has slightly lower raw single-thread throughput compared to native C SQLite bindings.

---

## ADR-008: Probabilistic SkipList vs. Red-Black Tree for MemTable

### Context
The in-memory MemTable in an LSM storage engine must support fast concurrent lookups, point updates, and ordered sequential range iteration.

### Decision
Implement a probabilistic SkipList. Nodes maintain multi-level forward pointers determined by coin flips ($p = 0.5$, $\text{maxLevel} = 16$). Synchronize concurrent access using `sync.RWMutex`.

### Consequences
* **Positive**: Simpler lock-free and read-write concurrency models compared to rebalancing Red-Black trees. Provides $O(\log n)$ search and insert with trivial range scans.
* **Negative**: Uses slightly more pointer memory per entry than compact binary trees.

---

## ADR-009: CRC32-Castagnoli Polynomial vs. Standard IEEE 802.3 for WAL

### Context
The Write-Ahead Log must detect single-bit, double-bit, and burst transmission errors caused by disk corruption or power loss before replaying mutations.

### Decision
Frame all WAL records with CRC32-Castagnoli checksums using polynomial `0x82F63B78`.

### Consequences
* **Positive**: Guy Castagnoli's polynomial provides mathematically superior Hamming distance error detection over standard IEEE 802.3 CRC32, and leverages native hardware acceleration instructions on modern CPUs.
* **Negative**: None. Standard library `hash/crc32` supports `MakeTable(Castagnoli)` out of the box.

---

## ADR-010: Leveled Compaction ($L0 \rightarrow L1 \rightarrow L2$) vs. Size-Tiered Compaction

### Context
Size-Tiered Compaction (STCS) causes high read amplification because keys can exist across multiple overlapping SSTables in the same tier.

### Decision
Adopt Leveled Compaction (LCS). While $L0$ contains overlapping keys directly flushed from MemTables, all subsequent levels ($L1, L2, \dots$) are strictly partitioned into non-overlapping SSTables. A key exists in at most one SSTable per level.

### Consequences
* **Positive**: Read amplification is bounded to $O(\text{Levels})$ instead of $O(\text{SSTables})$. Space amplification is drastically reduced.
* **Negative**: Compaction worker performs more background write operations (write amplification).

