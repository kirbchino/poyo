// Package vim provides Vim-style editing mode support.
package vim

import (
	"strings"
	"sync"
	"time"
)

// Mode represents a Vim mode
type Mode string

const (
	ModeNormal   Mode = "normal"
	ModeInsert   Mode = "insert"
	ModeVisual   Mode = "visual"
	ModeVisualLine Mode = "visual_line"
	ModeVisualBlock Mode = "visual_block"
	ModeCommand  Mode = "command"
	ModeReplace  Mode = "replace"
)

// Key represents a key press
type Key struct {
	Rune      rune
	Modifier  Modifier
	Special   SpecialKey
}

// Modifier represents key modifiers
type Modifier int

const (
	ModNone Modifier = 0
	ModShift Modifier = 1 << iota
	ModCtrl
	ModAlt
	ModMeta
)

// SpecialKey represents special keys
type SpecialKey int

const (
	SpecialNone SpecialKey = iota
	SpecialEscape
	SpecialEnter
	SpecialBackspace
	SpecialDelete
	SpecialTab
	SpecialUp
	SpecialDown
	SpecialLeft
	SpecialRight
	SpecialHome
	SpecialEnd
	SpecialPageUp
	SpecialPageDown
)

// Command represents a Vim command
type Command struct {
	Type       CommandType
	Motion     Motion
	Operator   Operator
	Count      int
	TextObject TextObject
	Register   string
	Argument   string
}

// CommandType represents the type of command
type CommandType int

const (
	CmdMotion CommandType = iota
	CmdOperator
	CmdOperatorMotion
	CmdTextObject
	CmdInsert
	CmdExCommand
	CmdModeSwitch
)

// Motion represents cursor motions
type Motion int

const (
	MotionNone Motion = iota
	MotionLeft
	MotionRight
	MotionUp
	MotionDown
	MotionWordForward
	MotionWordBackward
	MotionWordEnd
	MotionLineStart
	MotionLineEnd
	MotionLineFirst
	MotionLineLast
	MotionFileStart
	MotionFileEnd
	MotionParagraphForward
	MotionParagraphBackward
	MotionSearchForward
	MotionSearchBackward
	MotionMatchPair
	MotionLineN
	MotionColumnN
)

// Operator represents operators
type Operator int

const (
	OpNone Operator = iota
	OpDelete
	OpChange
	OpYank
	OpPut
	OpReplace
	OpShiftLeft
	OpShiftRight
	OpFormat
	OpFilter
	OpUndo
	OpRedo
)

// TextObject represents text objects
type TextObject int

const (
	TextObjNone TextObject = iota
	TextObjWord
	TextObjWordInner
	TextObjSentence
	TextObjSentenceInner
	TextObjParagraph
	TextObjParagraphInner
	TextObjBlock
	TextObjBlockInner
	TextObjQuote
	TextObjQuoteInner
	TextObjTag
	TextObjTagInner
	TextObjLine
	TextObjLineInner
)

// Position represents a cursor position
type Position struct {
	Line   int
	Column int
}

// Selection represents a visual selection
type Selection struct {
	Start Position
	End   Position
	Type  Mode
}

// Register represents a Vim register
type Register struct {
	Name    string
	Content string
	Type    RegisterType
}

// RegisterType represents the type of register content
type RegisterType int

const (
	RegLinewise RegisterType = iota
	RegCharacterwise
	RegBlockwise
)

// State represents the current Vim state
type State struct {
	Mode           Mode
	Cursor         Position
	Selection      *Selection
	Registers      map[string]*Register
	LastSearch     string
	LastSearchDir  Direction
	LastCommand    string
	LastInsert     string
	LastYank       string
	MacroRecording string
	MacroPlaying   string
	PendingCommand *Command
	PendingKeys    string
	SearchPattern  string
	ReplacePattern string
}

// Direction represents search direction
type Direction int

const (
	DirForward Direction = iota
	DirBackward
)

// Engine represents the Vim engine
type Engine struct {
	state    State
	keymaps  map[Mode]map[Key]Action
	macros   map[string][]Key
	recording []Key
	mu       sync.RWMutex
}

// Action represents an action to perform
type Action func(*Engine) error

// NewEngine creates a new Vim engine
func NewEngine() *Engine {
	e := &Engine{
		state: State{
			Mode:      ModeNormal,
			Cursor:    Position{Line: 0, Column: 0},
			Registers: make(map[string]*Register),
		},
		keymaps: make(map[Mode]map[Key]Action),
		macros:  make(map[string][]Key),
	}
	e.setupDefaultKeymaps()
	return e
}

// setupDefaultKeymaps sets up the default Vim keymaps
func (e *Engine) setupDefaultKeymaps() {
	// Normal mode keymaps
	normalKeys := map[Key]Action{
		{Rune: 'h'}: motionAction(MotionLeft),
		{Rune: 'j'}: motionAction(MotionDown),
		{Rune: 'k'}: motionAction(MotionUp),
		{Rune: 'l'}: motionAction(MotionRight),
		{Rune: 'w'}: motionAction(MotionWordForward),
		{Rune: 'W'}: motionAction(MotionWordForward), // WORD
		{Rune: 'b'}: motionAction(MotionWordBackward),
		{Rune: 'B'}: motionAction(MotionWordBackward), // WORD
		{Rune: 'e'}: motionAction(MotionWordEnd),
		{Rune: 'E'}: motionAction(MotionWordEnd), // WORD
		{Rune: '0'}: motionAction(MotionLineStart),
		{Rune: '^'}: motionAction(MotionLineFirst),
		{Rune: '$'}: motionAction(MotionLineEnd),
		{Rune: 'g'}: fileStartMotion(),
		{Rune: 'G'}: motionAction(MotionFileEnd),
		{Rune: 'i'}: modeSwitchAction(ModeInsert),
		{Rune: 'I'}: insertLineStartAction(),
		{Rune: 'a'}: insertAfterAction(),
		{Rune: 'A'}: insertLineEndAction(),
		{Rune: 'o'}: insertLineBelowAction(),
		{Rune: 'O'}: insertLineAboveAction(),
		{Rune: 'v'}: visualModeAction(ModeVisual),
		{Rune: 'V'}: visualModeAction(ModeVisualLine),
		{Rune: 'd'}: operatorAction(OpDelete),
		{Rune: 'D'}: deleteLineAction(),
		{Rune: 'c'}: operatorAction(OpChange),
		{Rune: 'C'}: changeLineAction(),
		{Rune: 'y'}: operatorAction(OpYank),
		{Rune: 'Y'}: yankLineAction(),
		{Rune: 'p'}: putAfterAction(),
		{Rune: 'P'}: putBeforeAction(),
		{Rune: 'x'}: deleteCharAction(),
		{Rune: 'X'}: deleteCharBeforeAction(),
		{Rune: 'r'}: replaceCharAction(),
		{Rune: 'R'}: modeSwitchAction(ModeReplace),
		{Rune: 'u'}: undoAction(),
		{Rune: 'U'}: undoLineAction(),
		{Rune: '.': repeatLastAction(),
		{Special: SpecialEscape}: modeSwitchAction(ModeNormal),
		{Special: SpecialEnter}:  motionAction(MotionDown),
		{Special: SpecialBackspace}: motionAction(MotionLeft),
	}
	e.keymaps[ModeNormal] = normalKeys

	// Insert mode keymaps
	insertKeys := map[Key]Action{
		{Special: SpecialEscape}: modeSwitchAction(ModeNormal),
		{Special: SpecialEnter}:  insertNewlineAction(),
		{Special: SpecialBackspace}: deleteBackAction(),
		{Special: SpecialTab}: insertTabAction(),
	}
	e.keymaps[ModeInsert] = insertKeys

	// Visual mode keymaps
	visualKeys := map[Key]Action{
		{Special: SpecialEscape}: modeSwitchAction(ModeNormal),
		{Rune: 'h'}: motionAction(MotionLeft),
		{Rune: 'j'}: motionAction(MotionDown),
		{Rune: 'k'}: motionAction(MotionUp),
		{Rune: 'l'}: motionAction(MotionRight),
		{Rune: 'w'}: motionAction(MotionWordForward),
		{Rune: 'b'}: motionAction(MotionWordBackward),
		{Rune: 'd'}: visualDeleteAction(),
		{Rune: 'y'}: visualYankAction(),
		{Rune: 'c'}: visualChangeAction(),
		{Rune: 'o'}: visualMoveOtherEndAction(),
	}
	e.keymaps[ModeVisual] = visualKeys
	e.keymaps[ModeVisualLine] = visualKeys
	e.keymaps[ModeVisualBlock] = visualKeys
}

// HandleKey handles a key press
func (e *Engine) HandleKey(key Key) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Record macro if recording
	if e.state.MacroRecording != "" {
		e.recording = append(e.recording, key)
	}

	// Get keymap for current mode
	keymap, ok := e.keymaps[e.state.Mode]
	if !ok {
		return nil
	}

	// Find action
	action, ok := keymap[key]
	if !ok {
		return e.handlePendingKeys(key)
	}

	return action(e)
}

// handlePendingKeys handles keys that form multi-key commands
func (e *Engine) handlePendingKeys(key Key) error {
	e.state.PendingKeys += string(key.Rune)

	// Check for number prefix (count)
	if e.state.PendingKeys == "0" && e.state.PendingCommand == nil {
		// 0 is line start motion, not count
		return motionAction(MotionLineStart)(e)
	}

	// Parse count
	if len(e.state.PendingKeys) > 0 {
		count := parseCount(e.state.PendingKeys)
		if count > 0 {
			e.state.PendingCommand = &Command{Count: count}
			e.state.PendingKeys = ""
			return nil
		}
	}

	// Check for multi-key commands
	switch e.state.PendingKeys {
	case "gg":
		e.state.PendingKeys = ""
		return motionAction(MotionFileStart)(e)
	case "dd":
		e.state.PendingKeys = ""
		return deleteLineAction()(e)
	case "yy":
		e.state.PendingKeys = ""
		return yankLineAction()(e)
	case "cc":
		e.state.PendingKeys = ""
		return changeLineAction()(e)
	case "dw":
		e.state.PendingKeys = ""
		return deleteWordAction()(e)
	case "cw":
		e.state.PendingKeys = ""
		return changeWordAction()(e)
	case "yw":
		e.state.PendingKeys = ""
		return yankWordAction()(e)
	case "di":
		// Wait for text object
		return nil
	case "da":
		// Wait for text object
		return nil
	}

	// Clear pending keys if no match after timeout
	if len(e.state.PendingKeys) > 3 {
		e.state.PendingKeys = ""
	}

	return nil
}

// parseCount parses a count prefix from keys
func parseCount(s string) int {
	count := 0
	for _, c := range s {
		if c >= '1' && c <= '9' {
			count = count*10 + int(c-'0')
		} else {
			break
		}
	}
	return count
}

// GetMode returns the current mode
func (e *Engine) GetMode() Mode {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state.Mode
}

// SetMode sets the current mode
func (e *Engine) SetMode(mode Mode) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.state.Mode = mode

	// Clear selection when leaving visual mode
	if mode != ModeVisual && mode != ModeVisualLine && mode != ModeVisualBlock {
		e.state.Selection = nil
	}

	// Initialize selection when entering visual mode
	if mode == ModeVisual || mode == ModeVisualLine || mode == ModeVisualBlock {
		e.state.Selection = &Selection{
			Start: e.state.Cursor,
			End:   e.state.Cursor,
			Type:  mode,
		}
	}
}

// GetCursor returns the current cursor position
func (e *Engine) GetCursor() Position {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state.Cursor
}

// SetCursor sets the cursor position
func (e *Engine) SetCursor(pos Position) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.state.Cursor = pos
}

// MoveCursor moves the cursor
func (e *Engine) MoveCursor(motion Motion, count int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if count == 0 {
		count = 1
	}

	for i := 0; i < count; i++ {
		e.moveCursorOnce(motion)
	}

	// Update visual selection
	if e.state.Selection != nil {
		e.state.Selection.End = e.state.Cursor
	}
}

// moveCursorOnce moves the cursor one step
func (e *Engine) moveCursorOnce(motion Motion) {
	switch motion {
	case MotionLeft:
		if e.state.Cursor.Column > 0 {
			e.state.Cursor.Column--
		}
	case MotionRight:
		e.state.Cursor.Column++
	case MotionUp:
		if e.state.Cursor.Line > 0 {
			e.state.Cursor.Line--
		}
	case MotionDown:
		e.state.Cursor.Line++
	case MotionWordForward:
		// Simplified: just move right
		e.state.Cursor.Column++
	case MotionWordBackward:
		if e.state.Cursor.Column > 0 {
			e.state.Cursor.Column--
		}
	case MotionLineStart:
		e.state.Cursor.Column = 0
	case MotionLineEnd:
		e.state.Cursor.Column = 9999 // Will be clamped
	case MotionFileStart:
		e.state.Cursor.Line = 0
		e.state.Cursor.Column = 0
	case MotionFileEnd:
		e.state.Cursor.Line = 9999
		e.state.Cursor.Column = 0
	}
}

// GetRegister gets a register by name
func (e *Engine) GetRegister(name string) *Register {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state.Registers[name]
}

// SetRegister sets a register
func (e *Engine) SetRegister(name string, content string, regType RegisterType) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.state.Registers[name] = &Register{
		Name:    name,
		Content: content,
		Type:    regType,
	}
	// Also set unnamed register
	e.state.Registers[""] = e.state.Registers[name]
}

// GetSelection returns the current selection
func (e *Engine) GetSelection() *Selection {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state.Selection
}

// HasSelection returns true if there is a selection
func (e *Engine) HasSelection() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state.Selection != nil
}

// StartMacro starts recording a macro
func (e *Engine) StartMacro(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.state.MacroRecording = name
	e.recording = nil
}

// StopMacro stops recording a macro
func (e *Engine) StopMacro() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state.MacroRecording != "" {
		e.macros[e.state.MacroRecording] = e.recording
		e.state.MacroRecording = ""
		e.recording = nil
	}
}

// PlayMacro plays a macro
func (e *Engine) PlayMacro(name string) error {
	e.mu.RLock()
	keys, ok := e.macros[name]
	e.mu.RUnlock()

	if !ok {
		return nil
	}

	for _, key := range keys {
		if err := e.HandleKey(key); err != nil {
			return err
		}
	}

	return nil
}

// ModeName returns a human-readable mode name
func (m Mode) String() string {
	switch m {
	case ModeNormal:
		return "NORMAL"
	case ModeInsert:
		return "INSERT"
	case ModeVisual:
		return "VISUAL"
	case ModeVisualLine:
		return "VISUAL LINE"
	case ModeVisualBlock:
		return "VISUAL BLOCK"
	case ModeCommand:
		return "COMMAND"
	case ModeReplace:
		return "REPLACE"
	default:
		return string(m)
	}
}

// IsInsertMode returns true if in an insert mode
func (m Mode) IsInsertMode() bool {
	return m == ModeInsert || m == ModeReplace
}

// IsVisualMode returns true if in a visual mode
func (m Mode) IsVisualMode() bool {
	return m == ModeVisual || m == ModeVisualLine || m == ModeVisualBlock
}

// Action factories

func motionAction(motion Motion) Action {
	return func(e *Engine) error {
		count := 1
		if e.state.PendingCommand != nil {
			count = e.state.PendingCommand.Count
			e.state.PendingCommand = nil
		}
		e.MoveCursor(motion, count)
		return nil
	}
}

func modeSwitchAction(mode Mode) Action {
	return func(e *Engine) error {
		e.SetMode(mode)
		return nil
	}
}

func visualModeAction(mode Mode) Action {
	return func(e *Engine) error {
		e.SetMode(mode)
		return nil
	}
}

func operatorAction(op Operator) Action {
	return func(e *Engine) error {
		e.state.PendingCommand = &Command{
			Type:     CmdOperator,
			Operator: op,
		}
		return nil
	}
}

func insertLineStartAction() Action {
	return func(e *Engine) error {
		e.state.Cursor.Column = 0
		e.SetMode(ModeInsert)
		return nil
	}
}

func insertAfterAction() Action {
	return func(e *Engine) error {
		e.state.Cursor.Column++
		e.SetMode(ModeInsert)
		return nil
	}
}

func insertLineEndAction() Action {
	return func(e *Engine) error {
		e.state.Cursor.Column = 9999
		e.SetMode(ModeInsert)
		return nil
	}
}

func insertLineBelowAction() Action {
	return func(e *Engine) error {
		e.state.Cursor.Line++
		e.state.Cursor.Column = 0
		e.SetMode(ModeInsert)
		return nil
	}
}

func insertLineAboveAction() Action {
	return func(e *Engine) error {
		e.state.Cursor.Column = 0
		e.SetMode(ModeInsert)
		return nil
	}
}

func deleteLineAction() Action {
	return func(e *Engine) error {
		// Mark line for deletion
		e.state.PendingCommand = &Command{
			Type:     CmdOperatorMotion,
			Operator: OpDelete,
		}
		return nil
	}
}

func changeLineAction() Action {
	return func(e *Engine) error {
		e.state.PendingCommand = &Command{
			Type:     CmdOperatorMotion,
			Operator: OpChange,
		}
		e.SetMode(ModeInsert)
		return nil
	}
}

func yankLineAction() Action {
	return func(e *Engine) error {
		// Yank current line
		e.SetRegister("", "", RegLinewise)
		return nil
	}
}

func deleteCharAction() Action {
	return func(e *Engine) error {
		// Delete character under cursor
		return nil
	}
}

func deleteCharBeforeAction() Action {
	return func(e *Engine) error {
		if e.state.Cursor.Column > 0 {
			e.state.Cursor.Column--
		}
		return nil
	}
}

func replaceCharAction() Action {
	return func(e *Engine) error {
		e.state.PendingCommand = &Command{
			Type: CmdOperator,
			Operator: OpReplace,
		}
		return nil
	}
}

func undoAction() Action {
	return func(e *Engine) error {
		// Trigger undo
		return nil
	}
}

func undoLineAction() Action {
	return func(e *Engine) error {
		// Undo all changes on current line
		return nil
	}
}

func repeatLastAction() Action {
	return func(e *Engine) error {
		// Repeat last command
		return nil
	}
}

func putAfterAction() Action {
	return func(e *Engine) error {
		// Put register content after cursor
		return nil
	}
}

func putBeforeAction() Action {
	return func(e *Engine) error {
		// Put register content before cursor
		return nil
	}
}

func deleteWordAction() Action {
	return func(e *Engine) error {
		// Delete word
		return nil
	}
}

func changeWordAction() Action {
	return func(e *Engine) error {
		e.SetMode(ModeInsert)
		return nil
	}
}

func yankWordAction() Action {
	return func(e *Engine) error {
		// Yank word
		return nil
	}
}

func fileStartMotion() Action {
	return func(e *Engine) error {
		count := 1
		if e.state.PendingCommand != nil {
			count = e.state.PendingCommand.Count
			e.state.PendingCommand = nil
		}
		if count == 1 {
			e.MoveCursor(MotionFileStart, 1)
		} else {
			// Go to line N
			e.state.Cursor.Line = count - 1
			e.state.Cursor.Column = 0
		}
		return nil
	}
}

func visualDeleteAction() Action {
	return func(e *Engine) error {
		// Delete selection
		e.SetMode(ModeNormal)
		return nil
	}
}

func visualYankAction() Action {
	return func(e *Engine) error {
		// Yank selection
		e.SetMode(ModeNormal)
		return nil
	}
}

func visualChangeAction() Action {
	return func(e *Engine) error {
		// Change selection
		e.SetMode(ModeInsert)
		return nil
	}
}

func visualMoveOtherEndAction() Action {
	return func(e *Engine) error {
		// Move to other end of selection
		if e.state.Selection != nil {
			e.state.Cursor = e.state.Selection.Start
			e.state.Selection.Start = e.state.Selection.End
			e.state.Selection.End = e.state.Cursor
		}
		return nil
	}
}

func insertNewlineAction() Action {
	return func(e *Engine) error {
		// Insert newline
		e.state.Cursor.Line++
		e.state.Cursor.Column = 0
		return nil
	}
}

func deleteBackAction() Action {
	return func(e *Engine) error {
		if e.state.Cursor.Column > 0 {
			e.state.Cursor.Column--
		}
		return nil
	}
}

func insertTabAction() Action {
	return func(e *Engine) error {
		e.state.Cursor.Column += 4
		return nil
	}
}

// Search functionality

// Search performs a search
func (e *Engine) Search(pattern string, direction Direction) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.state.LastSearch = pattern
	e.state.LastSearchDir = direction
	e.state.SearchPattern = pattern
}

// SearchNext searches for the next match
func (e *Engine) SearchNext() {
	// Search for next occurrence of LastSearch
}

// SearchPrev searches for the previous match
func (e *Engine) SearchPrev() {
	// Search for previous occurrence of LastSearch
}

// Ex commands

// ExecuteCommand executes an ex command
func (e *Engine) ExecuteCommand(cmd string) error {
	parts := strings.SplitN(cmd, " ", 2)
	command := parts[0]
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	switch command {
	case "w", "write":
		return e.cmdWrite(args)
	case "q", "quit":
		return e.cmdQuit(args)
	case "wq":
		e.cmdWrite(args)
		return e.cmdQuit(args)
	case "x", "exit":
		e.cmdWrite(args)
		return e.cmdQuit(args)
	case "e", "edit":
		return e.cmdEdit(args)
	case "s", "substitute":
		return e.cmdSubstitute(args)
	case "g", "global":
		return e.cmdGlobal(args)
	case "v", "vglobal":
		return e.cmdVGlobal(args)
	case "d", "delete":
		return e.cmdDelete(args)
	case "y", "yank":
		return e.cmdYank(args)
	case "p", "put":
		return e.cmdPut(args)
	case "%s":
		return e.cmdSubstituteAll(args)
	case "noh", "nohlsearch":
		return e.cmdNoHighlight(args)
	case "set":
		return e.cmdSet(args)
	case "map":
		return e.cmdMap(args)
	case "nmap":
		return e.cmdNmap(args)
	case "imap":
		return e.cmdImap(args)
	case "vmap":
		return e.cmdVmap(args)
	default:
		return nil
	}
}

func (e *Engine) cmdWrite(args string) error {
	// Save file
	return nil
}

func (e *Engine) cmdQuit(args string) error {
	// Quit
	return nil
}

func (e *Engine) cmdEdit(args string) error {
	// Edit file
	return nil
}

func (e *Engine) cmdSubstitute(args string) error {
	// Substitute pattern
	return nil
}

func (e *Engine) cmdGlobal(args string) error {
	// Global command
	return nil
}

func (e *Engine) cmdVGlobal(args string) error {
	// Inverse global command
	return nil
}

func (e *Engine) cmdDelete(args string) error {
	// Delete lines
	return nil
}

func (e *Engine) cmdYank(args string) error {
	// Yank lines
	return nil
}

func (e *Engine) cmdPut(args string) error {
	// Put lines
	return nil
}

func (e *Engine) cmdSubstituteAll(args string) error {
	// Substitute all occurrences
	return nil
}

func (e *Engine) cmdNoHighlight(args string) error {
	// Clear search highlight
	return nil
}

func (e *Engine) cmdSet(args string) error {
	// Set option
	return nil
}

func (e *Engine) cmdMap(args string) error {
	// Map key
	return nil
}

func (e *Engine) cmdNmap(args string) error {
	// Normal mode map
	return nil
}

func (e *Engine) cmdImap(args string) error {
	// Insert mode map
	return nil
}

func (e *Engine) cmdVmap(args string) error {
	// Visual mode map
	return nil
}

// GetState returns a copy of the current state
func (e *Engine) GetState() State {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state
}

// SetKeymap sets a key mapping
func (e *Engine) SetKeymap(mode Mode, key Key, action Action) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.keymaps[mode] == nil {
		e.keymaps[mode] = make(map[Key]Action)
	}
	e.keymaps[mode][key] = action
}

// RemoveKeymap removes a key mapping
func (e *Engine) RemoveKeymap(mode Mode, key Key) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.keymaps[mode] != nil {
		delete(e.keymaps[mode], key)
	}
}
