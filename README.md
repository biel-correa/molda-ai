# MoldaAí — 3D Cookie & Clay Cutter Generator

A high-performance **Single-Binary** web application written in Go that converts 2D raster and vector artwork (PNG, JPEG, WebP, SVG, GIF) into 3D printable `.STL` (Binary) cutters and stamps for confectionery cookies, pastry fondant, and cold polymer clay crafts.

* **Zero Database, Zero Cloud Storage:** 100% in-memory processing. User designs and files are never persisted to disk or cloud buckets.
* **Single-Binary Architecture:** Static web application (HTML5/TailwindCSS + Three.js r160) embedded directly into the Go executable via `//go:embed`.
* **Hybrid Runtime:** Seamless execution as a local HTTP service (`http://localhost:8080`) powered by Gin or as a serverless microservice on **AWS Lambda** (ARM64 Graviton) using *Lambda Function URLs*.
* **Watertight 2-Manifold Guarantee:** Automated Earcut triangulation, vertical quad extrusion, inward offset calculation for internal holes (e.g. letters 'A', 'O', 'B'), and connecting support beams.
* **Line Smoothing & Antialiasing:** Built-in Laplacian filtering and Visvalingam-Whyatt simplification slider.
* **Bilingual UI:** Native live toggle between **English** and **Brazilian Portuguese (PT-BR)**.

---

## Repository Structure

* **[`app/`](app/):** Complete Go solution source code, embedded web interface, unit tests, and build recipes.
* **[`molda-ai-vault/`](molda-ai-vault/):** Comprehensive technical documentation vault in Markdown (architecture, computational geometry, API contracts, Three.js, 3D printing slicing guide, and AWS Lambda deployment).

---

## Quickstart

```bash
# Run locally on port :8080
make run

# Run unit tests and watertight verification
make test

# Build local standalone executable binary
make build-local

# Package deployment ZIP for AWS Lambda (ARM64 Graviton)
make build-lambda

# Deploy directly to AWS Lambda via AWS CLI
make deploy-lambda
```

Open your browser at: **`http://localhost:8080`**

---

## Automated CI/CD Pipeline (GitHub Actions)

Continuous integration and continuous deployment are fully automated via [`.github/workflows/deploy.yml`](.github/workflows/deploy.yml):
* **On Pull Request:** Automatically runs formatting checks, unit tests, and cross-compilation builds.
* **On Push to `main`:** Tests, cross-compiles for Linux ARM64 Graviton, deploys to **AWS Lambda**, updates Function URLs/CORS, and publishes the live application URL directly to the GitHub step summary.
* Supports **OIDC IAM Role assumption** (recommended) or traditional AWS API access keys.

---

## Documentation Vault (`molda-ai-vault/`)

1. **[[00 - Overview and Index]]** — System architecture summary, MOC, and tech stack overview.
2. **[[01 - Architecture and Software Engineering]]** — Single-binary design, hybrid runtime, and package structure.
3. **[[02 - Algorithmic Pipeline and Computational Geometry]]** — Marching Squares, Laplacian smoothing, holes, offsets, and binary STL serialization.
4. **[[03 - API Contract and HTTP Protocol]]** — `POST /api/generate` specification, validation, and Gzip optimization.
5. **[[04 - Web Interface and Three.js]]** — Client-side downscaling, live B&W preview, Three.js 3D viewport, and bilingual i18n.
6. **[[05 - 3D Printing Parameters and Slicing Guide]]** — Slicer guidelines (Cura, PrusaSlicer, Bambu Studio), food safety, and materials.
7. **[[06 - Compilation and AWS Lambda Deployment Guide]]** — ARM64 compilation, AWS Lambda Function URLs setup, and cost analysis.
8. **[[07 - Quality Assurance, Testing and Manifold Verification]]** — Directed edge-pairing tests and manifold closure proof.
