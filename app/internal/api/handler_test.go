package api

import (
	"bytes"
	"compress/gzip"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func createTestPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 80, 80))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	for y := 20; y < 60; y++ {
		for x := 20; x < 60; x++ {
			img.Set(x, y, color.Black)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func buildMultipartRequest(fields map[string]string, fileField, filename string, fileBytes []byte) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for k, v := range fields {
		_ = writer.WriteField(k, v)
	}

	if fileBytes != nil {
		part, err := writer.CreateFormFile(fileField, filename)
		if err != nil {
			return nil, err
		}
		_, _ = part.Write(fileBytes)
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	req := httptest.NewRequest(http.MethodPost, "/api/generate", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, nil
}

func TestGenerateSTLHandlerSuccess(t *testing.T) {
	router := SetupRouter(nil)

	pngData := createTestPNG()
	fields := map[string]string{
		"size_mm":            "60.0",
		"blade_thickness_mm": "0.8",
		"blade_height_mm":    "12.0",
		"base_thickness_mm":  "3.2",
		"base_height_mm":     "3.0",
		"threshold":          "128",
		"smoothing_mm":       "0.5",
	}

	req, err := buildMultipartRequest(fields, "image", "square.png", pngData)
	if err != nil {
		t.Fatalf("buildMultipartRequest failed: %v", err)
	}
	req.Header.Set("Accept-Encoding", "gzip")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d, body: %s", rr.Code, rr.Body.String())
	}

	if rr.Header().Get("Content-Type") != "application/sla" {
		t.Errorf("unexpected Content-Type: %s", rr.Header().Get("Content-Type"))
	}

	if rr.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("expected Content-Encoding: gzip, got: %s", rr.Header().Get("Content-Encoding"))
	}

	gzReader, err := gzip.NewReader(rr.Body)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	decompressed, err := io.ReadAll(gzReader)
	if err != nil {
		t.Fatalf("failed to decompress STL: %v", err)
	}

	if len(decompressed) < 84 {
		t.Fatalf("decompressed STL size too small: %d", len(decompressed))
	}

	header := string(decompressed[:20])
	if header != "MoldaAi Binary STL -" {
		t.Errorf("incorrect STL header: %s", header)
	}
}

func TestGenerateSTLHandlerMissingImage(t *testing.T) {
	router := SetupRouter(nil)

	req, err := buildMultipartRequest(map[string]string{"size_mm": "50"}, "", "", nil)
	if err != nil {
		t.Fatalf("buildMultipartRequest failed: %v", err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 Bad Request, got: %d", rr.Code)
	}
}

func TestGenerateSTLHandlerInvalidParams(t *testing.T) {
	router := SetupRouter(nil)

	pngData := createTestPNG()
	fields := map[string]string{
		"size_mm":         "500.0", // Out of bounds (>200)
		"blade_height_mm": "5.0",
		"base_height_mm":  "8.0", // base_height > blade_height
	}

	req, err := buildMultipartRequest(fields, "image", "test.png", pngData)
	if err != nil {
		t.Fatalf("buildMultipartRequest failed: %v", err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 Bad Request, got: %d", rr.Code)
	}
}
