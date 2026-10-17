# MoldaAí — Knowledge Vault & Technical Documentation

Welcome to the official technical documentation vault for **MoldaAí**. This collection of documents provides exhaustive specifications on architecture, mathematical formulation, computational geometry, software engineering, and the 3D manufacturing pipeline of the MoldaAí ecosystem.

---

## 🗺️ Map of Content (MOC)

1. **[[01 - Architecture and Software Engineering]]**
   * Single-Binary Architectural Overview
   * Hybrid Runtime: Local Gin HTTP vs AWS Lambda Function URLs
   * Zero-Cloud / Zero-DB / In-Memory Only Philosophy
   * Package Hierarchy and Separation of Concerns

2. **[[02 - Algorithmic Pipeline and Computational Geometry]]**
   * Multi-Format Image Decoding (PNG, JPEG, WebP, SVG, GIF)
   * Luminance Conversion & Binary Thresholding (ITU-R BT.601)
   * 2D Contour Extraction via Marching Squares
   * Polygon Smoothing, Douglas-Peucker & Metric Scaling
   * Internal Hole Detection & Inward Offset Calculation
   * Automatic Structural Bridging for Floating Island Features
   * Constrained Delaunay / Earcut Triangulation with Holes
   * Watertight Manifold 3D Extrusion
   * Little-Endian Binary STL Serialization

3. **[[03 - API Contract and HTTP Protocol]]**
   * Endpoint Specification: `POST /api/generate`
   * Strict Parameter Validation and Error Handling
   * Adaptive Gzip Compression and AWS Lambda 6MB Payload Optimization
   * Practical Request Examples (cURL and JavaScript Fetch)

4. **[[04 - Web Interface and Three.js]]**
   * Visual Design System & Palette (Terracotta & Sand Clay)
   * Client-Side Pre-processing (Canvas 2D & 800px Downscaling)
   * Real-Time Reactive Binarization Preview
   * Interactive 3D Viewport (Three.js r160, OrbitControls, STLLoader)
   * In-Memory STL Download without Refetch
   * Bilingual Internationalization (English & Brazilian Portuguese)

5. **[[05 - 3D Printing Parameters and Slicing Guide]]**
   * Preset Dimensions: Small (45mm), Medium (70mm), Large (100mm)
   * Slicer Guidelines (Cura, PrusaSlicer, Bambu Studio, OrcaSlicer)
   * Nozzle Calibration (0.4mm vs 0.2mm) and Perimeter Tuning
   * Food Safety & Hygiene Best Practices (Food-Safe 3D Printing)

6. **[[06 - Compilation and AWS Lambda Deployment Guide]]**
   * Local Development & Execution (`make run`, `make test`)
   * Cross-Compilation for Linux ARM64 Graviton (`provided.al2023`)
   * AWS Lambda Function URLs Setup & Permissions
   * Memory Sizing, Cold Starts, and Production Cost Metrics

7. **[[07 - Quality Assurance, Testing and Manifold Verification]]**
   * Automated Test Coverage (`app/internal/...`)
   * Mathematical Edge-Pairing Verification (Watertight Manifold Check)
   * Performance Benchmarks and RAM Consumption Analysis

---

## 📌 Executive Project Summary

| Attribute | Specification |
| :--- | :--- |
| **Language** | Pure Go (Golang) 1.22+ (Zero CGO) |
| **Supported Runtime** | macOS / Linux / Windows (Local HTTP) & AWS Lambda ARM64 (`provided.al2023`) |
| **Web Framework** | Gin Web Framework (`github.com/gin-gonic/gin`) + `ginadapter.NewV2` |
| **Input Formats** | 2D Raster & Vector: PNG, JPEG, WebP, SVG, GIF (max 4MB) |
| **Output Format** | 3D Watertight Mesh in Binary STL (`application/sla`) |
| **Storage Model** | Zero Disk, Zero Database, Zero S3 (100% In-Memory RAM) |
| **Frontend** | Vanilla JS + TailwindCSS + Three.js r160 embedded via `//go:embed` |
| **Bilingual UI** | English (Default) & Brazilian Portuguese (PT-BR) with live toggle |
| **Latency** | Typical response time < 50ms for end-to-end 3D mesh generation |
