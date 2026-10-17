package api

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"molda-ai/internal/generator"
)

const maxUploadSize = 4 << 20 // 4MB

// SetupRouter initializes the Gin engine with all routes and static file handlers.
func SetupRouter(webFS fs.FS) *gin.Engine {
	// Set Gin mode based on environment
	if osMode := strings.ToLower(gin.Mode()); osMode == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Endpoint POST /api/generate
	router.POST("/api/generate", GenerateSTLHandler)

	// Static frontend assets served from embedded FS
	if webFS != nil {
		router.NoRoute(gin.WrapH(http.FileServer(http.FS(webFS))))
	}

	return router
}

// GenerateSTLHandler handles POST /api/generate requests using Gin.
func GenerateSTLHandler(c *gin.Context) {
	// Limit request body size to 4MB (+ overhead for multipart headers)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize+1024*1024)

	fileHeader, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'image' not found in multipart form."})
		return
	}

	if fileHeader.Size > maxUploadSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Image file exceeds maximum allowed size of 4MB."})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to open uploaded image: %v", err)})
		return
	}
	defer file.Close()

	// Parse parameters with defaults and bounds
	params := generator.DefaultCutterParams()

	if val := c.PostForm("size_mm"); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 20.0 && f <= 200.0 {
			params.SizeMM = f
		} else if err != nil || f < 20.0 || f > 200.0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'size_mm' parameter. Must be between 20.0 and 200.0 mm."})
			return
		}
	}

	if val := c.PostForm("blade_thickness_mm"); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0.4 && f <= 5.0 {
			params.BladeThicknessMM = f
		} else if err != nil || f < 0.4 || f > 5.0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'blade_thickness_mm' parameter. Must be between 0.4 and 5.0 mm."})
			return
		}
	}

	if val := c.PostForm("blade_height_mm"); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 4.0 && f <= 50.0 {
			params.BladeHeightMM = f
		} else if err != nil || f < 4.0 || f > 50.0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'blade_height_mm' parameter. Must be between 4.0 and 50.0 mm."})
			return
		}
	}

	if val := c.PostForm("base_thickness_mm"); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 1.0 && f <= 15.0 {
			params.BaseThicknessMM = f
		} else if err != nil || f < 1.0 || f > 15.0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'base_thickness_mm' parameter. Must be between 1.0 and 15.0 mm."})
			return
		}
	}

	if val := c.PostForm("base_height_mm"); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 1.0 && f <= 20.0 {
			params.BaseHeightMM = f
		} else if err != nil || f < 1.0 || f > 20.0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'base_height_mm' parameter. Must be between 1.0 and 20.0 mm."})
			return
		}
	}

	if val := c.PostForm("threshold"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n >= 1 && n <= 254 {
			params.Threshold = n
		} else if err != nil || n < 1 || n > 254 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'threshold' parameter. Must be between 1 and 254."})
			return
		}
	}

	if val := c.PostForm("invert"); val != "" {
		params.Invert = val == "true" || val == "1" || val == "yes"
	}

	if val := c.PostForm("smoothing_mm"); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0.05 && f <= 3.0 {
			params.SmoothingMM = f
		} else if err != nil || f < 0.05 || f > 3.0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'smoothing_mm' parameter. Must be between 0.05 and 3.0 mm."})
			return
		}
	} else if val := c.PostForm("smoothing"); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0.05 && f <= 3.0 {
			params.SmoothingMM = f
		}
	}

	// Cross-field validation
	if params.BaseHeightMM >= params.BladeHeightMM {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Base height (%.1f mm) must be strictly less than blade height (%.1f mm).", params.BaseHeightMM, params.BladeHeightMM)})
		return
	}

	if params.BaseThicknessMM <= params.BladeThicknessMM {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Base thickness (%.1f mm) must be greater than blade thickness (%.1f mm).", params.BaseThicknessMM, params.BladeThicknessMM)})
		return
	}

	// Generate STL
	stlBytes, err := generator.GenerateSTL(file, params)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": fmt.Sprintf("Failed to generate 3D model: %v", err)})
		return
	}

	// Response Headers
	c.Header("Content-Type", "application/sla")
	c.Header("Content-Disposition", `attachment; filename="molde-moldaai.stl"`)
	c.Header("Vary", "Accept-Encoding")

	// Check if client accepts gzip compression
	if strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
		c.Header("Content-Encoding", "gzip")
		var gzBuf bytes.Buffer
		gzWriter := gzip.NewWriter(&gzBuf)
		if _, err := gzWriter.Write(stlBytes); err == nil {
			if err := gzWriter.Close(); err == nil {
				c.Data(http.StatusOK, "application/sla", gzBuf.Bytes())
				return
			}
		}
	}

	// If no gzip requested, return raw binary STL bytes
	c.Data(http.StatusOK, "application/sla", stlBytes)
}
