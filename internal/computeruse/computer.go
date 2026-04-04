// Package computeruse provides computer use capabilities (screen, mouse, keyboard).
package computeruse

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ActionType represents the type of computer action
type ActionType string

const (
	ActionScreenshot   ActionType = "screenshot"
	ActionMouseClick   ActionType = "mouse_click"
	ActionMouseDoubleClick ActionType = "mouse_double_click"
	ActionMouseRightClick  ActionType = "mouse_right_click"
	ActionMouseMove   ActionType = "mouse_move"
	ActionMouseDrag   ActionType = "mouse_drag"
	ActionMouseScroll ActionType = "mouse_scroll"
	ActionKeyType     ActionType = "key"
	ActionKeyTypeText ActionType = "type_text"
	ActionKeyHotkey   ActionType = "hotkey"
	ActionWait        ActionType = "wait"
)

// MouseButton represents mouse button
type MouseButton string

const (
	MouseLeft   MouseButton = "left"
	MouseRight  MouseButton = "right"
	MouseMiddle MouseButton = "middle"
)

// Coordinate represents screen coordinates
type Coordinate struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Action represents a computer action
type Action struct {
	Type        ActionType  `json:"type"`
	Coordinate  *Coordinate `json:"coordinate,omitempty"`
	CoordinateEnd *Coordinate `json:"coordinate_end,omitempty"`
	Button      MouseButton `json:"button,omitempty"`
	Text        string      `json:"text,omitempty"`
	Keys        []string    `json:"keys,omitempty"`
	ScrollAmount int         `json:"scroll_amount,omitempty"`
	ScrollDirection string    `json:"scroll_direction,omitempty"`
	Duration    int         `json:"duration,omitempty"` // milliseconds
}

// ActionResult represents the result of an action
type ActionResult struct {
	Success     bool        `json:"success"`
	Message     string      `json:"message,omitempty"`
	Screenshot  *Screenshot `json:"screenshot,omitempty"`
	Error       string      `json:"error,omitempty"`
	Duration    int64       `json:"duration_ms"`
}

// Screenshot represents a screenshot
type Screenshot struct {
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Data      string `json:"data"` // Base64 encoded
	Format    string `json:"format"`
	Timestamp time.Time `json:"timestamp"`
}

// ScreenInfo represents screen information
type ScreenInfo struct {
	Width           int `json:"width"`
	Height          int `json:"height"`
	ScalingFactor   float64 `json:"scaling_factor"`
	NumDisplays     int `json:"num_displays"`
	PrimaryDisplay  int `json:"primary_display"`
}

// ComputerUseTool provides computer use capabilities
type ComputerUseTool struct {
	name        string
	description string
	screenInfo  *ScreenInfo
	lastScreenshot *Screenshot
	mu          sync.RWMutex
	config      *Config
}

// Config represents computer use configuration
type Config struct {
	ScreenshotDir    string
	AutoScreenshot   bool // Take screenshot after each action
	DefaultWait      int  // Default wait time in ms
	ScreenScaling    float64
}

// NewComputerUseTool creates a new computer use tool
func NewComputerUseTool(config *Config) *ComputerUseTool {
	if config == nil {
		config = &Config{
			ScreenshotDir:  os.TempDir(),
			AutoScreenshot: true,
			DefaultWait:    100,
			ScreenScaling:  1.0,
		}
	}

	tool := &ComputerUseTool{
		name:        "ComputerUse",
		description: "Control the computer with mouse and keyboard actions",
		config:      config,
	}

	// Detect screen info
	tool.screenInfo = tool.detectScreenInfo()

	return tool
}

// Name returns the tool name
func (t *ComputerUseTool) Name() string {
	return t.name
}

// Description returns the tool description
func (t *ComputerUseTool) Description() string {
	return t.description
}

// InputSchema returns the JSON schema for tool input
func (t *ComputerUseTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        []string{
					"screenshot", "mouse_click", "mouse_double_click", "mouse_right_click",
					"mouse_move", "mouse_drag", "mouse_scroll", "key", "type_text", "hotkey", "wait",
				},
				"description": "The action to perform",
			},
			"coordinate": map[string]interface{}{
				"type":        "object",
				"properties": map[string]interface{}{
					"x": map[string]interface{}{"type": "integer"},
					"y": map[string]interface{}{"type": "integer"},
				},
				"description": "Screen coordinate",
			},
			"coordinate_end": map[string]interface{}{
				"type":        "object",
				"properties": map[string]interface{}{
					"x": map[string]interface{}{"type": "integer"},
					"y": map[string]interface{}{"type": "integer"},
				},
				"description": "End coordinate for drag operations",
			},
			"button": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"left", "right", "middle"},
				"description": "Mouse button",
			},
			"text": map[string]interface{}{
				"type":        "string",
				"description": "Text to type",
			},
			"keys": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Keys for hotkey",
			},
			"scroll_amount": map[string]interface{}{
				"type":        "integer",
				"description": "Scroll amount",
			},
			"scroll_direction": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"up", "down", "left", "right"},
				"description": "Scroll direction",
			},
			"duration": map[string]interface{}{
				"type":        "integer",
				"description": "Duration in milliseconds",
			},
		},
		"required": []string{"action"},
	}
}

// ToolInput represents the parsed tool input
type ToolInput struct {
	Action          ActionType  `json:"action"`
	Coordinate      *Coordinate `json:"coordinate,omitempty"`
	CoordinateEnd   *Coordinate `json:"coordinate_end,omitempty"`
	Button          MouseButton `json:"button,omitempty"`
	Text            string      `json:"text,omitempty"`
	Keys            []string    `json:"keys,omitempty"`
	ScrollAmount    int         `json:"scroll_amount,omitempty"`
	ScrollDirection string      `json:"scroll_direction,omitempty"`
	Duration        int         `json:"duration,omitempty"`
}

// Execute executes the tool
func (t *ComputerUseTool) Execute(ctx context.Context, input []byte) (interface{}, error) {
	var req ToolInput
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	startTime := time.Now()
	var result *ActionResult

	switch req.Action {
	case ActionScreenshot:
		result = t.takeScreenshot()
	case ActionMouseClick:
		result = t.mouseClick(req)
	case ActionMouseDoubleClick:
		result = t.mouseDoubleClick(req)
	case ActionMouseRightClick:
		result = t.mouseRightClick(req)
	case ActionMouseMove:
		result = t.mouseMove(req)
	case ActionMouseDrag:
		result = t.mouseDrag(req)
	case ActionMouseScroll:
		result = t.mouseScroll(req)
	case ActionKeyType:
		result = t.keyPress(req)
	case ActionKeyTypeText:
		result = t.typeText(req)
	case ActionKeyHotkey:
		result = t.hotkey(req)
	case ActionWait:
		result = t.wait(req)
	default:
		return nil, fmt.Errorf("unknown action: %s", req.Action)
	}

	result.Duration = time.Since(startTime).Milliseconds()

	// Auto screenshot after action
	if t.config.AutoScreenshot && req.Action != ActionScreenshot {
		result.Screenshot = t.captureScreen()
	}

	return result, nil
}

// takeScreenshot takes a screenshot
func (t *ComputerUseTool) takeScreenshot() *ActionResult {
	screenshot := t.captureScreen()
	if screenshot == nil {
		return &ActionResult{
			Success: false,
			Error:   "Failed to capture screenshot",
		}
	}

	t.mu.Lock()
	t.lastScreenshot = screenshot
	t.mu.Unlock()

	return &ActionResult{
		Success:    true,
		Message:    fmt.Sprintf("Screenshot captured: %dx%d", screenshot.Width, screenshot.Height),
		Screenshot: screenshot,
	}
}

// captureScreen captures the screen
func (t *ComputerUseTool) captureScreen() *Screenshot {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("screencapture", "-x", "-")
	case "linux":
		cmd = exec.Command("import", "-window", "root", "-")
	case "windows":
		// Windows would need different approach
		return &Screenshot{
			Width:     t.screenInfo.Width,
			Height:    t.screenInfo.Height,
			Data:      "",
			Format:    "png",
			Timestamp: time.Now(),
		}
	default:
		return nil
	}

	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	// Decode image to get dimensions
	img, _, err := image.Decode(strings.NewReader(string(output)))
	if err != nil {
		return nil
	}

	bounds := img.Bounds()

	return &Screenshot{
		Width:     bounds.Dx(),
		Height:    bounds.Dy(),
		Data:      base64.StdEncoding.EncodeToString(output),
		Format:    "png",
		Timestamp: time.Now(),
	}
}

// mouseClick performs a mouse click
func (t *ComputerUseTool) mouseClick(req ToolInput) *ActionResult {
	if req.Coordinate == nil {
		return &ActionResult{Success: false, Error: "coordinate required"}
	}

	button := req.Button
	if button == "" {
		button = MouseLeft
	}

	// Execute mouse click using platform-specific commands
	err := t.executeMouseClick(req.Coordinate, button, 1)
	if err != nil {
		return &ActionResult{Success: false, Error: err.Error()}
	}

	return &ActionResult{
		Success: true,
		Message: fmt.Sprintf("Clicked %s at (%d, %d)", button, req.Coordinate.X, req.Coordinate.Y),
	}
}

// mouseDoubleClick performs a double click
func (t *ComputerUseTool) mouseDoubleClick(req ToolInput) *ActionResult {
	if req.Coordinate == nil {
		return &ActionResult{Success: false, Error: "coordinate required"}
	}

	err := t.executeMouseClick(req.Coordinate, MouseLeft, 2)
	if err != nil {
		return &ActionResult{Success: false, Error: err.Error()}
	}

	return &ActionResult{
		Success: true,
		Message: fmt.Sprintf("Double-clicked at (%d, %d)", req.Coordinate.X, req.Coordinate.Y),
	}
}

// mouseRightClick performs a right click
func (t *ComputerUseTool) mouseRightClick(req ToolInput) *ActionResult {
	if req.Coordinate == nil {
		return &ActionResult{Success: false, Error: "coordinate required"}
	}

	err := t.executeMouseClick(req.Coordinate, MouseRight, 1)
	if err != nil {
		return &ActionResult{Success: false, Error: err.Error()}
	}

	return &ActionResult{
		Success: true,
		Message: fmt.Sprintf("Right-clicked at (%d, %d)", req.Coordinate.X, req.Coordinate.Y),
	}
}

// mouseMove moves the mouse
func (t *ComputerUseTool) mouseMove(req ToolInput) *ActionResult {
	if req.Coordinate == nil {
		return &ActionResult{Success: false, Error: "coordinate required"}
	}

	err := t.executeMouseMove(req.Coordinate)
	if err != nil {
		return &ActionResult{Success: false, Error: err.Error()}
	}

	return &ActionResult{
		Success: true,
		Message: fmt.Sprintf("Moved mouse to (%d, %d)", req.Coordinate.X, req.Coordinate.Y),
	}
}

// mouseDrag performs a drag operation
func (t *ComputerUseTool) mouseDrag(req ToolInput) *ActionResult {
	if req.Coordinate == nil || req.CoordinateEnd == nil {
		return &ActionResult{Success: false, Error: "start and end coordinates required"}
	}

	err := t.executeMouseDrag(req.Coordinate, req.CoordinateEnd)
	if err != nil {
		return &ActionResult{Success: false, Error: err.Error()}
	}

	return &ActionResult{
		Success: true,
		Message: fmt.Sprintf("Dragged from (%d, %d) to (%d, %d)",
			req.Coordinate.X, req.Coordinate.Y,
			req.CoordinateEnd.X, req.CoordinateEnd.Y),
	}
}

// mouseScroll performs a scroll
func (t *ComputerUseTool) mouseScroll(req ToolInput) *ActionResult {
	amount := req.ScrollAmount
	if amount == 0 {
		amount = 3
	}

	direction := req.ScrollDirection
	if direction == "" {
		direction = "down"
	}

	err := t.executeMouseScroll(amount, direction)
	if err != nil {
		return &ActionResult{Success: false, Error: err.Error()}
	}

	return &ActionResult{
		Success: true,
		Message: fmt.Sprintf("Scrolled %s by %d", direction, amount),
	}
}

// keyPress presses a key
func (t *ComputerUseTool) keyPress(req ToolInput) *ActionResult {
	if len(req.Keys) == 0 {
		return &ActionResult{Success: false, Error: "keys required"}
	}

	err := t.executeKeyPress(req.Keys[0])
	if err != nil {
		return &ActionResult{Success: false, Error: err.Error()}
	}

	return &ActionResult{
		Success: true,
		Message: fmt.Sprintf("Pressed key: %s", req.Keys[0]),
	}
}

// typeText types text
func (t *ComputerUseTool) typeText(req ToolInput) *ActionResult {
	if req.Text == "" {
		return &ActionResult{Success: false, Error: "text required"}
	}

	err := t.executeTypeText(req.Text)
	if err != nil {
		return &ActionResult{Success: false, Error: err.Error()}
	}

	return &ActionResult{
		Success: true,
		Message: fmt.Sprintf("Typed text: %s", truncateText(req.Text, 50)),
	}
}

// hotkey presses a hotkey combination
func (t *ComputerUseTool) hotkey(req ToolInput) *ActionResult {
	if len(req.Keys) == 0 {
		return &ActionResult{Success: false, Error: "keys required"}
	}

	err := t.executeHotkey(req.Keys)
	if err != nil {
		return &ActionResult{Success: false, Error: err.Error()}
	}

	return &ActionResult{
		Success: true,
		Message: fmt.Sprintf("Pressed hotkey: %s", strings.Join(req.Keys, "+")),
	}
}

// wait waits for a duration
func (t *ComputerUseTool) wait(req ToolInput) *ActionResult {
	duration := req.Duration
	if duration == 0 {
		duration = t.config.DefaultWait
	}

	time.Sleep(time.Duration(duration) * time.Millisecond)

	return &ActionResult{
		Success: true,
		Message: fmt.Sprintf("Waited %dms", duration),
	}
}

// Platform-specific implementations

func (t *ComputerUseTool) executeMouseClick(coord *Coordinate, button MouseButton, count int) error {
	switch runtime.GOOS {
	case "darwin":
		// Use cliclick or similar on macOS
		// This is a placeholder - actual implementation would use proper tools
		return nil
	case "linux":
		// Use xdotool on Linux
		btn := "1"
		if button == MouseRight {
			btn = "3"
		} else if button == MouseMiddle {
			btn = "2"
		}
		cmd := exec.Command("xdotool", "mousemove", fmt.Sprintf("%d", coord.X), fmt.Sprintf("%d", coord.Y))
		cmd.Run()
		for i := 0; i < count; i++ {
			cmd = exec.Command("xdotool", "click", btn)
			cmd.Run()
		}
		return nil
	case "windows":
		// Use PowerShell on Windows
		return nil
	}
	return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
}

func (t *ComputerUseTool) executeMouseMove(coord *Coordinate) error {
	switch runtime.GOOS {
	case "linux":
		cmd := exec.Command("xdotool", "mousemove", fmt.Sprintf("%d", coord.X), fmt.Sprintf("%d", coord.Y))
		return cmd.Run()
	}
	return nil
}

func (t *ComputerUseTool) executeMouseDrag(start, end *Coordinate) error {
	switch runtime.GOOS {
	case "linux":
		// Move to start, mouse down, move to end, mouse up
		exec.Command("xdotool", "mousemove", fmt.Sprintf("%d", start.X), fmt.Sprintf("%d", start.Y)).Run()
		exec.Command("xdotool", "mousedown", "1").Run()
		exec.Command("xdotool", "mousemove", fmt.Sprintf("%d", end.X), fmt.Sprintf("%d", end.Y)).Run()
		exec.Command("xdotool", "mouseup", "1").Run()
		return nil
	}
	return nil
}

func (t *ComputerUseTool) executeMouseScroll(amount int, direction string) error {
	switch runtime.GOOS {
	case "linux":
		button := "4" // Up
		if direction == "down" {
			button = "5"
		}
		for i := 0; i < amount; i++ {
			exec.Command("xdotool", "click", button).Run()
		}
		return nil
	}
	return nil
}

func (t *ComputerUseTool) executeKeyPress(key string) error {
	switch runtime.GOOS {
	case "linux":
		cmd := exec.Command("xdotool", "key", key)
		return cmd.Run()
	}
	return nil
}

func (t *ComputerUseTool) executeTypeText(text string) error {
	switch runtime.GOOS {
	case "linux":
		cmd := exec.Command("xdotool", "type", "--delay", "50", text)
		return cmd.Run()
	}
	return nil
}

func (t *ComputerUseTool) executeHotkey(keys []string) error {
	switch runtime.GOOS {
	case "linux":
		keyStr := strings.Join(keys, "+")
		cmd := exec.Command("xdotool", "key", keyStr)
		return cmd.Run()
	}
	return nil
}

func (t *ComputerUseTool) detectScreenInfo() *ScreenInfo {
	info := &ScreenInfo{
		Width:         1920,
		Height:        1080,
		ScalingFactor: 1.0,
		NumDisplays:   1,
		PrimaryDisplay: 0,
	}

	switch runtime.GOOS {
	case "linux":
		cmd := exec.Command("xdpyinfo")
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, "dimensions:") {
					// Parse dimensions
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						dim := parts[1]
						dims := strings.Split(dim, "x")
						if len(dims) == 2 {
							fmt.Sscanf(dims[0], "%d", &info.Width)
							fmt.Sscanf(dims[1], "%d", &info.Height)
						}
					}
				}
			}
		}
	}

	return info
}

// GetScreenInfo returns screen information
func (t *ComputerUseTool) GetScreenInfo() *ScreenInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.screenInfo
}

// GetLastScreenshot returns the last captured screenshot
func (t *ComputerUseTool) GetLastScreenshot() *Screenshot {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.lastScreenshot
}

// SaveScreenshot saves a screenshot to a file
func (t *ComputerUseTool) SaveScreenshot(screenshot *Screenshot, path string) error {
	if screenshot == nil || screenshot.Data == "" {
		return fmt.Errorf("no screenshot data")
	}

	data, err := base64.StdEncoding.DecodeString(screenshot.Data)
	if err != nil {
		return fmt.Errorf("failed to decode screenshot: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// Helper function
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// Image utilities

// CreateImage creates an image from screenshot data
func (s *Screenshot) CreateImage() (image.Image, error) {
	if s.Data == "" {
		return nil, fmt.Errorf("no screenshot data")
	}

	data, err := base64.StdEncoding.DecodeString(s.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode: %w", err)
	}

	img, err := png.Decode(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to decode PNG: %w", err)
	}

	return img, nil
}

// CropRegion crops a region from the screenshot
func (s *Screenshot) CropRegion(x, y, width, height int) (*Screenshot, error) {
	img, err := s.CreateImage()
	if err != nil {
		return nil, err
	}

	// Clamp to bounds
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x+width > s.Width {
		width = s.Width - x
	}
	if y+height > s.Height {
		height = s.Height - y
	}

	cropImg := img.(interface {
		SubImage(r image.Rectangle) image.Image
	}).SubImage(image.Rect(x, y, x+width, y+height))

	// Encode back to base64
	var buf strings.Builder
	if err := png.Encode(&buf, cropImg); err != nil {
		return nil, err
	}

	return &Screenshot{
		Width:     width,
		Height:    height,
		Data:      base64.StdEncoding.EncodeToString([]byte(buf.String())),
		Format:    "png",
		Timestamp: time.Now(),
	}, nil
}
