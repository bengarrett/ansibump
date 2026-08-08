package ansibump //nolint:testpackage

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/nalgeon/be"
	"golang.org/x/text/encoding/charmap"
)

func TestNewDecoder(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		custom   *Customizer
		expected *Decoder
	}{
		{
			name:   "nil customizer defaults",
			custom: nil,
			expected: &Decoder{
				width:   DefaultColumns,
				charset: charmap.XUserDefined,
				buffer:  [][]cell{{}},
				x:       0,
				y:       0,
			},
		},
		{
			name: "custom width and strict flags",
			custom: &Customizer{
				Width:  120,
				Strict: true,
			},
			expected: &Decoder{
				width:   120,
				charset: charmap.XUserDefined,
				buffer:  [][]cell{{}},
				strict:  true,
			},
		},
		{
			name: "zero or negative width falls back to default",
			custom: &Customizer{
				Width: -5,
			},
			expected: &Decoder{
				width:   DefaultColumns,
				charset: charmap.XUserDefined,
				buffer:  [][]cell{{}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NewDecoder(tt.custom)
			be.Equal(t, got.width, tt.expected.width)
			be.Equal(t, got.strict, tt.expected.strict)
			be.Equal(t, got.amigaParser, tt.expected.amigaParser)
			be.Equal(t, len(got.buffer), len(tt.expected.buffer))
		})
	}
}

func TestSetWrapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cb       byte
		linewrap bool
		current  bool
		want     bool
	}{
		{"enable line wrap (=7h)", 'h', true, false, true},
		{"disable line wrap (=7l)", 'l', true, true, false},
		{"ignore non-7 mode enable", 'h', false, false, false},
		{"ignore non-7 mode disable", 'l', false, true, true},
		{"unknown character returns current state", 'x', true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := setWrapping(tt.cb, tt.linewrap, tt.current)
			be.Equal(t, got, tt.want)
		})
	}
}

type cursorTest = struct {
	name      string
	initial   int
	params    []int
	strict    bool
	want      int
	wantError bool
}

func TestCursorUp(t *testing.T) {
	t.Parallel()
	tests := []cursorTest{
		{
			name: "default moves up by 1", initial: 5, params: []int{}, want: 4,
		},
		{
			name: "explicit parameter moves up by N", initial: 5, params: []int{3}, want: 2,
		},
		{
			name: "explicit zero defaults to moving up by 1", initial: 5, params: []int{0}, want: 4,
		},
		{
			name: "too many parameters", initial: 5, params: []int{1, 2}, strict: true, want: 5, wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testCursor(t, tt, "up")
		})
	}
}

func TestCursorDown(t *testing.T) {
	t.Parallel()
	tests := []cursorTest{
		{
			name: "default moves down by 1", initial: 2, params: []int{}, want: 3,
		},
		{
			name: "explicit parameter moves down by N", initial: 2, params: []int{4}, want: 6,
		},
		{
			name: "explicit zero defaults to moving down by 1", initial: 2, params: []int{0}, want: 3,
		},
		{
			name: "too many parameters", initial: 2, params: []int{1, 2}, strict: true, want: 2, wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testCursor(t, tt, "down")
		})
	}
}

func TestCursorForward(t *testing.T) {
	t.Parallel()
	tests := []cursorTest{
		{
			name: "default moves forward by 1", initial: 2, params: []int{}, want: 3,
		},
		{
			name: "explicit parameter moves forward by N", initial: 2, params: []int{4}, want: 6,
		},
		{
			name: "explicit zero defaults to moving forward by 1", initial: 2, params: []int{0}, want: 3,
		},
		{
			name: "too many parameters", initial: 2, params: []int{1, 2}, strict: true, want: 2, wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testCursor(t, tt, "forward")
		})
	}
}

func TestCursorBack(t *testing.T) {
	t.Parallel()
	tests := []cursorTest{
		{
			name: "default moves back by 1", initial: 5, params: []int{}, want: 4,
		},
		{
			name: "explicit parameter moves back by N", initial: 5, params: []int{3}, want: 2,
		},
		{
			name: "explicit zero defaults to moving back by 1", initial: 5, params: []int{0}, want: 4,
		},
		{
			name: "too many parameters", initial: 5, params: []int{1, 2}, strict: true, want: 5, wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testCursor(t, tt, "back")
		})
	}
}

func testCursor(t *testing.T, tt cursorTest, test string) {
	t.Helper()
	d := NewDecoder(nil)
	d.strict = tt.strict
	var err error
	switch test {
	case "up":
		d.y = tt.initial
		d.x = 0
		err = d.CursorUp(tt.params)
	case "down":
		d.y = tt.initial
		d.x = 0
		err = d.CursorDown(tt.params)
	case "forward":
		d.y = 0
		d.x = tt.initial
		err = d.CursorForward(tt.params)
	case "back":
		d.y = 0
		d.x = tt.initial
		err = d.CursorBack(tt.params)
	default:
		panic("invalid test value")
	}
	if tt.wantError {
		be.Err(t, err)
	} else {
		be.Err(t, err, nil)
	}
	switch test {
	case "up", "down":
		be.Equal(t, d.y, tt.want)
	case "forward", "back":
		be.Equal(t, d.x, tt.want)
	}
}

type testLine struct {
	name      string
	initialX  int
	initialY  int
	params    []int
	strict    bool
	wantX     int
	wantY     int
	wantError bool
}

func TestCursorNextLine(t *testing.T) {
	t.Parallel()

	tests := []testLine{
		{
			name:     "default moves down by 1 and resets X to 0",
			initialX: 10,
			initialY: 5,
			params:   []int{},
			wantX:    0,
			wantY:    6,
		},
		{
			name:     "explicit parameter N moves down by N and resets X to 0",
			initialX: 10,
			initialY: 5,
			params:   []int{3},
			wantX:    0,
			wantY:    8,
		},
		{
			name:     "explicit 0 defaults to moving down by 1 and resets X to 0",
			initialX: 10,
			initialY: 5,
			params:   []int{0},
			wantX:    0,
			wantY:    6,
		},
		{
			name:      "too many parameters in strict mode returns error",
			initialX:  10,
			initialY:  5,
			params:    []int{1, 2},
			strict:    true,
			wantX:     10, // unchanged
			wantY:     5,  // unchanged
			wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			lineTest(t, tt, "next")
		})
	}
}

func TestCursorPreviousLine(t *testing.T) {
	t.Parallel()

	tests := []testLine{
		{
			name:     "default moves up by 1 and resets X to 0",
			initialX: 10,
			initialY: 5,
			params:   []int{},
			wantX:    0,
			wantY:    4,
		},
		{
			name:     "explicit parameter N moves up by N and resets X to 0",
			initialX: 10,
			initialY: 5,
			params:   []int{3},
			wantX:    0,
			wantY:    2,
		},
		{
			name:     "explicit 0 defaults to moving up by 1 and resets X to 0",
			initialX: 10,
			initialY: 5,
			params:   []int{0},
			wantX:    0,
			wantY:    4,
		},
		{
			name:      "too many parameters in strict mode returns error",
			initialX:  10,
			initialY:  5,
			params:    []int{1, 2},
			strict:    true,
			wantX:     10, // unchanged
			wantY:     5,  // unchanged
			wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			lineTest(t, tt, "prev")
		})
	}
}

func lineTest(t *testing.T, tt testLine, line string) {
	t.Helper()
	d := NewDecoder(nil)
	d.x = tt.initialX
	d.y = tt.initialY
	d.strict = tt.strict

	var err error
	switch line {
	case "next":
		err = d.CursorNextLine(tt.params)
	case "prev":
		err = d.CursorPreviousLine(tt.params)
	default:
		panic("invalid line value")
	}

	if tt.wantError {
		be.Err(t, err)
	} else {
		be.Err(t, err, nil)
	}

	be.Equal(t, d.x, tt.wantX)
	be.Equal(t, d.y, tt.wantY)
}

func TestCursorHorizontalAbsolute(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		params  []int
		strict  bool
		wantX   int
		wantErr bool
	}{
		{"default to col 1", []int{}, false, 1, false},
		{"explicit col", []int{5}, false, 5, false},
		{"zero maps to col 1", []int{0}, false, 1, false},
		{"too many params strict", []int{1, 2}, true, 10, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := &Decoder{x: 10, strict: tt.strict}
			err := d.CursorHorizontalAbsolute(tt.params)
			if tt.wantErr {
				be.Err(t, err)
			} else {
				be.Err(t, err, nil)
			}
			be.Equal(t, d.x, tt.wantX)
		})
	}
}

func TestCursorPosition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		params  []int
		strict  bool
		wantX   int
		wantY   int
		wantErr bool
	}{
		{"default to top-left (0,0)", []int{}, false, 0, 0, false},
		{"explicit row and col", []int{5, 10}, false, 9, 4, false},
		{"zeros map to 0,0", []int{0, 0}, false, 0, 0, false},
		{"single param non-strict sets row only", []int{3}, false, 0, 2, false},
		{"single param strict errors", []int{3}, true, 10, 10, true},
		{"too many params strict errors", []int{1, 2, 3}, true, 10, 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := &Decoder{x: 10, y: 10, strict: tt.strict}
			err := d.CursorPosition(tt.params)

			if tt.wantErr {
				be.Err(t, err)
			} else {
				be.Err(t, err, nil)
			}
			be.Equal(t, d.x, tt.wantX)
			be.Equal(t, d.y, tt.wantY)
		})
	}
}

func TestEraseInDisplay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		param   []int
		strict  bool
		wantX   int
		wantY   int
		wantErr bool
	}{
		{"default erase 0 (cursor to end)", []int{0}, false, 2, 1, false},
		{"erase 1 (start to cursor)", []int{1}, false, 2, 1, false},
		{"erase 2 (entire screen)", []int{2}, false, 0, 0, false},
		{"invalid param strict mode", []int{99}, true, 2, 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			line1 := []cell{{Char: 'a'}, {Char: 'b'}, {Char: 'c'}}
			line2 := []cell{{Char: 'd'}, {Char: 'e'}, {Char: 'f'}}

			d := &Decoder{
				x:           2,
				y:           1,
				buffer:      [][]cell{line1, line2},
				currentLine: line2,
				strict:      tt.strict,
			}

			err := d.EraseInDisplay(tt.param)

			if tt.wantErr {
				be.Err(t, err)
			} else {
				be.Err(t, err, nil)
			}
			be.Equal(t, d.x, tt.wantX)
			be.Equal(t, d.y, tt.wantY)
		})
	}
}

func TestEraseInDisplay_OutOfBoundsSafety(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		x      int
		y      int
		buffer [][]cell
		param  []int
	}{
		{
			name:   "erase0 with y past buffer end",
			x:      0,
			y:      5,
			buffer: [][]cell{{{Char: 'a'}}, {{Char: 'b'}}},
			param:  []int{0},
		},
		{
			name:   "erase1 with y past buffer end",
			x:      0,
			y:      10,
			buffer: [][]cell{{{Char: 'a'}}},
			param:  []int{1},
		},
		{
			name:   "erase0 with empty buffer",
			x:      2,
			y:      0,
			buffer: [][]cell{},
			param:  []int{0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := &Decoder{x: tt.x, y: tt.y, buffer: tt.buffer}
			err := d.EraseInDisplay(tt.param)
			be.Err(t, err, nil)
		})
	}
}

func TestEraseInLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		param    []int
		initialX int
		initialY int
		buf      [][]cell
		line     []cell
		strict   bool
		wantLen  int
		wantErr  bool
	}{
		{
			name:     "erase 0 (cursor to EOL)",
			param:    []int{0},
			initialX: 2,
			initialY: 0,
			buf:      [][]cell{{{Char: 'a'}, {Char: 'b'}, {Char: 'c'}, {Char: 'd'}}},
			line:     []cell{{Char: 'a'}, {Char: 'b'}, {Char: 'c'}, {Char: 'd'}},
			wantLen:  2,
			wantErr:  false,
		},
		{
			name:     "erase 2 (entire line)",
			param:    []int{2},
			initialX: 1,
			initialY: 0,
			buf:      [][]cell{{{Char: 'a'}, {Char: 'b'}}},
			line:     []cell{{Char: 'a'}, {Char: 'b'}},
			wantLen:  0,
			wantErr:  false,
		},
		{
			name:     "out of bounds y safety",
			param:    []int{0},
			initialX: 1,
			initialY: 10, // y is out of bounds (len is 1)
			buf:      [][]cell{{{Char: 'a'}}},
			line:     []cell{{Char: 'a'}},
			wantLen:  1,
			wantErr:  false,
		},
		{
			name:     "invalid param in strict mode errors",
			param:    []int{99},
			initialX: 0,
			initialY: 0,
			buf:      [][]cell{{{Char: 'a'}}},
			line:     []cell{{Char: 'a'}},
			strict:   true,
			wantLen:  1,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := &Decoder{
				x:           tt.initialX,
				y:           tt.initialY,
				buffer:      tt.buf,
				currentLine: tt.line,
				strict:      tt.strict,
			}

			err := d.EraseInLine(tt.param)

			if tt.wantErr {
				be.Err(t, err)
			} else {
				be.Err(t, err, nil)
			}

			be.Equal(t, len(d.currentLine), tt.wantLen)
		})
	}
}

func TestSaveRestoreCursorPosition(t *testing.T) {
	t.Parallel()

	t.Run("save and restore cursor", func(t *testing.T) {
		t.Parallel()
		d := &Decoder{x: 12, y: 34}

		_ = d.SaveCursorPosition(nil)
		d.x, d.y = 0, 0 // move cursor elsewhere

		_ = d.RestoreCursorPosition(nil)
		be.Equal(t, d.x, 12)
		be.Equal(t, d.y, 34)
	})

	t.Run("strict error on params", func(t *testing.T) {
		t.Parallel()
		d := &Decoder{strict: true}

		errSave := d.SaveCursorPosition([]int{1})
		be.Err(t, errSave)

		errRestore := d.RestoreCursorPosition([]int{1})
		be.Err(t, errRestore)
	})
}

func TestApplyCSI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		final   byte
		params  []int
		strict  bool
		wantErr bool
	}{
		{"valid CSI A (Up)", 'A', []int{1}, false, false},
		{"valid CSI H (Position)", 'H', []int{1, 1}, false, false},
		{"valid CSI f (Position alternative)", 'f', []int{1, 1}, false, false},
		{"unhandled final byte non-strict", 'Z', []int{}, false, false},
		{"unhandled final byte strict errors", 'Z', []int{}, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := &Decoder{strict: tt.strict}
			err := d.ApplyCSI(tt.final, tt.params)

			if tt.wantErr {
				be.Err(t, err)
			} else {
				be.Err(t, err, nil)
			}
		})
	}
}

func TestAmigaFixes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []byte
		want  []byte
	}{
		{
			name:  "fix broken CSI sequence double bracket",
			input: []byte{0x1b, '[', 0x1b, '[', '3', '1', 'm'},
			want:  []byte{0x1b, '[', '3', '1', 'm'},
		},
		{
			name:  "fix missing m on color sequence",
			input: []byte{0x1b, '[', '3', '4', ' '},
			want:  []byte{0x1b, '[', '3', '4', 'm', ' '},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := &Decoder{}
			r, err := d.amigaFixes(bytes.NewReader(tt.input))
			be.Err(t, err, nil)

			got, err := io.ReadAll(r)
			be.Err(t, err, nil)
			be.Equal(t, string(got), string(tt.want))
		})
	}
}

func TestParseCSI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		strict   bool
		initWrap bool
		wantWrap bool
		wantErr  bool
		wantX    int
		wantY    int
	}{
		{
			name:     "EOF right after ESC",
			input:    "",
			wantWrap: false,
		},
		{
			name:    "invalid character after ESC in strict mode",
			input:   "X",
			strict:  true,
			wantErr: true,
		},
		{
			name:     "invalid character after ESC non-strict mode",
			input:    "X",
			strict:   false,
			wantWrap: false,
		},
		{
			name:     "case '=' and setmode line wrap enable ('=7h')",
			input:    "[=7h",
			initWrap: false,
			wantWrap: true,
		},
		{
			name:     "case '=' and setmode line wrap disable ('=7l')",
			input:    "[=7l",
			initWrap: true,
			wantWrap: false,
		},
		{
			name:     "case '?' (private sequence ignored)",
			input:    "[?25h",
			initWrap: false,
			wantWrap: false,
		},
		{
			name:  "case ' ' ignored inside parameters",
			input: "[ 1 ; 2 A",
			wantX: 0,
			wantY: 0,
		},
		{
			name:  "case ';' with param in progress (multiple params)",
			input: "[5;10H",
			wantX: 9,
			wantY: 4,
		},
		{
			name:    "case ';' without param in progress in strict mode",
			input:   "[;5H",
			strict:  true,
			wantErr: true,
		},
		{
			name:   "case ';' without param in progress non-strict mode",
			input:  "[;5H",
			strict: false,
			wantX:  4,
		},
		{
			name:     "SGR sequence ('m')",
			input:    "[31m",
			wantWrap: false,
		},
		{
			name:  "CSI Cursor Forward ('C')",
			input: "[3C",
			wantX: 3,
		},
		{
			name:    "strict mode error from ApplyCSI (unknown command)",
			input:   "[1Z",
			strict:  true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := NewDecoder(nil)
			d.strict = tt.strict
			br := bufio.NewReader(strings.NewReader(tt.input))

			lineWrap := tt.initWrap
			initialAttr := Attribute{}

			_, err := d.parseCSI(br, initialAttr, &lineWrap)

			if tt.wantErr {
				be.Err(t, err)
			} else {
				be.Err(t, err, nil)
			}

			be.Equal(t, lineWrap, tt.wantWrap)
			if tt.wantX != 0 {
				be.Equal(t, d.x, tt.wantX)
			}
			if tt.wantY != 0 {
				be.Equal(t, d.y, tt.wantY)
			}
		})
	}
}

func TestApplySGR(t *testing.T) {
	t.Parallel()

	// Dummy palette for test verification
	var pal Palette

	tests := []struct {
		name    string
		params  []int
		initial Attribute
		want    Attribute
	}{
		{
			name:    "empty params resets to default",
			params:  []int{},
			initial: Attribute{Bold: true, Underline: true},
			want:    defaultAttr(pal),
		},
		{
			name:    "reset code (0)",
			params:  []int{Reset},
			initial: Attribute{Bold: true, Inverse: true},
			want:    defaultAttr(pal),
		},
		{
			name:    "bold enable and disable",
			params:  []int{Bold, NotBold},
			initial: Attribute{},
			want:    Attribute{Bold: false},
		},
		{
			name:    "underline and invert",
			params:  []int{Underline, Invert},
			initial: Attribute{},
			want:    Attribute{Underline: true, Inverse: true},
		},
		{
			name:    "reset underline and invert",
			params:  []int{NotUnderline, NotInvert},
			initial: Attribute{Underline: true, Inverse: true},
			want:    Attribute{Underline: false, Inverse: false},
		},
		{
			name:    "default FG and BG",
			params:  []int{DefaultFG, DefaultBG},
			initial: Attribute{FG: "#123456", BG: "#654321"},
			want:    Attribute{FG: "", BG: ""},
		},
		{
			name:    "basic foreground color",
			params:  []int{FG1st + 1}, // Red (31)
			initial: Attribute{},
			want:    Attribute{FG: BasicHex(1, false, pal)},
		},
		{
			name:    "basic background color",
			params:  []int{BG1st + 2}, // Green (42)
			initial: Attribute{},
			want:    Attribute{BG: BasicHex(2, false, pal)},
		},
		{
			name:    "bright foreground and background",
			params:  []int{BrightFG1st + 3, BrightBG1st + 4},
			initial: Attribute{},
			want:    Attribute{FG: BasicHex(3, true, pal), BG: BasicHex(4, true, pal)},
		},
		{
			name:    "256-color mode (SetFG mode 5)",
			params:  []int{SetFG, 5, 196}, // 38;5;196
			initial: Attribute{},
			want:    Attribute{FG: XtermHex(196, pal)},
		},
		{
			name:    "256-color mode (SetBG mode 5)",
			params:  []int{SetBG, 5, 82}, // 48;5;82
			initial: Attribute{},
			want:    Attribute{BG: XtermHex(82, pal)},
		},
		{
			name:    "RGB true-color mode (SetFG mode 2)",
			params:  []int{SetFG, 2, 255, 128, 64}, // 38;2;255;128;64
			initial: Attribute{},
			want:    Attribute{FG: RGBHex([]int{SetFG, 2, 255, 128, 64}, 0)},
		},
		{
			name:    "malformed 256-color sequence (truncated params)",
			params:  []int{SetFG, 5}, // missing color byte, should skip gracefully
			initial: Attribute{},
			want:    Attribute{},
		},
		{
			name:    "chained multiple SGR attributes",
			params:  []int{Bold, FG1st + 1, SetBG, 5, 200},
			initial: Attribute{},
			want: Attribute{
				Bold: true,
				FG:   BasicHex(1, false, pal),
				BG:   XtermHex(200, pal),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ApplySGR(tt.params, tt.initial, pal)
			be.Err(t, err, nil)
			be.Equal(t, got, tt.want)
		})
	}
}

func TestRGBHex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		params []int
		index  int
		want   string
	}{
		{
			name:   "standard pure red at index 0",
			params: []int{38, 2, 255, 0, 0},
			index:  0,
			want:   "ff0000",
		},
		{
			name:   "standard pure green at index 0",
			params: []int{38, 2, 0, 255, 0},
			index:  0,
			want:   "00ff00",
		},
		{
			name:   "custom color with offset index",
			params: []int{1, 38, 2, 18, 52, 86},
			index:  1,
			want:   "123456",
		},
		{
			name:   "clamp values above 255",
			params: []int{38, 2, 300, 500, 256},
			index:  0,
			want:   "ffffff",
		},
		{
			name:   "clamp negative values to 0",
			params: []int{38, 2, -10, -50, -1},
			index:  0,
			want:   "000000",
		},
		{
			name:   "insufficient parameters (truncated slice)",
			params: []int{38, 2, 255, 255},
			index:  0,
			want:   "",
		},
		{
			name:   "index out of bounds",
			params: []int{38, 2, 255, 255, 255},
			index:  2,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := RGBHex(tt.params, tt.index)
			be.Equal(t, got, tt.want)
		})
	}
}
