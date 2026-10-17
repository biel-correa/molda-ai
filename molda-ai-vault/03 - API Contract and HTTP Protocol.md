# 03 - API Contract and HTTP Protocol

The integration surface of MoldaAí is centered around a single, high-performance synchronous RESTful HTTP endpoint designed for seamless client-to-server interaction.

---

## 1. Endpoint Definition

* **Route:** `/api/generate`
* **HTTP Method:** `POST`
* **Content-Type:** `multipart/form-data`
* **Payload Limit:** 4 MB maximum image upload size (larger bodies rejected with `400 Bad Request`)

---

## 2. Input Parameters (Form Fields)

| Field | Type | Required? | Default | Min | Max | Description |
| :--- | :--- | :---: | :---: | :---: | :---: | :--- |
| `image` | `file` | **Yes** | — | — | 4 MB | Binary image file (PNG, JPEG, WebP, SVG, GIF). |
| `size_mm` | `float64` | No | `70.0` | `20.0` | `200.0` | Longest bounding box axis of the cutter in mm (X or Y). |
| `blade_thickness_mm` | `float64` | No | `0.8` | `0.4` | `5.0` | Wall thickness of cutting edge. Ideal for 2 perimeters on a 0.4mm nozzle. |
| `blade_height_mm` | `float64` | No | `12.0` | `4.0` | `50.0` | Total height of cutter in mm (top cutting elevation in Z). |
| `base_thickness_mm` | `float64` | No | `3.2` | `1.0` | `15.0` | Support rim width for finger comfort (mm). |
| `base_height_mm` | `float64` | No | `3.0` | `1.0` | `20.0` | Support rim height / bottom base reinforcement (mm). |
| `smoothing_mm` | `float64` | No | `0.5` | `0.05` | `3.0` | Contour smoothing and simplification tolerance in mm. |
| `threshold` | `int` | No | `128` | `1` | `254` | Grayscale luminance threshold (1 to 254). |
| `invert` | `string` | No | `false` | — | — | When `"true"`, `"1"`, or `"yes"`, inverts solid vs empty pixels. |

### Cross-Validation Invariants:
1. `base_height_mm` must be strictly less than `blade_height_mm`.
2. `base_thickness_mm` must be strictly greater than `blade_thickness_mm`.

---

## 3. HTTP Responses

### 3.1 Success (200 OK)
The response body contains the raw binary STL file generated entirely in memory.

* **Response Headers:**
  ```http
  HTTP/1.1 200 OK
  Content-Type: application/sla
  Content-Disposition: attachment; filename="cutter-moldaai.stl"
  Vary: Accept-Encoding
  Content-Length: <bytes>
  ```

#### Adaptive Gzip Compression:
When the client request includes the `Accept-Encoding: gzip` header, the MoldaAí server compresses the binary STL in memory on the fly:
```http
Content-Encoding: gzip
```
* **Efficiency:** Binary STL meshes typically achieve a 70% to 80% compression ratio. This ensures even high-poly designs stay well under the AWS Lambda 6MB synchronous response limit.

---

### 3.2 Error Responses

All error responses return a standardized JSON structure with `Content-Type: application/json`:
```json
{
  "error": "Detailed description of the validation or processing failure in English."
}
```

* **`400 Bad Request`:**
  * Missing `image` file field.
  * Corrupted image data or unsupported image format.
  * Numerical parameter out of allowed bounds.
  * Invariant violations (e.g. `base_height_mm >= blade_height_mm`).
* **`405 Method Not Allowed`:**
  * Attempting to call `/api/generate` with HTTP verbs other than `POST`.
* **`413 Request Entity Too Large`:**
  * Upload payload exceeds the 4 MB ceiling.
* **`422 Unprocessable Entity`:**
  * The thresholded image has no solid pixels or is completely solid (no closed contour can be extracted).

---

## 4. Integration Code Examples

### cURL CLI Example
```bash
curl -X POST http://localhost:8080/api/generate \
  -H "Accept-Encoding: gzip" \
  -F "image=@my-silhouette.png" \
  -F "size_mm=75.0" \
  -F "blade_thickness_mm=0.8" \
  -F "blade_height_mm=12.0" \
  -F "base_thickness_mm=3.2" \
  -F "base_height_mm=3.0" \
  -F "smoothing_mm=0.5" \
  -F "threshold=128" \
  --output cutter.stl
```

### Modern JavaScript (Browser Fetch API)
```javascript
const formData = new FormData();
formData.append('image', imageBlob, 'silhouette.png');
formData.append('size_mm', 70.0);
formData.append('blade_thickness_mm', 0.8);
formData.append('blade_height_mm', 12.0);
formData.append('base_thickness_mm', 3.2);
formData.append('base_height_mm', 3.0);
formData.append('smoothing_mm', 0.5);
formData.append('threshold', 128);

const response = await fetch('/api/generate', {
  method: 'POST',
  body: formData
});

if (!response.ok) {
  const err = await response.json();
  throw new Error(err.error);
}

const arrayBuffer = await response.arrayBuffer();
// Ready to pass to Three.js STLLoader or save as a Blob
const stlBlob = new Blob([arrayBuffer], { type: 'application/sla' });
```
