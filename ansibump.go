// Package ansibump converts ANSI escape sequences such as colors,
// cursor movements, and character deletions, into a HTML representation.
package ansibump

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"html"
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
	DP2                    // DP2 is a Commodore Amiga era Deluxe Paint II colorset that mimics the colors of CGA16
)

// Color code represented as hexadecimal numeric value.
// These are often 6 digit values RRGGBB (red, green, blue),
// however, certain values can be shortened to 3 digit values.
//
// For example, the code of CGA red "aa0000" (red: aa, green: 00, blue: 00) can shortened to "a00".
type Color string

const (
	CBlack     Color = "000"    // cga black
	CRed       Color = "a00"    // red
	CGreen     Color = "0a0"    // green
	CBrown     Color = "a50"    // yellow
	CBlue      Color = "00a"    // blue
	CMagenta   Color = "a0a"    // magenta
	CCyan      Color = "0aa"    // cyan
	CGray      Color = "aaa"    // white
	CDarkGray  Color = "555"    // bright black
	CLRed      Color = "f55"    // bright red
	CLGreen    Color = "5f5"    // bright green
	CYellow    Color = "ff5"    // bright yellow
	CLBlue     Color = "55f"    // bright blue
	CLMagenta  Color = "f5f"    // bright magenta
	CLCyan     Color = "5ff"    // bright cyan
	CWhite     Color = "fff"    // bright white
	XBlack     Color = "000"    // xterm black
	XMarron    Color = "800000" // red
	XGreen     Color = "008000" // green
	XOlive     Color = "808000" // yellow
	XNavy      Color = "000080" // blue
	XPurple    Color = "800080" // magenta
	XTeal      Color = "008080" // cyan
	XSilver    Color = "c0c0c0" // white
	XGray      Color = "808080" // bright black
	XRed       Color = "f00"    // bright red
	XLime      Color = "0f0"    // bright green
	XYellow    Color = "ff0"    // bright yellow
	XBlue      Color = "00f"    // bright blue
	XFuchsia   Color = "f5f"    // bright magenta
	XAqua      Color = "0ff"    // bright cyan
	XWhite     Color = "fff"    // bright white
	DPBlack    Color = "000"    // deluxe paint ii black
	DPRed      Color = "a80000" // red
	DPGreen    Color = "008800" // green
	DPBrown    Color = "a85420" // brown
	DPBlue     Color = "0000fc" // blue
	DPMagenta  Color = "cc0088" // magenta
	DPCyan     Color = "00a8fc" // cyan
	DPGray     Color = "747474" // white
	DPDarkGray Color = "646464" // bright black
	DPLRed     Color = "ec0000" // bright red
	DPLGreen   Color = "88fc00" // bright green
	DPYellow   Color = "fcec00" // bright yellow
	DPLBlue    Color = "0074cc" // bright blue
	DPLMagenta Color = "cc00ec" // bright magenta
	DPLCyan    Color = "00dcdc" // bright cyan
	DPWhite    Color = "fcfcfc" // bright white
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

type Colors [16]Color

// DefaultFG will return the "white" variant of the color palette,
// which can be used as the default foreground color.
func (c Colors) DefaultFG() Color {
	const white = 7
	return c[white]
}

// DefaultBG will return the "black" variant of the color palette,
// which can be used as the default background color.
func (c Colors) DefaultBG() Color {
	const black = 0
	return c[black]
}

func CGA() Colors {
	return Colors{
		CBlack, CRed, CGreen, CBrown, CBlue, CMagenta, CCyan, CGray,
		CDarkGray, CLRed, CLGreen, CYellow, CLBlue, CLMagenta, CLCyan, CWhite,
	}
}

func Xterm() Colors {
	return Colors{
		XBlack, XMarron, XGreen, XOlive, XNavy, XPurple, XTeal, XSilver,
		XGray, XRed, XLime, XYellow, XBlue, XFuchsia, XAqua, XWhite,
	}
}

func DPaint2() Colors {
	return Colors{
		DPBlack, DPRed, DPGreen, DPBrown, DPBlue, DPMagenta, DPCyan, DPGray,
		DPDarkGray, DPLRed, DPLGreen, DPYellow, DPLBlue, DPLMagenta, DPLCyan, DPWhite,
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
	amigaParser    bool
	strict         bool
}

// cell in the output buffer.
type cell struct {
	Attr Attribute
	Char rune
}

// Customizer is optional, and is used to configure the parsing of the ANSI encoded text.
//
// Usually, the defaults work for most texts.
type Customizer struct {
	// Width is the number of columns of the ANSI encoded text.
	// If a value provided is <= 0, then a common an 80 columns value is used.
	Width int
	// The AmigaParser should be set to false except with edge cases where unusual
	// Commodore Amiga specific encodings are to be parsed.
	AmigaParser bool
	// Strict is a debug mode that will throw errors when the ANSI includes malformed and invalid data or values.
	Strict bool
	// Color Palette can either be CGA16, Xterm16, or DP2.
	//   - CGA16 is the default, it is the Color Graphics Adapter colorset defined by IBM for the PC in 1981.
	//   - Xterm16 is the Xterm terminal emulator program for the X Window System colorset from the mid-1980s.
	//   - DP2 is a Commodore Amiga era Deluxe Paint II colorset that mimics the colors of CGA16.
	Color Palette
	// CharSet is the simple character encoding used by the text.
	//
	// Generally the charset of ANSI art should be [charmap.CodePage437],
	// however artworks for the Commodore Amiga are often [charmap.ISO8859_1].
	// Modern artworks or terminal text will usually be in UTF-8 encoding
	// which can be set with CharSet = nil or CharSet = [charmap.XUserDefined].
	CharSet *charmap.Charmap
}

const DefaultColumns = 80

// NewDecoder creates a Decoder with an optional Customizer.
// If c is nil then default values are used.
func NewDecoder(c *Customizer) *Decoder {
	if c == nil {
		c = &Customizer{}
	}
	width := c.Width
	if width <= 0 {
		width = DefaultColumns
	}

	defaultCharSet := charmap.XUserDefined
	charset := c.CharSet
	if charset == nil {
		charset = defaultCharSet
	}

	singleEmptyRow := [][]cell{{}}

	var def style
	def.set(c.Color)

	d := &Decoder{
		charset:     charset,
		palette:     c.Color,
		buffer:      singleEmptyRow,
		x:           0,
		y:           0,
		width:       width,
		defaultFG:   def.fg,
		defaultBG:   def.bg,
		amigaParser: c.AmigaParser,
		strict:      c.Strict,
	}
	return d
}

// Bytes returns the HTML elements of the ANSI encoded text found in the Reader.
//
// The parser configurations and arguments are configured using the [Customizer].
func (c *Customizer) Bytes(r io.Reader) ([]byte, error) {
	const format = "customizer bytes %s: %w"
	if r == nil {
		return nil, fmt.Errorf(format, "argument", ErrReader)
	}

	d := NewDecoder(c)
	if err := d.Decode(r); err != nil {
		return nil, fmt.Errorf(format, "read", err)
	}

	var b bytes.Buffer
	if err := d.Write(&b); err != nil {
		return nil, fmt.Errorf(format, "write", err)
	}

	return b.Bytes(), nil
}

// String returns the HTML elements of the ANSI encoded text found in the Reader.
//
// The parser configurations and arguments are configured using the [Customizer].
func (c *Customizer) String(r io.Reader) (string, error) {
	const format = "customizer string %s: %w"
	if r == nil {
		return "", fmt.Errorf(format, "argument", ErrReader)
	}

	d := NewDecoder(c)
	if err := d.Decode(r); err != nil {
		return "", fmt.Errorf(format, "read", err)
	}

	var b bytes.Buffer
	if err := d.Write(&b); err != nil {
		return "", fmt.Errorf(format, "write", err)
	}

	return b.String(), nil
}

// Bytes returns the HTML elements of the ANSI encoded text found in the Reader.
// It assumes the Reader is using IBM Code Page 437 encoding.
// If width is <= 0, an 80 columns value is used.
func Bytes(r io.Reader, width int) ([]byte, error) {
	cust := Customizer{
		Width:       width,
		AmigaParser: false,
		Strict:      false,
		Color:       CGA16,
		CharSet:     charmap.CodePage437,
	}
	p, err := cust.Bytes(r)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// String returns the HTML elements of the ANSI encoded text found in the Reader.
// It assumes the Reader is using IBM Code Page 437 encoding.
// If width is <= 0, an 80 columns value is used.
func String(r io.Reader, width int) (string, error) {
	cust := Customizer{
		Width:       width,
		AmigaParser: false,
		Strict:      false,
		Color:       CGA16,
		CharSet:     charmap.CodePage437,
	}
	s, err := cust.String(r)
	if err != nil {
		return "", err
	}
	return s, nil
}

// WriteTo writes to w the HTML elements of the ANSI encoded text found in the Reader.
// It assumes the Reader is using IBM Code Page 437 encoding.
// If width is <= 0, an 80 columns value is used.
//
// The return int is the number of bytes written.
func WriteTo(r io.Reader, w io.Writer, width int) (int, error) {
	const format = "buffer write to: %w"
	cust := Customizer{
		Width:       width,
		AmigaParser: false,
		Strict:      false,
		Color:       CGA16,
		CharSet:     charmap.CodePage437,
	}
	p, err := cust.Bytes(r)
	if err != nil {
		return 0, err
	}
	i, err := w.Write(p)
	if err != nil {
		return 0, fmt.Errorf(format, err)
	}
	return i, nil
}

// Write writes to w the full HTML fragment with outer div using default colors and inner lines joined with newlines.
func (d *Decoder) Write(w io.Writer) error {
	if w == nil {
		w = io.Discard
	}

	var err error
	write := func(s string) {
		if err != nil {
			return
		}
		if _, wErr := io.WriteString(w, s); err != nil {
			err = fmt.Errorf("write: %w", wErr)
		}
	}

	lines := d.Lines(d.palette)
	defaultFG := d.defaultFG
	defaultBG := d.defaultBG

	write(`<div style="`)
	write(defaultFG.FG())
	write(defaultBG.BG())
	write(`">`)
	for i, line := range lines {
		if i > 0 {
			write("\n")
		}
		write(line)
	}
	write(`</div>`)
	return nil
}

// Lines renders each buffer line into a single HTML string.
// Each contiguous run of identical attributes is wrapped in a <span style="...">.
func (d *Decoder) Lines(pal Palette) []string {
	var defaults style
	defaults.set(pal)

	lines := make([]string, 0, len(d.buffer))
	for _, cells := range d.buffer {
		if len(cells) == 0 {
			lines = append(lines, "")
			continue
		}
		var line strings.Builder
		lastAttr := cells[0].Attr
		elems := make([]rune, 0, len(cells))

		flushSpan := func() {
			if len(elems) == 0 {
				return
			}
			style := buildStyle(lastAttr, defaults)
			line.WriteString(`<span style="`)
			line.WriteString(html.EscapeString(style))
			line.WriteString(`">`)
			// Idiomatic: cast []rune directly to string
			line.WriteString(html.EscapeString(string(elems)))
			line.WriteString(`</span>`)
			// reset the elems buffer, reusing the memory
			elems = elems[:0]
		}

		for _, cell := range cells {
			if lastAttr != cell.Attr {
				flushSpan()
				lastAttr = cell.Attr
			}
			elems = append(elems, cell.Char)
		}
		flushSpan()
		lines = append(lines, line.String())
	}

	return lines
}

// Decode reads bytes from r and interprets ANSI sequences, updating the buffer.
func (d *Decoder) Decode(r io.Reader) error { //nolint:cyclop,funlen
	const format = "decoder decode %s: %w"
	if d.amigaParser {
		var err error
		r, err = d.amigaFixes(r)
		if err != nil {
			return fmt.Errorf(format, "", err)
		}
	}

	br := bufio.NewReader(r)
	cur := defaultAttr(d.palette)

	const substr = "code page"
	codepage := strings.Contains(strings.ToLower(d.charset.String()), substr)
	lineWrap := false
	const space = ' '

	for {
		b, err := br.ReadByte()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf(format, "read byte", err)
		}

		if b >= space {
			d.writeChar(b, cur)
			continue
		}

		switch b {
		case '\n':
			if !lineWrap {
				d.newline()
			}
		case '\r', NUL:
			// ignore
		case EOF:
			return nil
		case ESC:
			newAttr, err := d.parseCSI(br, cur, &lineWrap)
			if err != nil {
				return err
			}
			cur = newAttr
		default:
			switch {
			case codepage:
				d.writeChar(b, cur)
			case d.strict:
				const format = "%w: 0x%02x"
				return fmt.Errorf(format, ErrUnknownCtr, b)
			default:
				d.writeChar(space, cur)
			}
		}
	}
}

func resetCell(param int, params []int) (bool, int, []int) {
	params = append(params, param)
	return false, 0, params
}

func inProgress(cb byte, pip bool, param int) (bool, int) {
	if !pip {
		return true, int(cb - '0')
	}
	value := param*10 + int(cb-'0') //nolint:mnd
	return pip, value
}

// setWrapping uses the non-standard, [Screen Modes] controls found in ANSI.SYS.
// Only enable line wrapping (ESC[=7h) and disable line wrapping are used (ESC[=7l).
// Setting graphics and text modes are skipped.
//
// [Screen Modes]: https://gist.github.com/ConnerWill/d4b6c776b509add763e17f9f113fd25b<F6>
func setWrapping(cb byte, linewrap, current bool) bool {
	if linewrap && cb == 'l' {
		return false
	}
	if linewrap && cb == 'h' {
		return true
	}
	return current
}

func wrapMode(cb byte) bool {
	return cb == '7'
}

// CursorUp moves cursor up.
// Attr: CUU.
func (d *Decoder) CursorUp(params []int) error {
	const format = "CUU A: %w: %v"

	if len(params) == 0 {
		n := d.y - 1
		d.setCursor(nil, &n)
		return nil
	}

	if len(params) == 1 {
		n := d.y - cursor(params)
		d.setCursor(nil, &n)
		return nil
	}

	if d.strict {
		return fmt.Errorf(format, ErrExpect0or1, params)
	}
	return nil
}

// CursorDown moves cursor down.
// Attr: CUD.
func (d *Decoder) CursorDown(params []int) error {
	const format = "CUD B: %w: %v"

	if len(params) == 0 {
		n := d.y + 1
		d.setCursor(nil, &n)
		return nil
	}

	if len(params) == 1 {
		n := d.y + cursor(params)
		d.setCursor(nil, &n)
		return nil
	}

	if d.strict {
		return fmt.Errorf(format, ErrExpect0or1, params)
	}
	return nil
}

// CursorForward moves cursor forward.
// Attr: CUF.
func (d *Decoder) CursorForward(params []int) error {
	const format = "CUF C: %w: %v"

	if len(params) == 0 {
		n := d.x + 1
		d.setCursor(&n, nil)
		return nil
	}

	if len(params) == 1 {
		n := d.x + cursor(params)
		d.setCursor(&n, nil)
		return nil
	}

	if d.strict {
		return fmt.Errorf(format, ErrExpect0or1, params)
	}
	return nil
}

// CursorBack moves cursor back.
// Attr: CUB.
func (d *Decoder) CursorBack(params []int) error {
	const format = "CUB D: %w: %v"

	if len(params) == 0 {
		n := d.x - 1
		d.setCursor(&n, nil)
		return nil
	}

	if len(params) == 1 {
		n := d.x - cursor(params)
		d.setCursor(&n, nil)
		return nil
	}

	if d.strict {
		return fmt.Errorf(format, ErrExpect0or1, params)
	}
	return nil
}

// cursor enforces the ANSI spec where parameter of 0 defaults to 1.
func cursor(params []int) int {
	val := params[0]
	if val == 0 {
		val = 1
	}
	return val
}

// CursorNextLine moves cursor down to the beginning of the line.
// Attr: CNL.
func (d *Decoder) CursorNextLine(params []int) error {
	const format = "CNL E: %w: %v"
	if len(params) == 0 {
		zero := 0
		n := d.y + 1
		d.setCursor(&zero, &n)
		return nil
	}
	if len(params) == 1 {
		zero := 0
		n := d.y + cursor(params)
		d.setCursor(&zero, &n)
		return nil
	}
	if d.strict {
		return fmt.Errorf(format, ErrExpect0or1, params)
	}
	return nil
}

// CursorPreviousLine moves cursor up to the beginning of the line.
// Attr: CPL.
func (d *Decoder) CursorPreviousLine(params []int) error {
	const format = "CPL F: %w: %v"
	if len(params) == 0 {
		zero := 0
		n := d.y - 1
		d.setCursor(&zero, &n)
		return nil
	}
	if len(params) == 1 {
		zero := 0
		n := d.y - cursor(params)
		d.setCursor(&zero, &n)
		return nil
	}
	if d.strict {
		return fmt.Errorf(format, ErrExpect0or1, params)
	}
	return nil
}

// CursorHorizontalAbsolute moves the cursor to column.
// Attr: CHA.
func (d *Decoder) CursorHorizontalAbsolute(params []int) error {
	const format = "CHA G: %w: %v"
	if len(params) == 0 {
		n := 1 // or 0, depending
		d.setCursor(&n, nil)
		return nil
	}
	if len(params) == 1 {
		n := cursor(params)
		d.setCursor(&n, nil)
		return nil
	}
	if d.strict {
		return fmt.Errorf(format, ErrExpect0or1, params)
	}
	return nil
}

// CursorPosition moves the cursor to row and column.
// Attr: CUP.
func (d *Decoder) CursorPosition(params []int) error {
	const format = "CUP H/f: %w: %v"

	const (
		cup0 = 0
		cup1 = 1
		cup2 = 2
	)

	if len(params) == cup0 {
		x, y := 0, 0
		d.setCursor(&x, &y)
		return nil
	}

	if len(params) == cup1 {
		if d.strict {
			return fmt.Errorf(format, ErrExpect0or2, params)
		}
		y := toIndex(params[0])
		x := 0
		d.setCursor(&x, &y)
		return nil
	}

	if len(params) == cup2 {
		y := toIndex(params[0])
		x := toIndex(params[1])
		d.setCursor(&x, &y)
		return nil
	}
	if d.strict {
		return fmt.Errorf(format, ErrExpect0or2, params)
	}
	return nil
}

func toIndex(param int) int {
	if param <= 1 {
		return 0
	}
	return param - 1
}

// EraseInDisplay clears part of the screen.
// Attr: ED.
func (d *Decoder) EraseInDisplay(params []int) error { //nolint:cyclop
	const format = "ED J: %w: %v"

	param := 0
	if len(params) > 0 {
		param = params[0]
	}

	const (
		erase0 = 0 // from cursor to end of screen
		erase1 = 1 // from start of screen up to cursor
		erase2 = 2 // entire screen
		erase3 = 3 // entire screen + scrollback buffer
	)

	switch param {
	case erase0:
		if d.x < len(d.currentLine) {
			d.currentLine = d.currentLine[:d.x]
		}
		if d.y < len(d.buffer) {
			d.buffer = d.buffer[:d.y+1]
			d.buffer[d.y] = d.currentLine
		}
		return nil

	case erase1:
		for i := 0; i < d.y && i < len(d.buffer); i++ {
			d.buffer[i] = d.buffer[i][:0]
		}

		if d.x < len(d.currentLine) {
			for i := 0; i <= d.x; i++ {
				d.currentLine[i] = cell{Attr: defaultAttr(d.palette), Char: ' '}
			}
		} else {
			d.currentLine = d.currentLine[:0]
		}

		if d.y < len(d.buffer) {
			d.buffer[d.y] = d.currentLine
		}
		return nil

	case erase2, erase3:
		for i := range d.buffer {
			d.buffer[i] = d.buffer[i][:0]
		}
		d.currentLine = d.currentLine[:0]
		d.x = 0
		d.y = 0
		return nil

	default:
		if d.strict {
			return fmt.Errorf(format, ErrRecognized, params)
		}
		return nil
	}
}

// EraseInLine clears part of the line.
// Attr: EL.
func (d *Decoder) EraseInLine(params []int) error { //nolint:cyclop
	const format = "EL K: %w: %v"

	param := 0
	if len(params) > 0 {
		param = params[0]
	}

	const (
		erase0 = 0 // from cursor to end of line
		erase1 = 1 // from start of line up to cursor
		erase2 = 2 // entire line
	)

	switch param {
	case erase0:
		if d.x < len(d.currentLine) {
			d.currentLine = d.currentLine[:d.x]
		}
		if d.y < len(d.buffer) {
			d.buffer[d.y] = d.currentLine
		}
		return nil

	case erase1:
		if d.x < len(d.currentLine) {
			for i := 0; i <= d.x; i++ {
				d.currentLine[i] = cell{Attr: defaultAttr(d.palette), Char: ' '}
			}
		} else {
			d.currentLine = []cell{}
		}
		if d.y < len(d.buffer) {
			d.buffer[d.y] = d.currentLine
		}
		return nil

	case erase2:
		d.currentLine = []cell{}
		if d.y < len(d.buffer) {
			d.buffer[d.y] = d.currentLine
		}
		return nil

	default:
		if d.strict {
			return fmt.Errorf(format, ErrRecognized, params)
		}
		return nil
	}
}

// SaveCursorPosition saves the cursor state for later use.
// Abbr: SCP, SCOSC.
func (d *Decoder) SaveCursorPosition(params []int) error {
	const format = "SCP s: %w: %v"
	if len(params) != 0 && d.strict {
		return fmt.Errorf(format, ErrUnexpected, params)
	}
	d.savedX = d.x
	d.savedY = d.y
	return nil
}

// RestoreCursorPosition restores the saved cursor state.
// Abbr: RCP, SCOSC.
func (d *Decoder) RestoreCursorPosition(params []int) error {
	const format = "RCP u: %w: %v"
	if len(params) != 0 && d.strict {
		return fmt.Errorf(format, ErrUnexpected, params)
	}
	d.setCursor(&d.savedX, &d.savedY)
	return nil
}

// ApplyCSI handles cursor movement and erase sequences that alter the buffer or cursor.
// It follows standard ANSI/VT100 CSI final bytes used in the original request:
// A B C D E F G H f J K s u.
func (d *Decoder) ApplyCSI(final byte, params []int) error { //nolint:cyclop
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

// ExportParseCSI exposes parseCSI to external test packages.
func (d *Decoder) ExportParseCSI(br *bufio.Reader, cur Attribute, lineWrap *bool) (Attribute, error) {
	return d.parseCSI(br, cur, lineWrap)
}

// amigaFixes reads the stream into memory, applies Amiga-specific byte
// replacements, and returns a new reader.
func (d *Decoder) amigaFixes(r io.Reader) (io.Reader, error) {
	const format = "decoder amiga fixes: %w"
	const cent, space = 0x9b, 0x20

	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf(format, err)
	}

	replacements := []struct {
		old []byte
		new []byte
	}{
		{[]byte{0x1b, 'c', 0x0c, cent}, nil},
		{[]byte{'0', space, 'p', '\n'}, []byte{'\n'}},
		{[]byte{0x1b, '[', '0', space, 'p', 0x0c}, nil},
		{[]byte{0x1b, '[', 0x1b, '['}, []byte{0x1b, '['}},
		{[]byte{0x1b, '[', '0', space}, []byte{space}},
		{[]byte{0x1b, '[', '3', '4', space}, []byte{0x1b, '[', '3', '4', 'm', space}},
	}
	for _, rep := range replacements {
		b = bytes.ReplaceAll(b, rep.old, rep.new)
	}

	return bytes.NewReader(b), nil
}

func (d *Decoder) parseCSI(br *bufio.Reader, cur Attribute, lineWrap *bool) (Attribute, error) { //nolint:cyclop,funlen,lll,gocognit
	const format = "parse csi %s: %w"
	nb, err := br.ReadByte()
	if err != nil {
		if err == io.EOF {
			return cur, nil
		}
		return cur, fmt.Errorf(format, "read sequence", err)
	}

	if nb != '[' {
		if d.strict {
			return cur, fmt.Errorf(format, string(nb), ErrUnknownEsc)
		}
		return cur, nil
	}

	const size = 4
	params := make([]int, 0, size)
	paramInProgress, private, setmode, linewrp := false, false, false, false
	param := 0

	for {
		cb, err := br.ReadByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			return cur, fmt.Errorf(format, "read character", err)
		}
		switch cb {
		case '=':
			setmode = true
			continue
		case '?':
			private = true
			continue
		case ' ':
			continue
		case ';':
			if paramInProgress {
				paramInProgress, param, params = resetCell(param, params)
				continue
			}
			if d.strict {
				return cur, ErrParam
			}
			params = append(params, -1)
			continue
		}

		// handle ranges and state-dependent logic
		if setmode && cb >= '0' && cb <= '9' {
			linewrp = wrapMode(cb)
			continue
		}
		if setmode && (cb == 'h' || cb == 'l') {
			*lineWrap = setWrapping(cb, linewrp, *lineWrap)
			setmode = false
			continue
		}
		if cb >= '0' && cb <= '9' {
			paramInProgress, param = inProgress(cb, paramInProgress, param)
			continue
		}
		if paramInProgress {
			params = append(params, param)
		}
		if !private && cb == 'm' {
			return ApplySGR(params, cur, d.palette)
		}
		if !private {
			if err := d.ApplyCSI(cb, params); err != nil {
				return cur, err
			}
		}
		break
	}

	return cur, nil
}

// defaultAttr returns the default Attribute (no formatting and default foreground color).
func defaultAttr(pal Palette) Attribute {
	s := style{}
	s.set(pal)
	fg := s.fg
	bg := s.bg
	return Attribute{
		FG:        string(fg),
		BG:        string(bg),
		Bold:      false,
		Underline: false,
		Inverse:   false,
	}
}

// ApplySGR applies SGR parameters to an incoming attribute and returns a new Attribute.
func ApplySGR(params []int, current Attribute, pal Palette) (Attribute, error) { //nolint:cyclop,funlen
	if len(params) == 0 {
		return defaultAttr(pal), nil
	}

	attr := current
	for i := 0; i < len(params); {
		p := params[i]
		switch {
		case p == Reset:
			attr = defaultAttr(pal)
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
			isFG := p == SetFG
			// flattened lookahead logic for extended colors (mode 5 = 256-color)
			if i+2 < len(params) && params[i+1] == 5 {
				code := params[i+2]
				if isFG {
					attr.FG = XtermHex(code, pal)
				} else {
					attr.BG = XtermHex(code, pal)
				}
				i += 3
				continue
			}
			// flattened lookahead logic for true-color (mode 2 = RGB)
			if i+4 < len(params) && params[i+1] == 2 {
				if isFG {
					attr.FG = RGBHex(params, i)
				} else {
					attr.BG = RGBHex(params, i)
				}
				i += 5
				continue
			}
			// malformed or unknown extended color; strict mode could error here
			// however, the default permissive behavior is to skip the base parameter
		}

		i++ // advance by 1 for all standard codes or unrecognized codes
	}

	return attr, nil
}

// RGBHex converts the params into a "true color", red, green, blue hex string.
func RGBHex(params []int, i int) string {
	const format = "%02x%02x%02x"
	if len(params) < i+5 {
		return ""
	}
	const last = 255
	r := max(0, min(last, params[i+2]))
	g := max(0, min(last, params[i+3]))
	b := max(0, min(last, params[i+4]))
	return fmt.Sprintf(format, r, g, b)
}

// BasicHex takes a standard color code and returns a corresponding hexadecimal string.
// When bright is toggled, a lighter color variant is used.
//
// Codes are values between 0 and 7, and any invalid codes returns a blank string.
func BasicHex(code int, bright bool, pal Palette) string {
	const first = 0
	const last = 7
	const colors = 8
	if code < first || code > last {
		return ""
	}
	index := code
	if bright {
		index = code + colors
	}
	switch pal {
	case CGA16:
		return string(CGA()[index])
	case Xterm16:
		return string(Xterm()[index])
	case DP2:
		return string(DPaint2()[index])
	}
	return ""
}

// XtermHex takes a Xterm color code and returns the corresponding RBG values
// as a hexadecimal string.
//
// Codes are values between 0 and 255, and any invalid codes return a blank string.
// The Palette is only used for basic colors codes between 0 and 7.
func XtermHex(code int, pal Palette) string {
	if code < 0 || code > 255 {
		return ""
	}
	const basic = 7
	if code <= basic {
		return BasicHex(code, false, pal)
	}
	const bright = 15
	if code <= bright {
		const colors = 8
		c := code - colors
		return BasicHex(c, true, pal)
	}
	const xterm = 255
	if code <= xterm {
		r, g, b := XtermColors(code)
		const format = "%02x%02x%02x"
		return fmt.Sprintf(format, r, g, b)
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
	invalid := func() (int, int, int) {
		return -1, -1, -1
	}
	return invalid()
}

// XtermColor returns the RGB values for non-system Xterm colors.
func XtermColor(code int) (int, int, int) {
	const levels = 6
	const offset = 16
	const colors = 36
	c := code - offset
	return cubed(c / colors), cubed((c % colors) / levels), cubed(c % levels)
}

// cubed converts a 0-5 cube coordinate to an 8-bit RGB component.
func cubed(c int) int {
	const level0 = 0
	const level1 = 55
	if c == level0 {
		return 0
	}
	return level1 + c*40
}

// XtermGray returns the RGB values for the Xterm greyscale colors.
func XtermGray(code int) (int, int, int) {
	const grayBoundary = 232
	level := code - grayBoundary
	v := 8 + level*10 //nolint:mnd
	return v, v, v
}

// style contains the default Colors and palette.
type style struct {
	palette Palette
	fg      Color
	bg      Color
}

// set the default colors of the palette.
func (s *style) set(pal Palette) {
	s.palette = pal
	switch pal {
	case CGA16:
		s.fg = CGA().DefaultFG()
		s.bg = CGA().DefaultBG()
	case Xterm16:
		s.fg = Xterm().DefaultFG()
		s.bg = Xterm().DefaultBG()
	case DP2:
		s.fg = DPaint2().DefaultFG()
		s.bg = DPaint2().DefaultBG()
	}
}

// buildStyle takes the Attribute and returns a HTML style attribute.
func buildStyle(a Attribute, defaults style) string {
	fg := Color(a.FG) // foreground color
	bg := Color(a.BG) // background color
	if a.Inverse {
		fg, bg = bg, fg
	}

	const size = 3
	elems := make([]string, 0, size)

	foreground := fg
	if foreground == "" {
		foreground = defaults.fg
	}
	var c Color
	c = foreground
	if a.Bold {
		c = Bright(c, defaults.palette)
	}
	elems = append(elems, c.FG())

	if bg != "" {
		background := bg
		const black = CBlack
		if background != "" && background.BG() != black.BG() {
			elems = append(elems, background.BG())
		}
	}
	if a.Underline {
		const underline = "text-decoration:underline;"
		elems = append(elems, underline)
	}
	const sep = ""
	return strings.Join(elems, sep)
}

// Bright takes a palette color and swaps it for a lighter variant.
// For example, Color.CBlack (CGA black) returns Color.CDarkGray (CGA bright black).
func Bright(c Color, pal Palette) Color {
	var standard Colors
	switch pal {
	case CGA16:
		standard = CGA()
	case Xterm16:
		standard = Xterm()
	case DP2:
		standard = DPaint2()
	}
	const size = 16
	colors := make([]string, size)
	for i, x := range standard {
		colors[i] = string(x)
	}
	match := slices.Index(colors, string(c))
	if match >= 0 && match <= 7 {
		return standard[match+8]
	}
	return ""
}

// setCursor sets x and/or y (nil means unchanged).
func (d *Decoder) setCursor(xp *int, yp *int) {
	if xp != nil {
		d.x = max(0, *xp)
	}
	if yp != nil {
		d.y = max(0, *yp)
	}
	d.ensureLine(d.y)
}

// ensureLine ensures the current line exists.
func (d *Decoder) ensureLine(y int) {
	for y >= len(d.buffer) {
		d.buffer = append(d.buffer, []cell{})
	}
	d.currentLine = d.buffer[y]
}

// newline moves cursor to start of next line.
func (d *Decoder) newline() {
	d.setCursor(new(0), new(d.y+1))
}

// writeChar writes a printable character at the cursor location using the given attribute.
func (d *Decoder) writeChar(b byte, attr Attribute) {
	ch := rune(b)
	if d.charset != nil && d.charset != charmap.XUserDefined {
		ch = d.charset.DecodeByte(b)
	}
	d.ensureLine(d.y)

	// pad the line if the cursor jumped ahead
	gap := d.x - len(d.currentLine)
	if gap > 0 {
		d.currentLine = slices.Grow(d.currentLine, gap)
		padding := cell{Attr: defaultAttr(d.palette), Char: ' '}
		for range gap {
			d.currentLine = append(d.currentLine, padding)
		}
	}

	newCell := cell{Attr: attr, Char: ch}
	if d.x == len(d.currentLine) {
		// cursor is at the end of the line
		d.currentLine = append(d.currentLine, newCell)
	} else {
		// cursor is behind the end of the line (overwrite)
		d.currentLine[d.x] = newCell
	}
	d.buffer[d.y] = d.currentLine
	d.x++

	if d.x >= d.width {
		d.newline()
	}
}
