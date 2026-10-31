# 🎨 Fleet Design System & Tactical Ergonomics Contract
**Projects Covered**: `DETENT` (Chrono) | `ASTROLABE` (3D Spatial AI) | `LATTICE` (CRDT Sync) | `STRATA` (LSM Engine)  
**Classification**: Internal Technical Specification (Local Workspace Only — Do Not Commit / Push)

---

## 🏛️ Core Design Philosophy: Swiss Tactical Brutalism

The visual language of the fleet completely eliminates "AI-slop" conventions (generic purple/blue radial gradients, floating rounded cards with fuzzy drop shadows, and meaningless marketing jargon). Instead, it draws inspiration from:
1. **Swiss Horology & Precision Metrology**: Physical milled bezels, knurled metal grips, high-contrast mechanical detent markers.
2. **Declassified Aerospace & Defense Telemetry**: Monospace typography, crosshair target reticles, radar sweep trails, coordinate azimuths.
3. **Architectural Grid Systems**: Exposed 1px border lines, asymmetric bento layouts, high-contrast dark obsidian canvases.

---

## 📐 Typography & Metric Standards

### Font Families
* **Primary Technical & Numerals**: `JetBrains Mono`, `SF Mono`, `Menlo`, `monospace`.
* **Wide Architectural Display**: `TT Mussels`, `Mona Sans Variable`, `Univers Ultra Condensed`.
* **Body / System Text**: System native Swiss font stack (`-apple-system`, `BlinkMacSystemFont`, `sans-serif`).

### Numerical Rules
* All timers, counters, latencies, and token rates **MUST** enforce tabular lining numerals:
  ```css
  font-variant-numeric: tabular-nums;
  ```
* Prevents layout jitter and horizontal width oscillations during rapid numerical updates.

---

## 🎨 Color Token Scale (Obsidian & Tactical Accents)

```
┌────────────────────────────────────────────────────────────────────────┐
│                        TACTICAL COLOR PALETTE                          │
│                                                                        │
│   [ #030508 ]  Obsidian Canvas (Base background)                       │
│   [ #0f172a ]  Milled Titanium (Bezel & component background)          │
│   [ #1e293b ]  Knurled Rim Border (Subtle 1px architectural lines)     │
│   [ #D2FF00 ]  F1 Volt Neon (DETENT active indicator)                  │
│   [ #32feff ]  Cockpit Cyan (ASTROLABE target lock reticle)            │
│   [ #10b981 ]  Join Emerald (LATTICE convergence state)                │
│   [ #f59e0b ]  Hazard Amber (STRATA compaction warning)                │
│   [ #e5d8b2 ]  Champagne Gold (Drôle de Monsieur luxury tone)          │
│   [ #5bbad5 ]  Sonar Blue (USAF E.C.H.O. acoustic sweep)               │
└────────────────────────────────────────────────────────────────────────┘
```

### Contrast Floors (WCAG 2.1 AA / AAA)
* All active text on dark canvases maintains a contrast ratio $> 7:1$ (AAA).
* Monospace telemetry stamps in `#475569` and `#64748b` maintain a contrast ratio $> 4.5:1$ (AA Large).

---

## ⚙️ Micro-Interaction Physics & Physical Dial Mechanics

### 1. The 120-Tick Knurled Rotary Bezel (`DETENT`)
* **Physical Dimension**: 360px SVG viewbox with a 150px primary radius circle.
* **Detent Quantization**: 120 radial laser ticks:
  * Major marks every 30° (10 ticks): 14px length in `#475569`.
  * Medium marks every 15° (5 ticks): 8px length in `#334155`.
  * Minor marks: 4px length in `#1e293b`.
* **Haptic Drag Dynamics**:
  * Pointer capture engaged on `pointerdown`.
  * Polar angle computed dynamically via $\text{atan2}(dy, dx)$.
  * Every 6° threshold crossed triggers `playRatchetClick(detent)` via Web Audio.
  * Releasing the bezel triggers a crisp solenoid relay snap `playRelayClick(0.2)`.

### 2. 3D Spatial Camera Orbit (`ASTROLABE`)
* **Inertia Damping**: Rotational velocity dampened via exponential decay:
  $$\mathbf{v}_{t+1} = \mathbf{v}_t \cdot 0.92$$
* **Field of View**: Clamped at $48^\circ$ with near plane at $0.1$ and far plane at $1000$.
* **Target Lock Transitions**: Smooth camera dolly between nodes over $600\text{ms}$ using a cubic-bezier curve:
  $$\text{cubic-bezier}(0.16, 1, 0.3, 1)$$

---

## 🎛️ Component Specifications & Anatomy

### 1. The Tactical Telemetry Bar
* **Position**: Pinned top header, 100% width, 36px height.
* **Surface**: `bg-black/40 backdrop-blur-2xl border-b border-white/[0.07]`.
* **Telemetry Indicators**:
  * Left: Active visualizer mode with animated pulse indicator.
  * Center: Chronological countdown timer to next 30-minute epoch.
  * Right: Tactical mode cycler pill with immediate shader trigger.

### 2. The 3D Target Reticle Overlay
* **Visuals**: 4 corner brackets offset by 24px from projected node center.
* **Dynamics**: Reticle shrinks from 48px to 32px on node lock with 120ms spring physics.
* **Information Box**: 9px monospace label displaying Node ID, active models, tokens/second, and round-trip latency.

### 3. Split-Brain Network Partition Matrix (`LATTICE`)
* **Interactive Toggles**: Node partition pills allowing users to artificially sever WebSocket connections between peers.
* **Vector Clock Inspector**: Live component-wise array display (`[NodeA: 14, NodeB: 12, NodeC: 8]`) with real-time causal graph updates.

### 4. Leveled Compaction Heatmap (`STRATA`)
* **Layer Tiers**: Visual representation of $L0, L1, L2$ SSTables.
* **Metrics**: Compaction ratio bar, Bloom filter hit/miss gauge, and active MemTable byte threshold meter ($0\text{MB} \rightarrow 2\text{MB}$).

