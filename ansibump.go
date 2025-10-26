// Package ansibump converts ANSI escape sequences such as colors,
// cursor movements, and character deletions, into a HTML representation.
package ansibump

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"html"
	"html/template"
	"io"
	"slices"
	"strings"

	"golang.org/x/text/encoding/charmap"
)

var (
	ErrReader     = errors.New("reader is nil")
	ErrUnexpected = errors.New("unexpected parameters")
	ErrRecognized = errors.New("unrecognised parameters")
	ErrParam      = errors.New("encountered ';' without parameter")
	ErrExpect0or1 = errors.New("expected 0 or 1 parameters")
	ErrExpect0or2 = errors.New("expected 0 or 2 parameters")
	ErrExpect1    = errors.New("expected 1 parameter")
	ErrUnknownCSI = errors.New("unrecognized CSI final byte")
	ErrUnknownCtr = errors.New("unrecognized control byte")
	ErrUnknownEsc = errors.New("unrecognized ESC sequence after ESC")
)

const (
	NUL = 0x00 // NUL is an ASCII null character
	EOF = 0x1a // EOF is the MS-DOS end-of-file character value
	ESC = 0x1b // ESC is the escape control character code

	Reset        = 0
	Bold         = 1
	NotBold      = 21
	NotBoldFaint = 22
	Underline    = 4
	NotUnderline = 24
	Invert       = 7
	NotInvert    = 27
	DefaultFG    = 39
	DefaultBG    = 49
	FG1st        = 30
	FGEnd        = 37
	BG1st        = 40
	BGEnd        = 47
	BrightFG1st  = 90
	BrightFGEnd  = 97
	BrightBG1st  = 100
	BrightBGEnd  = 107
	SetFG        = 38
	SetBG        = 48
)

// Palette sets the ANSI 4-bit color codes to a colorset of RGB values.
// The ANSI standard never formalized color values and it was left to the system to determine.
// Wikipedia has a [useful table] of the common palettes.
//
// [useful table]: https://en.wikipedia.org/wiki/ANSI_escape_code#3-bit_and_4-bit
type Palette uint

const (
	CGA16   Palette = iota // Color Graphics Adapter colorset defined by IBM for the PC in 1981
	Xterm16                // Xterm terminal emulator program for the X Window System colorset from the mid-1980s
)

// Color code represented as hexadecimal numeric value.
// These are often 6 digit values RRGGBB (red, green, blue),
// however, certain values can be shortened to 3 digit values.
//
// For example, the code of CGA red "aa0000" (red: aa, green: 00, blue: 00) can shortened to "a00".
type Color string

const (
	CBlack    Color = "000"    // black
	CRed      Color = "a00"    // red
	CGreen    Color = "0a0"    // green
	CBrown    Color = "a50"    // yellow
	CBlue     Color = "00a"    // blue
	CMagenta  Color = "a0a"    // magenta
	CCyan     Color = "0aa"    // cyan
	CGray     Color = "aaa"    // white
	CDarkGray Color = "555"    // bright black
	CLRed     Color = "f55"    // bright red
	CLGreen   Color = "5f5"    // bright green
	CYellow   Color = "ff5"    // bright yellow
	CLBlue    Color = "55f"    // bright blue
	CLMagenta Color = "f5f"    // bright magenta
	CLCyan    Color = "5ff"    // bright cyan
	CWhite    Color = "fff"    // bright white
	XBlack    Color = "000"    // black
	XMarron   Color = "800000" // red
	XGreen    Color = "008000" // green
	XOlive    Color = "808000" // yellow
	XNavy     Color = "000080" // blue
	XPurple   Color = "800080" // magenta
	XTeal     Color = "008080" // cyan
	XSilver   Color = "c0c0c0" // white
	XGray     Color = "808080" // bright black
	XRed      Color = "f00"    // bright red
	XLime     Color = "0f0"    // bright green
	XYellow   Color = "ff0"    // bright yellow
	XBlue     Color = "00f"    // bright blue
	XFuchsia  Color = "f5f"    // bright magenta
	XAqua     Color = "0ff"    // bright cyan
	XWhite    Color = "fff"    // bright white
)

// BG returns the CSS background-color property and color value.
func (c Color) BG() string {
	if c == "" {
		return ""
	}
	return "background-color:#" + string(c) + ";"
}

// FG returns the CSS color property and color value.
func (c Color) FG() string {
	if c == "" {
		return ""
	}
	return "color:#" + string(c) + ";"
}

func CGA() [16]Color {
	return [16]Color{
		CBlack, CRed, CGreen, CBrown, CBlue, CMagenta, CCyan, CGray,
		CDarkGray, CLRed, CLGreen, CYellow, CLBlue, CLMagenta, CLCyan, CWhite,
	}
}

func Xterm() [16]Color {
	return [16]Color{
		XBlack, XMarron, XGreen, XOlive, XNavy, XPurple, XTeal, XSilver,
		XGray, XRed, XLime, XYellow, XBlue, XFuchsia, XAqua, XWhite,
	}
}

// Attribute describes styling for a single character cell.
type Attribute struct {
	FG        string // FG is a foreground hex color like "rrggbb" or (no leading #) or empty for default
	BG        string // BG is a background hex color like "rrggbb"
	Bold      bool   // Bold toggles a lighter color variation
	Underline bool   // Underline toggles a underline text decoration
	Inverse   bool   // Inverse swaps the background and foreground colors
}

// Decoder maintains the screen buffer and cursor state while parsing ANSI.
type Decoder struct {
	charset        *charmap.Charmap
	palette        Palette
	buffer         [][]cell
	currentLine    []cell
	x, y           int
	savedX, savedY int
	width          int
	defaultFG      Color
	defaultBG      Color
	strict         bool
}

// cell in the output buffer
type cell struct {
	Attr Attribute
	Char string
}

// NewDecoder creates a Decoder with a given width (columns). If width <= 0, 80 is used.
//
// Palette can either be CGA16 or Xterm16.
//
// Generally the charset of ANSI art should be [charmap.CodePage437],
// however artworks for the Commodore Amiga can be [charmap.ISO8859_1].
// Modern artworks or terminal text will usually be in UTF-8 encoding
// which can be set with charset as a nil value or charset as [charmap.XUserDefined].
//
// Strict is a debug mode that will throw errors when the ANSI includes malformed
// and invalid data or values.
func NewDecoder(width int, strict bool, pal Palette, charset *charmap.Charmap) *Decoder {
	if width <= 0 {
		width = 80
	}
	if charset == nil {
		charset = charmap.XUserDefined
	}
	d := &Decoder{
		charset:   charset,
		palette:   pal,
		buffer:    [][]cell{{}},
		x:         0,
		y:         0,
		width:     width,
		defaultFG: CGray,
		defaultBG: CBlack,
		strict:    strict,
	}
	d.currentLine = d.buffer[0]
	return d
}

// Buffer creates a new Buffer containing the HTML elements of the ANSI encoded text
// found in the Reader.
//
// The other arguments are used by the [NewDecoder] which documents their purpose.
func Buffer(r io.Reader, width int, strict bool, pal Palette, charset *charmap.Charmap) (*bytes.Buffer, error) {
	if r == nil {
		return nil, ErrReader
	}
	if charset == nil {
		charset = charmap.XUserDefined
	}
	d := NewDecoder(width, strict, pal, charset)
	if err := d.Read(r); err != nil {
		return nil, err
	}
	var b bytes.Buffer
	out := bufio.NewWriter(&b)
	if err := d.Write(out); err != nil {
		return nil, err
	}
	if err := out.Flush(); err != nil {
		return nil, fmt.Errorf("buffer out flush: %w", err)
	}
	return &b, nil
}

// Bytes returns the HTML elements of the ANSI encoded text found in the Reader.
// It assumes the Reader is using IBM Code Page 437 encoding.
// If width is <= 0, an 80 columns value is used.
func Bytes(r io.Reader, width int) ([]byte, error) {
	buf, err := Buffer(r, width, false, CGA16, charmap.CodePage437)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// String returns the HTML elements of the ANSI encoded text found in the Reader.
// It assumes the Reader is using IBM Code Page 437 encoding.
// If width is <= 0, an 80 columns value is used.
func String(r io.Reader, width int) (string, error) {
	buf, err := Buffer(r, width, false, CGA16, charmap.CodePage437)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// WriteTo writes to w the HTML elements of the ANSI encoded text found in the Reader.
// It assumes the Reader is using IBM Code Page 437 encoding.
// If width is <= 0, an 80 columns value is used.
//
// The return int64 is the number of bytes written.
func WriteTo(r io.Reader, w io.Writer, width int) (int64, error) {
	buf, err := Buffer(r, width, false, CGA16, charmap.CodePage437)
	if err != nil {
		return 0, err
	}
	i, err := buf.WriteTo(w)
	if err != nil {
		return 0, fmt.Errorf("buffer write to: %w", err)
	}
	return i, nil
}

// Write writes to w the full HTML fragment with outer div using default colors and inner lines joined with newlines.
func (d *Decoder) Write(w io.Writer) error {
	if w == nil {
		w = io.Discard
	}
	lines := d.Lines()
	// build default color values if possible: fallback to defaults in Decoder
	defFg := d.defaultFG
	defBg := d.defaultBG
	t, err := template.New("ansi").Parse(
		`{{define "T"}}<div style="` + defFg.FG() + defBg.BG() + `">{{ . }}</div>{{end}}`)
	if err != nil {
		return fmt.Errorf("write template parse: %w", err)
	}
	if err := t.ExecuteTemplate(w, "T",
		template.HTML(strings.Join(lines, "\n"))); err != nil { //nolint:gosec
		return fmt.Errorf("write template execute: %w", err)
	}
	return nil
}

// Lines renders each buffer line into a single HTML string.
// Each contiguous run of identical attributes is wrapped in a <span style="...">.
func (d *Decoder) Lines() []string {
	out := []string{}
	for _, line := range d.buffer {
		if len(line) == 0 {
			out = append(out, "")
			continue
		}
		var lastAttr *Attribute
		var run []string
		var spans []struct {
			Attr Attribute
			Text string
		}
		for _, c := range line {
			if lastAttr == nil || !attrEqual(*lastAttr, c.Attr) {
				if len(run) > 0 && lastAttr != nil {
					spans = append(spans, struct {
						Attr Attribute
						Text string
					}{Attr: *lastAttr, Text: strings.Join(run, "")})
				}
				tmp := c.Attr
				lastAttr = &tmp
				run = []string{}
			}
			run = append(run, c.Char)
		}
		if len(run) > 0 && lastAttr != nil {
			spans = append(spans, struct {
				Attr Attribute
				Text string
			}{Attr: *lastAttr, Text: strings.Join(run, "")})
		}
		// Build HTML for line
		var b strings.Builder
		for _, sp := range spans {
			style := buildStyle(sp.Attr)
			b.WriteString(`<span style="`)
			b.WriteString(html.EscapeString(style))
			b.WriteString(`">`)
			// escape text but preserve spaces
			b.WriteString(html.EscapeString(sp.Text))
			b.WriteString(`</span>`)
		}
		out = append(out, b.String())
	}
	return out
}

// Read reads bytes from r and interprets ANSI sequences, updating the buffer.
func (d *Decoder) Read(r io.Reader) error { //nolint:gocyclo,gocognit
	br := bufio.NewReader(r)
	// current attribute applied to subsequent characters
	cur := defaultAttr()
	const space = ' '
	pcdos := d.charset == charmap.CodePage437
	for {
		b, err := br.ReadByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("play byte reader: %w", err)
		}
		if b >= space {
			d.writeChar(b, cur)
			continue
		}
		switch b {
		case '\n':
			d.newline()
		case '\r', NUL:
			// ignore NUL and CR
			continue
		case EOF:
			return nil
		case ESC:
			nb, err := br.ReadByte()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return fmt.Errorf("play sequence reader: %w", err)
			}
			if nb != '[' {
				// We only handle CSI sequences (ESC [ ... )
				if d.strict {
					return fmt.Errorf("%w: %q", ErrUnknownEsc, nb)
				}
				continue
			}
			// parse CSI
			params := []int{}
			paramInProgress := false
			paramVal := 0
			private := false
			for {
				cb, err := br.ReadByte()
				if err == io.EOF {
					// truncated sequence
					break
				}
				if err != nil {
					return fmt.Errorf("play character reader: %w", err)
				}
				if '0' <= cb && cb <= '9' {
					if !paramInProgress {
						paramInProgress = true
						paramVal = int(cb - '0')
					} else {
						paramVal = paramVal*10 + int(cb-'0') //nolint:mnd
					}
					continue
				}
				if cb == ';' {
					if paramInProgress {
						params = append(params, paramVal)
						paramInProgress = false
						paramVal = 0
					} else {
						// ;; without number
						if d.strict {
							return ErrParam
						}
						params = append(params, -1)
					}
					continue
				}
				if cb == ' ' {
					continue
				}
				if cb == '?' {
					private = true
					continue
				}
				// final byte of CSI
				if paramInProgress {
					params = append(params, paramVal)
				}
				if !private && cb == 'm' {
					// SGR sequence: can be complex (including 38/48 extended)
					newAttr, err := ApplySGR(params, cur, d.palette)
					if err != nil {
						return err
					}
					cur = newAttr
				}
				if !private && cb != 'm' {
					// other CSI sequences that affect cursor / buffer
					if err := d.ApplyCSI(cb, params); err != nil {
						return err
					}
				}
				break
			}
		default:
			if pcdos {
				d.writeChar(b, cur)
				continue
			}
			// control codes like BEL, VT, etc. Ignore unless remap required.
			if d.strict {
				return fmt.Errorf("%w: 0x%02x", ErrUnknownCtr, b)
			}
			d.writeChar(byte(' '), cur)
		}
	}
	return nil
}

// CursorUp moves cursor up.
// Attr: CUU.
func (d *Decoder) CursorUp(params []int) error {
	if len(params) == 0 {
		n := d.y - 1
		d.setCursor(nil, &n)
		return nil
	}
	if len(params) == 1 {
		n := d.y - params[0]
		d.setCursor(nil, &n)
		return nil
	}
	if d.strict {
		return fmt.Errorf("CUU A: %w: %d", ErrExpect0or1, params)
	}
	return nil
}

// CursorDown moves cursor down.
// Attr: CUD.
func (d *Decoder) CursorDown(params []int) error {
	if len(params) == 0 {
		n := d.y + 1
		d.setCursor(nil, &n)
		return nil
	}
	if len(params) == 1 {
		n := d.y + params[0]
		d.setCursor(nil, &n)
		return nil
	}
	if d.strict {
		return fmt.Errorf("CUD B: %w: %d", ErrExpect0or1, params)
	}
	return nil
}

// CursorForward moves cursor forward.
// Attr: CUF.
func (d *Decoder) CursorForward(params []int) error {
	if len(params) == 0 {
		n := d.x + 1
		d.setCursor(&n, nil)
		return nil
	}
	if len(params) == 1 {
		n := d.x + params[0]
		d.setCursor(&n, nil)
		return nil
	}
	if d.strict {
		return fmt.Errorf("CUF C: %w: %d", ErrExpect0or1, params)
	}
	return nil
}

// CursorBack moves cursor back.
// Attr: CUB.
func (d *Decoder) CursorBack(params []int) error {
	if len(params) == 0 {
		n := d.x - 1
		d.setCursor(&n, nil)
		return nil
	}
	if len(params) == 1 {
		n := d.x - params[0]
		d.setCursor(&n, nil)
		return nil
	}
	if d.strict {
		return fmt.Errorf("CUB D: %w: %d", ErrExpect0or1, params)
	}
	return nil
}

// CursorNextLine moves cursor down to the beginning of the line.
// Attr: CNL.
func (d *Decoder) CursorNextLine(params []int) error {
	if len(params) == 0 {
		n := d.y + 1
		d.setCursor(ptrInt(0), &n)
		return nil
	}
	if len(params) == 1 {
		n := d.y + params[0]
		d.setCursor(ptrInt(0), &n)
		return nil
	}
	if d.strict {
		return fmt.Errorf("CNL E: %w: %d", ErrExpect0or1, params)
	}
	return nil
}

// CursorPreviousLine moves cursor up to the beginning of the line.
// Attr: CPL.
func (d *Decoder) CursorPreviousLine(params []int) error {
	if len(params) == 0 {
		n := d.y - 1
		d.setCursor(ptrInt(0), &n)
		return nil
	}
	if len(params) == 1 {
		n := d.y - params[0]
		d.setCursor(ptrInt(0), &n)
		return nil
	}
	if d.strict {
		return fmt.Errorf("CPL F: %w: %d", ErrExpect0or1, params)
	}
	return nil
}

// CursorHorizontalAbsolute moves the cursor to column.
// Attr: CHA.
func (d *Decoder) CursorHorizontalAbsolute(params []int) error {
	if len(params) == 1 {
		n := params[0]
		d.setCursor(&n, nil)
		return nil
	}
	if d.strict {
		return fmt.Errorf("CHA G: %w: %d", ErrExpect1, params)
	}
	return nil
}

// CursorPosition moves the cursor to row and column.
// Attr: CUP.
func (d *Decoder) CursorPosition(params []int) error {
	if len(params) == 0 {
		x := 0
		y := 0
		d.setCursor(&x, &y)
		return nil
	}
	const pair = 2
	if len(params) == pair {
		x := params[1] - 1
		y := params[0] - 1
		d.setCursor(&x, &y)
		return nil
	}
	if d.strict {
		return fmt.Errorf("CUP H/f: %w: %d", ErrExpect0or2, params)
	}
	// return nil
	if len(params) == 1 {
		y := params[0] - 1
		x := 0
		d.setCursor(&x, &y)
		return nil
	}
	return nil
}

// EraseInDisplay clears part of the screen.
// Attr: ED.
func (d *Decoder) EraseInDisplay(params []int) error {
	// 0 or empty: from cursor to end of screen
	cursorToEOS := len(params) == 0 || (len(params) == 1 && params[0] == 0)
	if cursorToEOS {
		// truncate current line from cursor onward
		if d.x < len(d.currentLine) {
			d.currentLine = d.currentLine[:d.x]
		}
		// truncate lines below current
		if d.y+1 < len(d.buffer) {
			d.buffer = d.buffer[:d.y+1]
		}
		d.buffer[d.y] = d.currentLine
		return nil
	}
	// erase up to cursor (from top to cursor)
	fromTop := len(params) == 1 && params[0] == 1
	if fromTop {
		for i := range d.y {
			d.buffer[i] = []cell{}
		}
		if d.x < len(d.currentLine) {
			for i := 0; i <= d.x && i < len(d.currentLine); i++ {
				d.currentLine[i] = cell{Attr: defaultAttr(), Char: " "}
			}
		} else {
			d.currentLine = []cell{}
		}
		d.buffer[d.y] = d.currentLine
		return nil
	}
	// erase entire screen
	entireScreen := len(params) == 1 && params[0] == 2 //nolint:mnd
	if entireScreen {
		for i := range d.buffer {
			d.buffer[i] = []cell{}
		}
		d.currentLine = []cell{}
		d.x = 0
		d.y = 0
		return nil
	}
	if d.strict {
		return fmt.Errorf("ED J: %w: %d", ErrRecognized, params)
	}
	return nil
}

// EraseInLine part of the line.
// Attr: EL.
func (d *Decoder) EraseInLine(params []int) error {
	// 0 or empty: from cursor to end of line
	cursorToEOL := len(params) == 0 || (len(params) == 1 && params[0] == 0)
	if cursorToEOL {
		if d.x < len(d.currentLine) {
			d.currentLine = d.currentLine[:d.x]
		}
		d.buffer[d.y] = d.currentLine
		return nil
	}
	// erase up to cursor in line
	cursorInLine := len(params) == 1 && params[0] == 1
	if cursorInLine {
		if d.x < len(d.currentLine) {
			for i := 0; i <= d.x && i < len(d.currentLine); i++ {
				d.currentLine[i] = cell{Attr: defaultAttr(), Char: " "}
			}
		} else {
			d.currentLine = []cell{}
		}
		d.buffer[d.y] = d.currentLine
		return nil
	}
	// erase entire line
	entireLine := len(params) == 1 && params[0] == 2 //nolint:mnd
	if entireLine {
		d.currentLine = []cell{}
		d.buffer[d.y] = d.currentLine
	}
	if d.strict {
		return fmt.Errorf("EL K: %w: %d", ErrRecognized, params)
	}
	return nil
}

// SaveCursorPosition saves the cursor state for later use.
// Abbr: RCP, SCORC.
func (d *Decoder) SaveCursorPosition(params []int) error {
	if len(params) != 0 && d.strict {
		return fmt.Errorf("SCP s: %w: %d", ErrUnexpected, params)
	}
	d.savedX = d.x
	d.savedY = d.y
	return nil
}

// RestoreCursorPosition restores the saved cursor state.
// Abbr: SCP, SCOSC.
func (d *Decoder) RestoreCursorPosition(params []int) error {
	if len(params) != 0 && d.strict {
		return fmt.Errorf("RCP u: %w: %d", ErrUnexpected, params)
	}
	d.setCursor(&d.savedX, &d.savedY)
	return nil
}

// ApplyCSI handles cursor movement and erase sequences that alter the buffer or cursor.
// It follows standard ANSI/VT100 CSI final bytes used in the original request:
// A B C D E F G H f J K s u
func (d *Decoder) ApplyCSI(final byte, params []int) error {
	switch final {
	case 'A':
		return d.CursorUp(params)
	case 'B':
		return d.CursorDown(params)
	case 'C':
		return d.CursorForward(params)
	case 'D':
		return d.CursorBack(params)
	case 'E':
		return d.CursorNextLine(params)
	case 'F':
		return d.CursorPreviousLine(params)
	case 'G':
		return d.CursorHorizontalAbsolute(params)
	case 'H', 'f':
		return d.CursorPosition(params)
	case 'J':
		return d.EraseInDisplay(params)
	case 'K':
		return d.EraseInLine(params)
	case 's':
		return d.SaveCursorPosition(params)
	case 'u':
		return d.RestoreCursorPosition(params)
	default:
		if d.strict {
			return fmt.Errorf("%w: %c", ErrUnknownCSI, final)
		}
	}
	return nil
}

// defaultAttr returns the default Attribute (no styles).
func defaultAttr() Attribute {
	return Attribute{FG: "", BG: "", Bold: false, Underline: false, Inverse: false}
}

// ApplySGR applies SGR parameters to an incoming attribute and returns a new Attribute.
func ApplySGR(params []int, cur Attribute, pal Palette) (Attribute, error) { //nolint:gocyclo,gocognit
	attr := cur // start from current
	if len(params) == 0 {
		// treat empty SGR as reset per common implementations
		return defaultAttr(), nil
	}
	const xterm, truecolor = 5, 2
	i := Reset
	for i < len(params) {
		p := params[i]
		switch {
		case p == Reset:
			attr = defaultAttr()
		case p == Bold:
			attr.Bold = true
		case p == NotBold || p == NotBoldFaint:
			attr.Bold = false
		case p == Underline:
			attr.Underline = true
		case p == NotUnderline:
			attr.Underline = false
		case p == Invert:
			attr.Inverse = true
		case p == NotInvert:
			attr.Inverse = false
		case p == DefaultFG:
			attr.FG = ""
		case p == DefaultBG:
			attr.BG = ""
		case FG1st <= p && p <= FGEnd:
			attr.FG = BasicHex(p-FG1st, false, pal)
		case BG1st <= p && p <= BGEnd:
			attr.BG = BasicHex(p-BG1st, false, pal)
		case BrightFG1st <= p && p <= BrightFGEnd:
			attr.FG = BasicHex(p-BrightFG1st, true, pal)
		case BrightBG1st <= p && p <= BrightBGEnd:
			attr.BG = BasicHex(p-BrightBG1st, true, pal)
		case p == SetFG || p == SetBG:
			// extended color: either 5;n (256 color) or 2;r;g;b (truecolor)
			isFG := (p == SetFG)
			// look ahead
			if i+1 >= len(params) {
				if true {
					// malformed; ignore per permissive behavior
					i++
					continue
				}
			}
			mode := params[i+1]
			if mode == xterm {
				vals := 2
				// 256-color: next param is index
				if i+vals >= len(params) {
					// malformed, skip
					i += vals
					continue
				}
				idx := params[i+vals]
				if isFG {
					attr.FG = XtermHex(idx, pal)
				} else {
					attr.BG = XtermHex(idx, pal)
				}
				i += 3
				continue
			}
			if mode == truecolor {
				vals := 4
				if i+vals >= len(params) {
					i++
					continue
				}
				if isFG {
					attr.FG = RGB(params, i)
				} else {
					attr.BG = RGB(params, i)
				}
				i += 5
				continue
			}
			// unknown mode; skip the indicator
			i++
		default:
			// unhandled codes are ignored in permissive mode
			// TODO: throw an error in strict mode?
		}
		i++
	}
	// fmt.Printf("%+v\n", attr)
	return attr, nil
}

// RGB converts the params into a "true color", red, green, blue hex string.
func RGB(params []int, i int) string {
	if len(params) < i+4 {
		return ""
	}
	const hi = 255
	r := clamp(params[i+2], 0, hi)
	g := clamp(params[i+3], 0, hi)
	b := clamp(params[i+4], 0, hi)
	return fmt.Sprintf("%02x%02x%02x", r, g, b)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// BasicHex takes a standard color code and returns a corresponding hexadecimal string.
// When bright is toggled, a lighter color variant is used.
// Codes are values between 0 and 7, and any invalid codes returns a blank string.
//
//nolint:mnd
func BasicHex(code int, bright bool, p Palette) string {
	const first, last = 0, 7
	if code < first || code > last {
		return ""
	}
	index := code
	if bright {
		index = code + 8
	}
	switch p {
	case CGA16:
		return string(CGA()[index])
	case Xterm16:
		return string(Xterm()[index])
	}
	return ""
}

// XtermHex takes a Xterm color code and returns the corresponding RBG values
// as a hexadecimal string.
// Codes are values between 0 and 255, and any invalid codes return a blank string.
// The Palette is only used for basic colors codes between 0 and 7.
//
//nolint:mnd
func XtermHex(code int, p Palette) string {
	const hex = "%02x%02x%02x"
	if code < 0 || code > 255 {
		return ""
	}
	if code <= 7 {
		return BasicHex(code, false, p)
	}
	if code <= 15 {
		c := code - 8
		return BasicHex(c, true, p)
	}
	if code <= 255 {
		r, g, b := XtermColors(code)
		return fmt.Sprintf(hex, r, g, b)
	}
	return ""
}

// XtermColors takes a Xterm non-system color code and returns the corresponding RGB values.
// The code values begin at 16 and finish at 255.
// If a code is out of range, then the returned RGB values will be -1, which are invalid.
//
// Some helpful links, [256 colors cheat sheet], [Xterm Colors], and [8-bit colors wiki].
//
// [256 colors cheat sheet]: https://www.ditig.com/256-colors-cheat-sheet
// [Xterm Colors]: https://lucianofedericopereira.github.io/xterm-colors-cheat-sheet
// [8-bit colors wiki]: https://en.wikipedia.org/wiki/ANSI_escape_code#8-bit
func XtermColors(code int) (int, int, int) {
	if code >= 16 && code <= 231 {
		return XtermColor(code)
	}
	if code >= 232 && code <= 255 {
		return XtermGray(code)
	}
	return -1, -1, -1
}

// XtermColor returns the RGB values for non-system Xterm colors.
//
//nolint:mnd
func XtermColor(code int) (int, int, int) {
	c := code - 16
	r := c / 36
	g := (c % 36) / 6
	b := c % 6
	calc := func(c int) int {
		if c == 0 {
			return 0
		}
		return 55 + c*40
	}
	r = calc(r)
	g = calc(g)
	b = calc(b)
	return r, g, b
}

// XtermGray returns the RGB values for the Xterm greyscale colors.
//
//nolint:mnd
func XtermGray(code int) (int, int, int) {
	level := code - 232
	v := 8 + level*10
	return v, v, v
}

func attrEqual(a, b Attribute) bool {
	return a.FG == b.FG && a.BG == b.BG && a.Bold == b.Bold && a.Underline == b.Underline && a.Inverse == b.Inverse
}

// buildStyle takes the Attribute and returns a HTML style attribute.
func buildStyle(a Attribute) string {
	// Determine effective fg/bg respecting inverse
	fg := a.FG
	bg := a.BG
	if a.Inverse {
		fg, bg = bg, fg
	}
	parts := []string{}
	switch {
	case a.Bold && fg != "":
		val := Bright(Color(fg))
		parts = append(parts, val.FG())
	case a.Bold && fg == "":
		parts = append(parts, CWhite.FG())
	case fg != "":
		val := Color(fg)
		parts = append(parts, val.FG())
	case fg == "":
		parts = append(parts, CGray.FG())
	}
	if bg != "" {
		val := Color(bg)
		// if a.Bold {
		// 	val = Bright(val)
		// }
		parts = append(parts, val.BG())
	}
	if a.Underline {
		parts = append(parts, "text-decoration:underline;")
	}
	return strings.Join(parts, "")
}

// Bright takes a CGA color and swaps it for a lighter variant.
// For example, Color.CBlack (black) returns Color.CDarkGray (bright black).
//
//nolint:mnd
func Bright(c Color) Color {
	colors := make([]string, 16)
	for i, x := range CGA() {
		colors[i] = string(x)
	}
	match := slices.Index(colors, string(c))
	if match >= 0 && match <= 7 {
		return CGA()[match+8]
	}
	return ""
}

// --- Helpers for managing cursor and buffer ---

// setCursor sets x and/or y (nil means unchanged)
func (d *Decoder) setCursor(xp *int, yp *int) {
	if xp != nil {
		d.x = max(0, *xp)
	}
	if yp != nil {
		d.y = max(0, *yp)
	}
	d.ensureLine(d.y)
}

func (d *Decoder) ensureLine(y int) {
	for y >= len(d.buffer) {
		d.buffer = append(d.buffer, []cell{})
	}
	d.currentLine = d.buffer[y]
}

// newline moves cursor to start of next line
func (d *Decoder) newline() {
	d.setCursor(ptrInt(0), ptrInt(d.y+1))
}

// writeChar writes a printable character at the cursor location using given attribute.
func (d *Decoder) writeChar(b byte, attr Attribute) {
	ch := string(b)
	if d.charset != nil && d.charset != charmap.XUserDefined {
		ch = string(d.charset.DecodeByte(b))
	}
	// ensure current line exists
	d.ensureLine(d.y)
	// expand line with spaces if needed
	for len(d.currentLine) < d.x {
		d.currentLine = append(d.currentLine, cell{Attr: defaultAttr(), Char: " "})
	}
	if d.x < len(d.currentLine) {
		d.currentLine[d.x] = cell{Attr: attr, Char: ch}
	} else {
		d.currentLine = append(d.currentLine, cell{Attr: attr, Char: ch})
	}
	d.buffer[d.y] = d.currentLine
	d.x++
	if d.x >= d.width {
		d.newline()
	}
}

func ptrInt(v int) *int { return &v }
