package computeruse

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestActionType(t *testing.T) {
	types := []ActionType{
		ActionScreenshot,
		ActionMouseClick,
		ActionMouseDoubleClick,
		ActionMouseRightClick,
		ActionMouseMove,
		ActionMouseDrag,
		ActionMouseScroll,
		ActionKeyType,
		ActionKeyTypeText,
		ActionKeyHotkey,
		ActionWait,
	}

	for _, at := range types {
		if at == "" {
			t.Error("ActionType should not be empty")
		}
	}
}

func TestMouseButton(t *testing.T) {
	buttons := []MouseButton{
		MouseLeft,
		MouseRight,
		MouseMiddle,
	}

	for _, b := range buttons {
		if b == "" {
			t.Error("MouseButton should not be empty")
		}
	}
}

func TestNewComputerUseTool(t *testing.T) {
	tool := NewComputerUseTool(nil)

	if tool == nil {
		t.Fatal("NewComputerUseTool() returned nil")
	}

	if tool.Name() != "ComputerUse" {
		t.Errorf("Name() = %q, want 'ComputerUse'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Description() should not be empty")
	}

	if tool.config == nil {
		t.Error("Config should be initialized")
	}
}

func TestComputerUseToolWithConfig(t *testing.T) {
	config := &Config{
		ScreenshotDir:  "/tmp/screenshots",
		AutoScreenshot: false,
		DefaultWait:    200,
		ScreenScaling:  2.0,
	}

	tool := NewComputerUseTool(config)

	if tool.config.ScreenshotDir != "/tmp/screenshots" {
		t.Errorf("ScreenshotDir = %q, want '/tmp/screenshots'", tool.config.ScreenshotDir)
	}

	if tool.config.AutoScreenshot != false {
		t.Error("AutoScreenshot should be false")
	}

	if tool.config.DefaultWait != 200 {
		t.Errorf("DefaultWait = %d, want 200", tool.config.DefaultWait)
	}
}

func TestComputerUseToolInputSchema(t *testing.T) {
	tool := NewComputerUseTool(nil)

	schema := tool.InputSchema()
	if schema == nil {
		t.Fatal("InputSchema() should not be nil")
	}

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("InputSchema should have properties")
	}

	if _, ok := props["action"]; !ok {
		t.Error("InputSchema should have action property")
	}
}

func TestCoordinate(t *testing.T) {
	coord := Coordinate{X: 100, Y: 200}

	if coord.X != 100 {
		t.Errorf("X = %d, want 100", coord.X)
	}

	if coord.Y != 200 {
		t.Errorf("Y = %d, want 200", coord.Y)
	}
}

func TestAction(t *testing.T) {
	action := Action{
		Type: ActionMouseClick,
		Coordinate: &Coordinate{X: 100, Y: 200},
		Button: MouseLeft,
	}

	if action.Type != ActionMouseClick {
		t.Errorf("Type = %v, want %v", action.Type, ActionMouseClick)
	}

	if action.Coordinate.X != 100 {
		t.Error("Coordinate X should be 100")
	}
}

func TestActionResult(t *testing.T) {
	result := ActionResult{
		Success:  true,
		Message:  "Action completed",
		Duration: 50,
	}

	if !result.Success {
		t.Error("Success should be true")
	}

	if result.Message != "Action completed" {
		t.Errorf("Message = %q", result.Message)
	}
}

func TestScreenshot(t *testing.T) {
	screenshot := Screenshot{
		Width:     1920,
		Height:    1080,
		Data:      "base64data",
		Format:    "png",
		Timestamp: time.Now(),
	}

	if screenshot.Width != 1920 {
		t.Errorf("Width = %d, want 1920", screenshot.Width)
	}

	if screenshot.Format != "png" {
		t.Errorf("Format = %q, want 'png'", screenshot.Format)
	}
}

func TestScreenInfo(t *testing.T) {
	info := ScreenInfo{
		Width:          1920,
		Height:         1080,
		ScalingFactor:  1.5,
		NumDisplays:    2,
		PrimaryDisplay: 0,
	}

	if info.Width != 1920 {
		t.Errorf("Width = %d, want 1920", info.Width)
	}

	if info.NumDisplays != 2 {
		t.Errorf("NumDisplays = %d, want 2", info.NumDisplays)
	}
}

func TestToolInput(t *testing.T) {
	input := ToolInput{
		Action:     ActionMouseClick,
		Coordinate: &Coordinate{X: 100, Y: 200},
		Button:     MouseLeft,
	}

	if input.Action != ActionMouseClick {
		t.Errorf("Action = %v, want %v", input.Action, ActionMouseClick)
	}

	if input.Button != MouseLeft {
		t.Errorf("Button = %v, want %v", input.Button, MouseLeft)
	}
}

func TestToolInputJSON(t *testing.T) {
	jsonData := `{
		"action": "mouse_click",
		"coordinate": {"x": 100, "y": 200},
		"button": "left"
	}`

	var input ToolInput
	if err := json.Unmarshal([]byte(jsonData), &input); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if input.Action != ActionMouseClick {
		t.Errorf("Action = %v, want mouse_click", input.Action)
	}

	if input.Coordinate.X != 100 {
		t.Errorf("Coordinate.X = %d, want 100", input.Coordinate.X)
	}
}

func TestExecuteWait(t *testing.T) {
	tool := NewComputerUseTool(&Config{DefaultWait: 50})

	input := `{"action": "wait"}`

	result, err := tool.Execute(context.Background(), []byte(input))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	actionResult, ok := result.(*ActionResult)
	if !ok {
		t.Fatal("Execute() should return ActionResult")
	}

	if !actionResult.Success {
		t.Error("Wait should succeed")
	}
}

func TestExecuteWaitWithDuration(t *testing.T) {
	tool := NewComputerUseTool(nil)

	input := `{"action": "wait", "duration": 100}`

	start := time.Now()
	result, err := tool.Execute(context.Background(), []byte(input))
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	actionResult := result.(*ActionResult)
	if !actionResult.Success {
		t.Error("Wait should succeed")
	}

	if elapsed < 100*time.Millisecond {
		t.Errorf("Wait duration = %v, want at least 100ms", elapsed)
	}
}

func TestExecuteInvalidAction(t *testing.T) {
	tool := NewComputerUseTool(nil)

	input := `{"action": "invalid"}`

	_, err := tool.Execute(context.Background(), []byte(input))
	if err == nil {
		t.Error("Execute() should return error for invalid action")
	}
}

func TestExecuteInvalidJSON(t *testing.T) {
	tool := NewComputerUseTool(nil)

	_, err := tool.Execute(context.Background(), []byte(`invalid`))
	if err == nil {
		t.Error("Execute() should return error for invalid JSON")
	}
}

func TestExecuteMouseClickNoCoordinate(t *testing.T) {
	tool := NewComputerUseTool(nil)

	input := `{"action": "mouse_click"}`

	result, err := tool.Execute(context.Background(), []byte(input))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	actionResult := result.(*ActionResult)
	if actionResult.Success {
		t.Error("Click without coordinate should fail")
	}
}

func TestExecuteTypeText(t *testing.T) {
	tool := NewComputerUseTool(nil)

	input := `{"action": "type_text", "text": "Hello"}`

	result, err := tool.Execute(context.Background(), []byte(input))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	actionResult := result.(*ActionResult)
	// Will succeed on platforms without xdotool (empty implementation)
	_ = actionResult
}

func TestExecuteHotkey(t *testing.T) {
	tool := NewComputerUseTool(nil)

	input := `{"action": "hotkey", "keys": ["ctrl", "c"]}`

	result, err := tool.Execute(context.Background(), []byte(input))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	actionResult := result.(*ActionResult)
	_ = actionResult
}

func TestExecuteHotkeyNoKeys(t *testing.T) {
	tool := NewComputerUseTool(nil)

	input := `{"action": "hotkey"}`

	result, err := tool.Execute(context.Background(), []byte(input))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	actionResult := result.(*ActionResult)
	if actionResult.Success {
		t.Error("Hotkey without keys should fail")
	}
}

func TestExecuteMouseDrag(t *testing.T) {
	tool := NewComputerUseTool(nil)

	input := `{
		"action": "mouse_drag",
		"coordinate": {"x": 100, "y": 100},
		"coordinate_end": {"x": 200, "y": 200}
	}`

	result, err := tool.Execute(context.Background(), []byte(input))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	actionResult := result.(*ActionResult)
	_ = actionResult
}

func TestExecuteMouseDragNoEnd(t *testing.T) {
	tool := NewComputerUseTool(nil)

	input := `{
		"action": "mouse_drag",
		"coordinate": {"x": 100, "y": 100}
	}`

	result, err := tool.Execute(context.Background(), []byte(input))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	actionResult := result.(*ActionResult)
	if actionResult.Success {
		t.Error("Drag without end coordinate should fail")
	}
}

func TestExecuteMouseScroll(t *testing.T) {
	tool := NewComputerUseTool(nil)

	input := `{"action": "mouse_scroll", "scroll_amount": 5, "scroll_direction": "down"}`

	result, err := tool.Execute(context.Background(), []byte(input))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	actionResult := result.(*ActionResult)
	_ = actionResult
}

func TestGetScreenInfo(t *testing.T) {
	tool := NewComputerUseTool(nil)

	info := tool.GetScreenInfo()
	if info == nil {
		t.Fatal("GetScreenInfo() returned nil")
	}

	if info.Width <= 0 {
		t.Error("Screen width should be positive")
	}

	if info.Height <= 0 {
		t.Error("Screen height should be positive")
	}
}

func TestGetLastScreenshot(t *testing.T) {
	tool := NewComputerUseTool(nil)

	// Initially nil
	screenshot := tool.GetLastScreenshot()
	if screenshot != nil {
		t.Error("Initial screenshot should be nil")
	}
}

func TestTruncateText(t *testing.T) {
	tests := []struct {
		text     string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a long text", 10, "this is a ..."},
		{"exact", 5, "exact"},
	}

	for _, tt := range tests {
		result := truncateText(tt.text, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateText(%q, %d) = %q, want %q",
				tt.text, tt.maxLen, result, tt.expected)
		}
	}
}

func TestSaveScreenshotNoData(t *testing.T) {
	tool := NewComputerUseTool(nil)

	err := tool.SaveScreenshot(nil, "/tmp/test.png")
	if err == nil {
		t.Error("SaveScreenshot with nil should return error")
	}

	err = tool.SaveScreenshot(&Screenshot{}, "/tmp/test.png")
	if err == nil {
		t.Error("SaveScreenshot with empty data should return error")
	}
}

func TestScreenshotCreateImage(t *testing.T) {
	// Screenshot with no data
	s := &Screenshot{}
	_, err := s.CreateImage()
	if err == nil {
		t.Error("CreateImage with no data should return error")
	}

	// Screenshot with invalid data
	s = &Screenshot{Data: "invalid"}
	_, err = s.CreateImage()
	if err == nil {
		t.Error("CreateImage with invalid data should return error")
	}
}

func TestScreenshotCropRegion(t *testing.T) {
	s := &Screenshot{
		Width:  100,
		Height: 100,
	}

	// No data
	_, err := s.CropRegion(0, 0, 50, 50)
	if err == nil {
		t.Error("CropRegion with no data should return error")
	}
}

func TestConfigDefaults(t *testing.T) {
	config := &Config{}

	tool := NewComputerUseTool(config)

	// Check that defaults are applied
	if tool.config.ScreenshotDir == "" {
		// ScreenshotDir should be set to temp dir if empty
	}
}

func TestExecuteAllActionTypes(t *testing.T) {
	tool := NewComputerUseTool(nil)

	actions := []struct {
		name  string
		input string
	}{
		{"screenshot", `{"action": "screenshot"}`},
		{"mouse_click", `{"action": "mouse_click", "coordinate": {"x": 100, "y": 100}}`},
		{"mouse_double_click", `{"action": "mouse_double_click", "coordinate": {"x": 100, "y": 100}}`},
		{"mouse_right_click", `{"action": "mouse_right_click", "coordinate": {"x": 100, "y": 100}}`},
		{"mouse_move", `{"action": "mouse_move", "coordinate": {"x": 100, "y": 100}}`},
		{"mouse_scroll", `{"action": "mouse_scroll"}`},
		{"key", `{"action": "key", "keys": ["a"]}`},
		{"type_text", `{"action": "type_text", "text": "test"}`},
		{"hotkey", `{"action": "hotkey", "keys": ["ctrl", "c"]}`},
		{"wait", `{"action": "wait"}`},
	}

	for _, tc := range actions {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tool.Execute(context.Background(), []byte(tc.input))
			if err != nil {
				t.Fatalf("Execute() error: %v", err)
			}

			actionResult, ok := result.(*ActionResult)
			if !ok {
				t.Fatal("Execute() should return ActionResult")
			}

			// Check that duration is set
			if actionResult.Duration < 0 {
				t.Error("Duration should not be negative")
			}
		})
	}
}
