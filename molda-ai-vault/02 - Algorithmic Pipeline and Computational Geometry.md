# 02 - Algorithmic Pipeline and Computational Geometry

The core of **MoldaAí** is a sequential geometric pipeline that converts a 2D matrix (pixels/vector paths) into a closed 3D polyhedral mesh (*watertight 2-manifold*) ready for additive manufacturing slicers (Cura, PrusaSlicer, Bambu Studio, OrcaSlicer).

---

## 1. Pipeline Overview

```mermaid
graph LR
    A["Input Image (PNG, JPG, WebP, SVG)"] --> B["Luminance & Thresholding"]
    B --> C["Marching Squares (2D Contours)"]
    C --> D["Smoothing & Metric Scaling"]
    D --> E["Hole Detection & Offsets"]
    E --> F["Bridging Support Beams"]
    F --> G["Earcut Triangulation (Caps)"]
    G --> H["Vertical Quad Extrusion (Walls)"]
    H --> I["Little-Endian Binary STL Output"]
```

---

## 2. Step-by-Step Mathematical & Algorithmic Breakdown

### Step 1: Multi-Format Image Decoding & Binarization (`image.go`)

1. **Multi-Format Image Support:**
   * **Raster Formats:** Pure Go decoders for PNG (`image/png`), JPEG (`image/jpeg`), GIF (`image/gif`), and WebP (`golang.org/x/image/webp`).
   * **Vector Format (SVG):** SVG parsing and rasterization powered by pure Go libraries `github.com/srwiley/oksvg` and `rasterx`. Vector paths and sharp icons are rasterized cleanly to a high-resolution buffer before thresholding.
2. **Alpha Transparency Handling:**
   * Pixels with alpha transparency where $A < 128$ (< 50% opacity) are classified as background (`false`). This ensures transparent PNGs and cutout silhouettes are processed accurately.
3. **Perceptual Luminance Formula (ITU-R BT.601 Standard):**
   $$Y = 0.299 \times R + 0.587 \times G + 0.114 \times B$$
4. **Binary Thresholding:**
   * If $Y < \text{threshold}$, the pixel is classified as solid (`true`).
   * If $Y \ge \text{threshold}$, the pixel is classified as background (`false`).
   * If the `invert` parameter is enabled, the boolean classification is inverted.
5. **Protective Downscaling:** If an uploaded image exceeds `1000px` on its largest axis, it is proportionally downscaled on the server to maintain sub-second processing.

---

### Step 2: 2D Contour Extraction via Marching Squares (`vector.go`)

To convert the discrete boolean grid into continuous vector polygons:
1. **Grid Padding:** The boolean matrix is enclosed in a 1-pixel border of `false`. This guarantees that silhouettes touching image borders form closed boundary loops.
2. **Marching Squares:** Iterates over each $2 \times 2$ cell configuration:
   $$\text{code} = (\text{TL} \ll 3) \mid (\text{TR} \ll 2) \mid (\text{BR} \ll 1) \mid \text{BL}$$
   * Produces directed line segments across cell edge midpoints.
   * Directs segments consistently, ensuring the solid interior remains on the left side of the displacement vector.
3. **Loop Assembly:** A directed topological graph connects segments $A \to B$ until the terminal coordinate meets the origin coordinate, producing closed polygon rings.
4. **Primary Contour Selection:** Evaluates the absolute polygon area and designates the largest ring as the outer shell ($P_{\text{orig}}$). Remaining closed rings are evaluated as candidate interior holes.

---

### Step 3: Metric Scaling, Inversion and Smoothing (`vector.go`)

1. **Y-Axis Inversion:** In 2D computer graphics, the Y-axis points downwards; in 3D Cartesian coordinates (STL), the Y-axis points upwards/north. Coordinates are reflected along Y to preserve the visual orientation of the input art.
2. **Metric Scaling to Millimeters:** The largest axis of the polygon ($\max(\Delta X, \Delta Y)$) is scaled to match `size_mm` (e.g. $70.0 \text{ mm}$):
   $$\text{scale} = \frac{\text{size\_mm}}{\max(\text{width}, \text{height})}$$
3. **Centering at Origin:** The polygon bounding box center is translated to coordinate $(X=0, Y=0)$.
4. **Laplacian Smoothing Filter:**
   $$P_i' = 0.25 P_{i-1} + 0.5 P_i + 0.25 P_{i+1}$$
   Smoothing passes eliminate raster staircase artifacts without shrinking feature bounds.
5. **Visvalingam-Whyatt Simplification:** Simplifies redundant collinear vertices using the `smoothing_mm` parameter, fusing pixel steps into smooth, straight segments while reducing STL triangle counts by up to 80%.

---

### Step 4: Concentric Polygon Offsets (`vector.go`)

The cutter geometry requires 3 concentric boundary rings for the outer wall:
1. **$P_{\text{orig}}$ (Original Contour):** Defines the dough cutting boundary.
2. **$P_{\text{blade}}$ (Blade Contour):** Expanded outward from $P_{\text{orig}}$ by `bladeThicknessMM` (e.g. $+0.8 \text{ mm}$).
3. **$P_{\text{base}}$ (Support Rim Contour):** Expanded outward from $P_{\text{orig}}$ by `baseThicknessMM` (e.g. $+3.2 \text{ mm}$).

---

### Step 5: Triangulation via Earcut (`mesh.go`)

To transform planar horizontal ring regions into triangle faces:
1. **Winding Order Rules:**
   * Outer contour ($P_{\text{outer}}$) is normalized to Counter-Clockwise (CCW).
   * Inner contour ($P_{\text{inner}}$, acting as hole boundary) is normalized to Clockwise (CW).
2. **Earcut Algorithm:**
   * Triangulates the 2D planar annulus with arbitrary hole counts using `github.com/rclancey/earcut`.
   * Returns facet vertex index triplets.

---

### Step 6: Watertight Manifold 3D Extrusion (`mesh.go`)

The 3D model is constructed without internal intersecting geometry or hidden coplanar faces, producing a single, clean watertight solid:

```text
               Z = bladeHeight (e.g., 12.0 mm)
               ┌──────────┐ <--- Cutting Edge (Top of Blade)
               │  Blade   │
Z = baseHeight ├──────────┴──────────┐ <--- Support Step / Finger Rest
(e.g., 3.0 mm) │                     │
               │   Reinforced Base   │
       Z = 0   └─────────────────────┘ <--- Bottom Rim (Print Bed Surface)
               P_orig     P_blade    P_base
```

1. **Bottom Rim Cap ($Z = 0$):**
   * Horizontal annulus between $P_{\text{base}}$ and $P_{\text{orig}}$.
   * Normal points downward $(0, 0, -1)$ — vertices reversed ($V_1, V_3, V_2$).
2. **Outer Base Wall ($Z \in [0, \text{baseHeight}]$):**
   * Extrusion of $P_{\text{base}}$ quad segments with outward-facing normals.
3. **Base Step / Finger Rest Cap ($Z = \text{baseHeight}$):**
   * Horizontal annulus between $P_{\text{base}}$ and $P_{\text{blade}}$.
   * Normal points upward $(0, 0, 1)$ — standard CCW order ($V_1, V_2, V_3$).
4. **Outer Blade Wall ($Z \in [\text{baseHeight}, \text{bladeHeight}]$):**
   * Extrusion of $P_{\text{blade}}$ with outward-facing normals.
5. **Top Cutting Cap ($Z = \text{bladeHeight}$):**
   * Horizontal cutting edge between $P_{\text{blade}}$ and $P_{\text{orig}}$.
   * Normal points upward $(0, 0, 1)$ — blade wall thickness equals `bladeThicknessMM`.
6. **Inner Cutting Wall ($Z \in [0, \text{bladeHeight}]$):**
   * Extrusion of $P_{\text{orig}}$ from $Z=0$ to $Z=\text{bladeHeight}$.
   * Normal points into the center void.

---

### Step 7: Internal Holes & Structural Connecting Bridges

For geometries with internal negative spaces (such as letters 'A', 'B', 'O', 'D'):
1. **Hole Identification:** Polygons fully enclosed within $P_{\text{orig}}$ are identified using ray-casting point-in-polygon tests.
2. **Inverted Offset Direction:** Because dough is *outside* the hole, cutter walls must expand into the **INTERIOR** of the hole:
   $$H_{\text{blade}} = H_{\text{orig}}.\text{Offset}(-\text{bladeThicknessMM})$$
   $$H_{\text{base}} = H_{\text{orig}}.\text{Offset}(-\text{baseThicknessMM})$$
3. **Solid Floor for Narrow Holes:** If a hole is too small to fit a hollow base ring without self-intersecting, the hole base is automatically capped solid from $Z=0$ to $Z=\text{baseHeightMM}$.
4. **Structural Support Bridges:** To prevent the internal hole cutter from floating disconnected during 3D printing, horizontal support beams are generated connecting the hole base to the outer cutter base:
   * Positioned along $Z \in [0, \text{baseHeightMM}]$ (well below the $12\text{ mm}$ cutting edge).
   * Beam width calibrated to base thickness ($2.5\text{ mm}$ to $3.0\text{ mm}$).
   * Ensures the resulting STL prints as a single monolithic body.

---

### Step 8: Little-Endian Binary STL Serialization (`mesh.go`)

The binary STL payload strictly complies with standard additive manufacturing specifications:

| Offset (Bytes) | Length | Data Type | Value |
| :--- | :--- | :--- | :--- |
| `0` .. `79` | 80 bytes | ASCII / Bytes | Header: `"MoldaAi Binary STL - Single-Binary 3D Cutter Generator"` |
| `80` .. `83` | 4 bytes | `uint32` (Little Endian) | Total triangle count ($N$) |
| `84` onward | $N \times 50$ bytes | Triangle Structs | Exactly 50 bytes per triangle facet |

#### Triangle Struct Layout (50 Bytes):
* **Normal Vector:** $3 \times \text{float32}$ (Little Endian) = 12 bytes
* **Vertex 1:** $3 \times \text{float32}$ (Little Endian) = 12 bytes
* **Vertex 2:** $3 \times \text{float32}$ (Little Endian) = 12 bytes
* **Vertex 3:** $3 \times \text{float32}$ (Little Endian) = 12 bytes
* **Attribute Byte Count:** $\text{uint16}$ (Little Endian, constant `0`) = 2 bytes
