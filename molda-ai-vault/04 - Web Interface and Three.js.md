# 04 - Web Interface and Three.js

The user interface of **MoldaAí** is implemented as a lightweight Single Page Application (SPA) built with modern Vanilla JavaScript (ES Modules), TailwindCSS via CDN, and Three.js r160.

---

## 1. Design System & Visual Identity

* **Domain & Audience:** Cookie decoration, confectionery art, and cold ceramic/polymer clay crafting (earrings, jewelry, pottery stamps).
* **Color Palette:**
  * **Terracotta / Clay:** Warm accents `#C85A32` and `#B44723` for primary calls to action, evoking warm clay, ceramics, and 3D printer PLA filament.
  * **Sand / Cream:** Warm neutral backgrounds `#FAF7F2` and `#F3EFEA` providing high contrast while reducing eye fatigue.
  * **Dark Slate:** Text colors `#43372D` and `#232433` meeting WCAG AAA accessibility contrast ratios.

---

## 2. Client-Side Pre-processing Pipeline

Direct transmission of raw high-resolution images (smartphone cameras frequently produce 12MP to 48MP photos weighing 5MB to 20MB) would impose significant network latency and violate the strict 6MB synchronous payload ceiling of **AWS Lambda**.

MoldaAí mitigates this through client-side HTML5 canvas pre-processing:

```mermaid
flowchart TD
    User["User selects image (e.g. 4000x3000px)"] --> Hidden["Hidden HTML5 Canvas"]
    Hidden --> Scale["Proportional Downscale (max 800px)"]
    Scale --> PrevCanvas["Live 2D B&W Preview"]
    PrevCanvas --> UserFeedback["User tweaks Threshold / Invert / Smoothing"]
    PrevCanvas -->|Export via toBlob('image/png')| BlobReady["Compact Blob (< 100 KB)"]
    BlobReady -->|Fast Upload| Server["MoldaAí API (POST /api/generate)"]
```

1. **Automatic Proportional Downscaling:** Upon selecting an image, it is drawn to a hidden `<canvas>` scaled down so that its longest axis does not exceed `800px`.
2. **Instant 2D Binarization Preview:**
   * The user immediately inspects a 2D canvas displaying the exact silhouette cutlines.
   * Adjusting the **Contrast Threshold** slider (or toggling **Invert Colors**) repaints the 2D canvas dynamically via `getImageData` and `putImageData` without contacting the server.
3. **Ultra-Fast Upload Payload:** When "Generate 3D Cutter" is clicked, the canvas is exported via `toBlob('image/png')` producing a compressed PNG under 100 KB.

---

## 3. Bilingual Support (i18n)

The interface natively provides live dual-language support:
* **Languages:** English (`en`) and Brazilian Portuguese (`pt-BR`).
* **Switching Mechanism:** A header button group toggles between `🇺🇸 EN` and `🇧🇷 PT`.
* **State Persistence:** Saved to `localStorage` under key `moldaai_lang`.
* **Auto-Detection:** Automatically defaults to the browser's language (`navigator.language`) if no preference is explicitly stored.
* **Full Localization:** All UI cards, inputs, preset labels, smoothing hints, dynamic error messages, 3D statistics (triangle counts), and download file names (`cutter-moldaai.stl` vs `molde-moldaai.stl`) update instantly without reloading the page.

---

## 4. Interactive 3D Viewport (Three.js)

### 4.1 Scene Setup & Architecture
* **Engine:** Three.js `0.160.0` loaded via native browser `importmap` without bundlers.
* **Core Components:**
  * `THREE.Scene`: Dark slate background (`#232433`) providing contrast for terracotta materials.
  * `THREE.PerspectiveCamera`: 45° field of view positioned at an isometric angle.
  * `OrbitControls`: Full 360° rotation, smooth mouse wheel zooming, and right-click panning. Clamped at `maxPolarAngle = 90°` to prevent the camera from clipping below the print bed.
  * `GridHelper`: A $160 \times 160 \text{ mm}$ orthogonal build plate with $5 \text{ mm}$ increments, accurately representing standard 3D printer beds.
  * **Three-Point Lighting:** Soft `AmbientLight` paired with two directional lights to highlight chamfers, steps, and cutting walls.

### 4.2 Material & Geometry Ingestion
* The binary STL stream is decoded using `STLLoader.parse(arrayBuffer)`.
* Vertex normals are computed via `computeVertexNormals()`.
* The cutter is centered at $(0, 0)$ and its bottom base is elevated so it rests on top of the print bed ($Z = 0$).
* **PBR Material (`MeshStandardMaterial`):**
  * Color: `#C85A32` (PLA Terracotta)
  * Roughness: `0.38`
  * Metalness: `0.08`

---

## 5. In-Memory Instant Download

Once the binary response arrives from the server, the `ArrayBuffer` is wrapped in a `Blob`:

```javascript
const arrayBuffer = await response.arrayBuffer();
currentStlBlob = new Blob([arrayBuffer], { type: 'application/sla' });
```

When the user clicks **"Download .STL File"**:
* A transient anchor link is triggered with `URL.createObjectURL(currentStlBlob)`.
* **Zero extra server requests:** The download completes instantly from RAM with zero additional bandwidth overhead.
