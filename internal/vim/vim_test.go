package vim

import (
	"testing"
)

func TestMode(t *testing.T) {
	modes := []Mode{
		ModeNormal,
		ModeInsert,
		ModeVisual,
		ModeVisualLine,
		ModeVisualBlock,
		ModeCommand,
		ModeReplace,
	}

	for _, mode := range modes {
		if mode == "" {
			t.Error("Mode should not be empty")
		}
	}
}

func TestModeString(t *testing.T) {
	tests := []struct {
		mode     Mode
		expected string
	}{
		{ModeNormal, "NORMAL"},
		{ModeInsert, "INSERT"},
		{ModeVisual, "VISUAL"},
		{ModeVisualLine, "VISUAL LINE"},
		{ModeVisualBlock, "VISUAL BLOCK"},
		{ModeCommand, "COMMAND"},
		{ModeReplace, "REPLACE"},
	}

	for _, tt := range tests {
		result := tt.mode.String()
		if result != tt.expected {
			t.Errorf("Mode(%q).String() = %q, want %q", tt.mode, result, tt.expected)
		}
	}
}

func TestModeIsInsertMode(t *testing.T) {
	tests := []struct {
		mode     Mode
		expected bool
	}{
		{ModeNormal, false},
		{ModeInsert, true},
		{ModeReplace, true},
		{ModeVisual, false},
	}

	for _, tt := range tests {
		result := tt.mode.IsInsertMode()
		if result != tt.expected {
			t.Errorf("Mode(%q).IsInsertMode() = %v, want %v", tt.mode, result, tt.expected)
		}
	}
}

func TestModeIsVisualMode(t *testing.T) {
	tests := []struct {
		mode     Mode
		expected bool
	}{
		{ModeNormal, false},
		{ModeVisual, true},
		{ModeVisualLine, true},
		{ModeVisualBlock, true},
		{ModeInsert, false},
	}

	for _, tt := range tests {
		result := tt.mode.IsVisualMode()
		if result != tt.expected {
			t.Errorf("Mode(%q).IsVisualMode() = %v, want %v", tt.mode, result, tt.expected)
		}
	}
}

func TestNewEngine(t *testing.T) {
	e := NewEngine()
	if e == nil {
		t.Fatal("NewEngine() returned nil")
	}

	if e.GetMode() != ModeNormal {
		t.Errorf("Initial mode = %v, want %v", e.GetMode(), ModeNormal)
	}

	if e.keymaps == nil {
		t.Error("Keymaps should be initialized")
	}

	if e.state.Registers == nil {
		t.Error("Registers should be initialized")
	}
}

func TestEngineSetMode(t *testing.T) {
	e := NewEngine()

	e.SetMode(ModeInsert)
	if e.GetMode() != ModeInsert {
		t.Errorf("Mode = %v, want %v", e.GetMode(), ModeInsert)
	}

	e.SetMode(ModeNormal)
	if e.GetMode() != ModeNormal {
		t.Errorf("Mode = %v, want %v", e.GetMode(), ModeNormal)
	}
}

func TestEngineSetModeVisual(t *testing.T) {
	e := NewEngine()

	// Enter visual mode
	e.SetMode(ModeVisual)
	if !e.HasSelection() {
		t.Error("Visual mode should have selection")
	}

	// Exit visual mode
	e.SetMode(ModeNormal)
	if e.HasSelection() {
		t.Error("Normal mode should not have selection")
	}
}

func TestEngineCursor(t *testing.T) {
	e := NewEngine()

	cursor := e.GetCursor()
	if cursor.Line != 0 || cursor.Column != 0 {
		t.Error("Initial cursor should be at (0, 0)")
	}

	e.SetCursor(Position{Line: 5, Column: 10})
	cursor = e.GetCursor()
	if cursor.Line != 5 || cursor.Column != 10 {
		t.Errorf("Cursor = (%d, %d), want (5, 10)", cursor.Line, cursor.Column)
	}
}

func TestEngineMoveCursor(t *testing.T) {
	e := NewEngine()

	// Move right
	e.MoveCursor(MotionRight, 1)
	cursor := e.GetCursor()
	if cursor.Column != 1 {
		t.Errorf("Column = %d, want 1", cursor.Column)
	}

	// Move down
	e.MoveCursor(MotionDown, 1)
	cursor = e.GetCursor()
	if cursor.Line != 1 {
		t.Errorf("Line = %d, want 1", cursor.Line)
	}

	// Move left
	e.MoveCursor(MotionLeft, 1)
	cursor = e.GetCursor()
	if cursor.Column != 0 {
		t.Errorf("Column = %d, want 0", cursor.Column)
	}

	// Move with count
	e.MoveCursor(MotionRight, 5)
	cursor = e.GetCursor()
	if cursor.Column != 5 {
		t.Errorf("Column = %d, want 5", cursor.Column)
	}
}

func TestEngineMoveCursorBounds(t *testing.T) {
	e := NewEngine()

	// Move left at start should stay at 0
	e.MoveCursor(MotionLeft, 1)
	cursor := e.GetCursor()
	if cursor.Column != 0 {
		t.Errorf("Column = %d, want 0", cursor.Column)
	}

	// Move up at start should stay at 0
	e.MoveCursor(MotionUp, 1)
	cursor = e.GetCursor()
	if cursor.Line != 0 {
		t.Errorf("Line = %d, want 0", cursor.Line)
	}
}

func TestEngineMotionActions(t *testing.T) {
	e := NewEngine()

	// Line start
	e.SetCursor(Position{Line: 0, Column: 10})
	e.MoveCursor(MotionLineStart, 1)
	cursor := e.GetCursor()
	if cursor.Column != 0 {
		t.Errorf("Column = %d, want 0", cursor.Column)
	}

	// File start
	e.SetCursor(Position{Line: 10, Column: 10})
	e.MoveCursor(MotionFileStart, 1)
	cursor = e.GetCursor()
	if cursor.Line != 0 || cursor.Column != 0 {
		t.Errorf("Cursor = (%d, %d), want (0, 0)", cursor.Line, cursor.Column)
	}
}

func TestEngineRegisters(t *testing.T) {
	e := NewEngine()

	// Set register
	e.SetRegister("a", "test content", RegLinewise)
	reg := e.GetRegister("a")
	if reg == nil {
		t.Fatal("Register should exist")
	}

	if reg.Content != "test content" {
		t.Errorf("Content = %q, want 'test content'", reg.Content)
	}

	if reg.Type != RegLinewise {
		t.Errorf("Type = %v, want %v", reg.Type, RegLinewise)
	}

	// Unnamed register
	unnamed := e.GetRegister("")
	if unnamed == nil || unnamed.Content != "test content" {
		t.Error("Unnamed register should have same content")
	}
}

func TestEngineMacros(t *testing.T) {
	e := NewEngine()

	// Start recording
	e.StartMacro("a")
	if e.state.MacroRecording != "a" {
		t.Error("Should be recording macro 'a'")
	}

	// Stop recording
	e.StopMacro()
	if e.state.MacroRecording != "" {
		t.Error("Should not be recording")
	}

	// Play macro (should not error)
	e.PlayMacro("a")
}

func TestEngineHandleKeyNormalMode(t *testing.T) {
	e := NewEngine()

	// Test h (left)
	e.SetCursor(Position{Line: 0, Column: 5})
	e.HandleKey(Key{Rune: 'h'})
	cursor := e.GetCursor()
	if cursor.Column != 4 {
		t.Errorf("Column = %d, want 4", cursor.Column)
	}

	// Test l (right)
	e.HandleKey(Key{Rune: 'l'})
	cursor = e.GetCursor()
	if cursor.Column != 5 {
		t.Errorf("Column = %d, want 5", cursor.Column)
	}

	// Test j (down)
	e.HandleKey(Key{Rune: 'j'})
	cursor = e.GetCursor()
	if cursor.Line != 1 {
		t.Errorf("Line = %d, want 1", cursor.Line)
	}

	// Test k (up)
	e.HandleKey(Key{Rune: 'k'})
	cursor = e.GetCursor()
	if cursor.Line != 0 {
		t.Errorf("Line = %d, want 0", cursor.Line)
	}
}

func TestEngineHandleKeyModeSwitch(t *testing.T) {
	e := NewEngine()

	// i -> insert mode
	e.HandleKey(Key{Rune: 'i'})
	if e.GetMode() != ModeInsert {
		t.Errorf("Mode = %v, want %v", e.GetMode(), ModeInsert)
	}

	// Escape -> normal mode
	e.HandleKey(Key{Special: SpecialEscape})
	if e.GetMode() != ModeNormal {
		t.Errorf("Mode = %v, want %v", e.GetMode(), ModeNormal)
	}

	// v -> visual mode
	e.HandleKey(Key{Rune: 'v'})
	if e.GetMode() != ModeVisual {
		t.Errorf("Mode = %v, want %v", e.GetMode(), ModeVisual)
	}

	// Escape -> normal mode
	e.HandleKey(Key{Special: SpecialEscape})
	if e.GetMode() != ModeNormal {
		t.Errorf("Mode = %v, want %v", e.GetMode(), ModeNormal)
	}
}

func TestEngineHandleKeyInsertMode(t *testing.T) {
	e := NewEngine()

	// Enter insert mode
	e.HandleKey(Key{Rune: 'i'})

	// Enter should add newline
	e.HandleKey(Key{Special: SpecialEnter})

	// Backspace should move back
	e.HandleKey(Key{Special: SpecialBackspace})

	// Tab should add spaces
	e.HandleKey(Key{Special: SpecialTab})
}

func TestEngineVisualSelection(t *testing.T) {
	e := NewEngine()

	// Enter visual mode
	e.SetMode(ModeVisual)
	sel := e.GetSelection()
	if sel == nil {
		t.Fatal("Should have selection")
	}

	// Selection should start at cursor
	if sel.Start.Line != e.GetCursor().Line || sel.Start.Column != e.GetCursor().Column {
		t.Error("Selection should start at cursor")
	}

	// Move cursor should extend selection
	e.MoveCursor(MotionRight, 5)
	sel = e.GetSelection()
	if sel.End.Column != 5 {
		t.Errorf("End column = %d, want 5", sel.End.Column)
	}
}

func TestParseCount(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"1", 1},
		{"5", 5},
		{"10", 10},
		{"100", 100},
		{"abc", 0},
		{"1a", 0},
	}

	for _, tt := range tests {
		result := parseCount(tt.input)
		if result != tt.expected {
			t.Errorf("parseCount(%q) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestEngineExecuteCommand(t *testing.T) {
	e := NewEngine()

	// Test various ex commands (should not error)
	e.ExecuteCommand("w")
	e.ExecuteCommand("q")
	e.ExecuteCommand("wq")
	e.ExecuteCommand("x")
	e.ExecuteCommand("e test.txt")
	e.ExecuteCommand("s/foo/bar/")
	e.ExecuteCommand("noh")
	e.ExecuteCommand("set number")
}

func TestEngineSearch(t *testing.T) {
	e := NewEngine()

	e.Search("test", DirForward)
	if e.state.LastSearch != "test" {
		t.Error("LastSearch should be 'test'")
	}

	if e.state.LastSearchDir != DirForward {
		t.Error("LastSearchDir should be DirForward")
	}

	e.SearchNext()
	e.SearchPrev()
}

func TestEngineKeymap(t *testing.T) {
	e := NewEngine()

	called := false
	action := func(e *Engine) error {
		called = true
		return nil
	}

	// Set custom keymap
	e.SetKeymap(ModeNormal, Key{Rune: 'z'}, action)

	// Press key
	e.HandleKey(Key{Rune: 'z'})

	if !called {
		t.Error("Custom action should be called")
	}

	// Remove keymap
	e.RemoveKeymap(ModeNormal, Key{Rune: 'z'})
	e.HandleKey(Key{Rune: 'z'})
}

func TestPosition(t *testing.T) {
	pos := Position{Line: 5, Column: 10}

	if pos.Line != 5 {
		t.Errorf("Line = %d, want 5", pos.Line)
	}

	if pos.Column != 10 {
		t.Errorf("Column = %d, want 10", pos.Column)
	}
}

func TestSelection(t *testing.T) {
	sel := Selection{
		Start: Position{Line: 0, Column: 0},
		End:   Position{Line: 5, Column: 10},
		Type:  ModeVisual,
	}

	if sel.Start.Line != 0 {
		t.Error("Selection start line should be 0")
	}

	if sel.End.Line != 5 {
		t.Error("Selection end line should be 5")
	}

	if sel.Type != ModeVisual {
		t.Errorf("Selection type = %v, want %v", sel.Type, ModeVisual)
	}
}

func TestRegister(t *testing.T) {
	reg := Register{
		Name:    "a",
		Content: "test content",
		Type:    RegLinewise,
	}

	if reg.Name != "a" {
		t.Error("Register name should be 'a'")
	}

	if reg.Content != "test content" {
		t.Error("Register content mismatch")
	}
}

func TestKey(t *testing.T) {
	// Regular key
	key := Key{Rune: 'a'}
	if key.Rune != 'a' {
		t.Error("Key rune should be 'a'")
	}

	// Special key
	specialKey := Key{Special: SpecialEscape}
	if specialKey.Special != SpecialEscape {
		t.Error("Key special should be SpecialEscape")
	}

	// Key with modifier
	modKey := Key{Rune: 'c', Modifier: ModCtrl}
	if modKey.Modifier != ModCtrl {
		t.Error("Key modifier should be ModCtrl")
	}
}

func TestCommand(t *testing.T) {
	cmd := Command{
		Type:     CmdOperatorMotion,
		Operator: OpDelete,
		Motion:   MotionWordForward,
		Count:    3,
	}

	if cmd.Type != CmdOperatorMotion {
		t.Error("Command type mismatch")
	}

	if cmd.Count != 3 {
		t.Error("Command count should be 3")
	}
}

func TestEngineState(t *testing.T) {
	e := NewEngine()

	state := e.GetState()
	if state.Mode != ModeNormal {
		t.Errorf("State mode = %v, want %v", state.Mode, ModeNormal)
	}

	if state.Registers == nil {
		t.Error("State registers should not be nil")
	}
}

func TestSpecialKeys(t *testing.T) {
	keys := []SpecialKey{
		SpecialNone,
		SpecialEscape,
		SpecialEnter,
		SpecialBackspace,
		SpecialDelete,
		SpecialTab,
		SpecialUp,
		SpecialDown,
		SpecialLeft,
		SpecialRight,
		SpecialHome,
		SpecialEnd,
		SpecialPageUp,
		SpecialPageDown,
	}

	for _, key := range keys {
		if key < SpecialNone || key > SpecialPageDown {
			t.Errorf("Invalid special key: %d", key)
		}
	}
}

func TestModifiers(t *testing.T) {
	// Test individual modifiers
	if ModShift&ModShift == 0 {
		t.Error("ModShift should be set")
	}
	if ModCtrl&ModCtrl == 0 {
		t.Error("ModCtrl should be set")
	}
	if ModAlt&ModAlt == 0 {
		t.Error("ModAlt should be set")
	}

	// Test combined modifiers
	combined := ModCtrl | ModAlt
	if combined&ModCtrl == 0 {
		t.Error("Combined should have ModCtrl")
	}
	if combined&ModAlt == 0 {
		t.Error("Combined should have ModAlt")
	}
}

func TestMotions(t *testing.T) {
	motions := []Motion{
		MotionNone,
		MotionLeft,
		MotionRight,
		MotionUp,
		MotionDown,
		MotionWordForward,
		MotionWordBackward,
		MotionWordEnd,
		MotionLineStart,
		MotionLineEnd,
		MotionFileStart,
		MotionFileEnd,
	}

	for _, motion := range motions {
		if motion < MotionNone {
			t.Errorf("Invalid motion: %d", motion)
		}
	}
}

func TestOperators(t *testing.T) {
	operators := []Operator{
		OpNone,
		OpDelete,
		OpChange,
		OpYank,
		OpPut,
		OpReplace,
		OpUndo,
		OpRedo,
	}

	for _, op := range operators {
		if op < OpNone {
			t.Errorf("Invalid operator: %d", op)
		}
	}
}

func TestTextObjects(t *testing.T) {
	objects := []TextObject{
		TextObjNone,
		TextObjWord,
		TextObjWordInner,
		TextObjSentence,
		TextObjParagraph,
		TextObjBlock,
		TextObjQuote,
		TextObjLine,
	}

	for _, obj := range objects {
		if obj < TextObjNone {
			t.Errorf("Invalid text object: %d", obj)
		}
	}
}

func TestEngineConcurrency(t *testing.T) {
	e := NewEngine()
	done := make(chan bool, 100)

	// Concurrent reads
	for i := 0; i < 50; i++ {
		go func() {
			e.GetMode()
			e.GetCursor()
			done <- true
		}()
	}

	// Concurrent writes
	for i := 0; i < 50; i++ {
		go func(idx int) {
			e.MoveCursor(MotionRight, 1)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	// Should not have race conditions
	cursor := e.GetCursor()
	if cursor.Column < 0 {
		t.Error("Column should not be negative")
	}
}
