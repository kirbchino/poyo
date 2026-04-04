// Package tools implements media reading capabilities (images, PDFs)
package tools

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ImageReader reads image files
type ImageReader struct {
	supportedFormats map[string]bool
}

// NewImageReader creates a new ImageReader
func NewImageReader() *ImageReader {
	return &ImageReader{
		supportedFormats: map[string]bool{
			".png":  true,
			".jpg":  true,
			".jpeg": true,
			".gif":  true,
			".webp": true,
			".bmp":  true,
		},
	}
}

// CanRead checks if the file is a supported image format
func (r *ImageReader) CanRead(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return r.supportedFormats[ext]
}

// ReadImage reads an image file and returns base64 encoded data
func (r *ImageReader) ReadImage(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read image: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	mimeType := "image/png"
	switch ext {
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".gif":
		mimeType = "image/gif"
	case ".webp":
		mimeType = "image/webp"
	case ".bmp":
		mimeType = "image/bmp"
	}

	return map[string]interface{}{
		"type":        "image",
		"mimeType":    mimeType,
		"data":        base64.StdEncoding.EncodeToString(data),
		"size":        len(data),
		"filename":    filepath.Base(path),
		"description": fmt.Sprintf("📸 Poyo 吸入了图片: %s (%d bytes)", filepath.Base(path), len(data)),
	}, nil
}

// PDFReader reads PDF files
type PDFReader struct {
	// In a real implementation, this would use a PDF library
}

// NewPDFReader creates a new PDFReader
func NewPDFReader() *PDFReader {
	return &PDFReader{}
}

// CanRead checks if the file is a PDF
func (r *PDFReader) CanRead(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".pdf"
}

// ReadPDF reads a PDF file and extracts text
// pages parameter format: "1-5" or "1,3,5" or "all"
func (r *PDFReader) ReadPDF(path string, pages string) (map[string]interface{}, error) {
	// Check file exists
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to access PDF: %w", err)
	}

	// In a real implementation, this would use a PDF parsing library
	// For now, return a placeholder with file info
	return map[string]interface{}{
		"type":        "pdf",
		"path":        path,
		"size":        info.Size(),
		"pages":       pages,
		"filename":    filepath.Base(path),
		"content":     "📄 PDF 内容需要 PDF 解析库支持。在实际实现中，这里会返回提取的文本内容。",
		"description": fmt.Sprintf("📄 Poyo 吸入了 PDF: %s (%d bytes)", filepath.Base(path), info.Size()),
		"note":        "要完全支持 PDF 解析，需要集成 pdf 解析库",
	}, nil
}

// MediaReadTool is a unified tool for reading media files
type MediaReadTool struct {
	BaseTool
	imageReader *ImageReader
	pdfReader   *PDFReader
}

// NewMediaReadTool creates a new MediaReadTool
func NewMediaReadTool() *MediaReadTool {
	return &MediaReadTool{
		BaseTool: BaseTool{
			name:        "MediaRead",
			description: "📸 读取图片和 PDF 文件",
			inputSchema: ToolInputJSONSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"path": {
						"type":        "string",
						"description": "文件路径",
					},
					"type": {
						"type":        "string",
						"description": "媒体类型: image 或 pdf",
						"enum":        []string{"image", "pdf", "auto"},
					},
					"pages": {
						"type":        "string",
						"description": "PDF 页码范围（仅 PDF）: '1-5', '1,3,5', 或 'all'",
					},
				},
				Required: []string{"path"},
			},
			isEnabled:  true,
			isReadOnly: true,
		},
		imageReader: NewImageReader(),
		pdfReader:   NewPDFReader(),
	}
}

// Call executes the MediaRead tool
func (t *MediaReadTool) Call(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext, _ CanUseToolFunc, _ ToolCallProgress) (*ToolResult, error) {
	path, ok := input["path"].(string)
	if !ok || path == "" {
		return nil, fmt.Errorf("path is required")
	}

	readType, _ := input["type"].(string)
	if readType == "" {
		readType = "auto"
	}

	// Auto-detect type
	if readType == "auto" {
		if t.imageReader.CanRead(path) {
			readType = "image"
		} else if t.pdfReader.CanRead(path) {
			readType = "pdf"
		} else {
			return nil, fmt.Errorf("unsupported file type: %s", filepath.Ext(path))
		}
	}

	switch readType {
	case "image":
		result, err := t.imageReader.ReadImage(path)
		if err != nil {
			return nil, err
		}
		return &ToolResult{Data: result}, nil

	case "pdf":
		pages, _ := input["pages"].(string)
		if pages == "" {
			pages = "all"
		}
		result, err := t.pdfReader.ReadPDF(path, pages)
		if err != nil {
			return nil, err
		}
		return &ToolResult{Data: result}, nil

	default:
		return nil, fmt.Errorf("unsupported media type: %s", readType)
	}
}

// InputSchema returns the input schema
func (t *MediaReadTool) InputSchema() ToolInputJSONSchema {
	return t.inputSchema
}

// RegisterMediaTools registers media reading tools
func RegisterMediaTools() {
	RegisterTool(NewMediaReadTool())
}
