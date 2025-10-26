ANSIbump was originally vibe coded on 2025, Oct-26 with GTP-5 mini using the following prompt:

> "in golang create a package that converters ansi escape codes including colors and cursor movement into html"

The code below was the result and heavily adapted and modified for ANSIbump.

Some of the changes that were manually applied,

- Support for IBM Code Pages including CP437
- Support for color palettes
- Replaced a bold CSS flag with light color values
- Many many many linter fixes
- Refactored the names of funcs and values to be more Go idiomatic
- The use of Go errors and Go templates
- Use of a custom Color type instead of hard coded string hex values
- Removed the deep nested conditionals and complexity
- Support for 1 value cursor movements commonly found in BBS era ANSI art
- Tests, documentation, and examples
- Bug fixes
- Quality of life funcs to return HTML as bytes.Buffer, []bytes, or write to an io.Writer.

```go
package ansihtml

import (
	"bufio"
	"fmt"
	"html"
	"io"
	"strconv"
	"strings"
)

// ansihtml converts ANSI escape sequences (colors, cursor movement, erases)
// into an HTML representation. The API centers on Decoder which reads an
// io.Reader and builds a character buffer with attributes, then renders it as HTML.

// Attribute describes styling for a single character cell.
type Attribute struct {
	FG        string // hex color like "rrggbb" (no leading #) or empty for default
	BG        string // hex color like "rrggbb"
	Bold      bool
	Underline bool
	Inverse   bool
}

// cell in the output buffer
type cell struct {
	Attr Attribute
	Char string
}

// Decoder maintains the screen buffer and cursor state while parsing ANSI.
type Decoder struct {
	buffer      [][]cell
	currentLine []cell
	x, y        int

	savedX, savedY int

	width int

	// default colors used for the outer div
	defaultFG string
	defaultBG string

	strict bool
}

// NewDecoder creates a Decoder with a given width (columns). If width <= 0, 80 is used.
func NewDecoder(width int, strict bool) *Decoder {
	if width <= 0 {
		width = 80
	}
	d := &Decoder{
		buffer:    [][]cell{{}},
		x:         0,
		y:         0,
		width:     width,
		defaultFG: "000000",
		defaultBG: "ffffff",
		strict:    strict,
	}
	d.currentLine = d.buffer[0]
	return d
}

// Convert is a convenience: create a Decoder, Play the input, and return AsHTML or an error.
func Convert(r io.Reader, width int, strict bool) (string, error) {
	d := NewDecoder(width, strict)
	if err := d.Play(r); err != nil {
		return "", err
	}
	return d.AsHTML(), nil
}

// --- Helpers for managing cursor and buffer ---

func (d *Decoder) ensureLine(y int) {
	for y >= len(d.buffer) {
		d.buffer = append(d.buffer, []cell{})
	}
	d.currentLine = d.buffer[y]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// setCursor sets x and/or y (nil means unchanged)
func (d *Decoder) setCursor(xp *int, yp *int) {
	if xp != nil {
		if *xp < 0 {
			d.x = 0
		} else {
			d.x = *xp
		}
	}
	if yp != nil {
		if *yp < 0 {
			d.y = 0
		} else {
			d.y = *yp
		}
	}
	d.ensureLine(d.y)
}

// newline moves cursor to start of next line
func (d *Decoder) newline() {
	d.setCursor(ptrInt(0), ptrInt(d.y+1))
}

// writeChar writes a printable character at the cursor location using given attribute.
func (d *Decoder) writeChar(ch string, attr Attribute) {
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

// defaultAttr returns the default Attribute (no styles).
func defaultAttr() Attribute {
	return Attribute{FG: "", BG: "", Bold: false, Underline: false, Inverse: false}
}

// merge adjacent cells (not strictly necessary for buffer fidelity; done at render time)

// --- Parsing / Play ---

// Play reads bytes from r and interprets ANSI sequences, updating the buffer.
func (d *Decoder) Play(r io.Reader) error {
	br := bufio.NewReader(r)
	// current attribute applied to subsequent characters
	cur := defaultAttr()
	for {
		b, err := br.ReadByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if b >= ' ' {
			// printable
			d.writeChar(string(b), cur)
			continue
		}
		switch b {
		case '\n':
			d.newline()
		case '\r':
			// ignore CR
		case 0x1A:
			// EOF (SAUCE) break
			return nil
		case 0x1B: // ESC
			nb, err := br.ReadByte()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
			if nb != '[' {
				// We only handle CSI sequences (ESC [ ... )
				if d.strict {
					return fmt.Errorf("unrecognized ESC sequence after ESC: %q", nb)
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
					return err
				}
				if '0' <= cb && cb <= '9' {
					if !paramInProgress {
						paramInProgress = true
						paramVal = int(cb - '0')
					} else {
						paramVal = paramVal*10 + int(cb-'0')
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
							return fmt.Errorf("encountered ';' without parameter")
						}
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
					paramInProgress = false
					paramVal = 0
				}
				if !private {
					if cb == 'm' {
						// SGR sequence: can be complex (including 38/48 extended)
						newAttr, err := applySGR(params, cur)
						if err != nil {
							return err
						}
						cur = newAttr
					} else {
						// other CSI sequences that affect cursor / buffer
						if err := d.applyCSI(cb, params, &cur); err != nil {
							return err
						}
					}
				}
				break
			}
		default:
			// control codes like BEL, VT, etc. Ignore unless remap required.
			// We won't remap CP437 nonprintables here.
			if d.strict {
				return fmt.Errorf("unrecognised control byte: 0x%02x", b)
			}
		}
	}
	return nil
}

// applyCSI handles cursor movement and erase sequences that alter the buffer or cursor.
// It follows standard ANSI/VT100 CSI final bytes used in the original request:
// A B C D E F G H f J K s u
func (d *Decoder) applyCSI(final byte, params []int, cur *Attribute) error {
	switch final {
	case 'A': // CUU - up
		if len(params) == 0 {
			n := d.y - 1
			d.setCursor(nil, &n)
		} else if len(params) == 1 {
			n := d.y - params[0]
			d.setCursor(nil, &n)
		} else if d.strict {
			return fmt.Errorf("A: expected 0 or 1 param")
		}
	case 'B': // CUD - down
		if len(params) == 0 {
			n := d.y + 1
			d.setCursor(nil, &n)
		} else if len(params) == 1 {
			n := d.y + params[0]
			d.setCursor(nil, &n)
		} else if d.strict {
			return fmt.Errorf("B: expected 0 or 1 param")
		}
	case 'C': // CUF - right
		if len(params) == 0 {
			n := d.x + 1
			d.setCursor(&n, nil)
		} else if len(params) == 1 {
			n := d.x + params[0]
			d.setCursor(&n, nil)
		} else if d.strict {
			return fmt.Errorf("C: expected 0 or 1 param")
		}
	case 'D': // CUB - left
		if len(params) == 0 {
			n := d.x - 1
			d.setCursor(&n, nil)
		} else if len(params) == 1 {
			n := d.x - params[0]
			d.setCursor(&n, nil)
		} else if d.strict {
			return fmt.Errorf("D: expected 0 or 1 param")
		}
	case 'E': // CNL - next line (cursor to beginning of line N lines down)
		if len(params) == 0 {
			n := d.y + 1
			d.setCursor(ptrInt(0), &n)
		} else if len(params) == 1 {
			n := d.y + params[0]
			d.setCursor(ptrInt(0), &n)
		} else if d.strict {
			return fmt.Errorf("E: expected 0 or 1 param")
		}
	case 'F': // CPL - previous line (cursor to beginning of line N lines up)
		if len(params) == 0 {
			n := d.y - 1
			d.setCursor(ptrInt(0), &n)
		} else if len(params) == 1 {
			n := d.y - params[0]
			d.setCursor(ptrInt(0), &n)
		} else if d.strict {
			return fmt.Errorf("F: expected 0 or 1 param")
		}
	case 'G': // CHA - cursor horizontal absolute (column)
		if len(params) == 1 {
			n := params[0]
			d.setCursor(&n, nil)
		} else if d.strict {
			return fmt.Errorf("G: expected 1 param")
		}
	case 'H', 'f': // CUP - cursor position (line;col) 1-based
		if len(params) == 0 {
			x := 0
			y := 0
			d.setCursor(&x, &y)
		} else if len(params) == 2 {
			x := params[1] - 1
			y := params[0] - 1
			d.setCursor(&x, &y)
		} else if d.strict {
			return fmt.Errorf("H/f: expected 0 or 2 params")
		}
	case 'J': // ED - erase in display
		// 0 or empty: from cursor to end of screen
		if len(params) == 0 || (len(params) == 1 && params[0] == 0) {
			// truncate current line from cursor onward
			if d.x < len(d.currentLine) {
				d.currentLine = d.currentLine[:d.x]
			}
			// truncate lines below current
			if d.y+1 < len(d.buffer) {
				d.buffer = d.buffer[:d.y+1]
			}
			d.buffer[d.y] = d.currentLine
		} else if len(params) == 1 && params[0] == 1 {
			// erase up to cursor (from top to cursor)
			for i := 0; i < d.y; i++ {
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
		} else if len(params) == 1 && params[0] == 2 {
			// erase entire screen
			for i := range d.buffer {
				d.buffer[i] = []cell{}
			}
		} else if d.strict {
			return fmt.Errorf("J: unrecognised parameters")
		}
	case 'K': // EL - erase in line
		// 0 or empty: from cursor to end of line
		if len(params) == 0 || (len(params) == 1 && params[0] == 0) {
			if d.x < len(d.currentLine) {
				d.currentLine = d.currentLine[:d.x]
			}
			d.buffer[d.y] = d.currentLine
		} else if len(params) == 1 && params[0] == 1 {
			// erase up to cursor in line
			if d.x < len(d.currentLine) {
				for i := 0; i <= d.x && i < len(d.currentLine); i++ {
					d.currentLine[i] = cell{Attr: defaultAttr(), Char: " "}
				}
			} else {
				d.currentLine = []cell{}
			}
			d.buffer[d.y] = d.currentLine
		} else if len(params) == 1 && params[0] == 2 {
			// erase entire line
			d.currentLine = []cell{}
			d.buffer[d.y] = d.currentLine
		} else if d.strict {
			return fmt.Errorf("K: unrecognised parameters")
		}
	case 's': // save cursor
		if len(params) != 0 && d.strict {
			return fmt.Errorf("s: unexpected params")
		}
		d.savedX = d.x
		d.savedY = d.y
	case 'u': // restore cursor
		if len(params) != 0 && d.strict {
			return fmt.Errorf("u: unexpected params")
		}
		d.setCursor(&d.savedX, &d.savedY)
	default:
		if d.strict {
			return fmt.Errorf("unrecognized CSI final byte: %c", final)
		}
	}
	return nil
}

// --- SGR (m) handling including 8/16 colors, 256 / truecolor. ---

// applySGR applies SGR parameters to an incoming attribute and returns a new Attribute.
func applySGR(params []int, cur Attribute) (Attribute, error) {
	attr := cur // start from current
	if len(params) == 0 {
		// treat empty SGR as reset per common implementations
		return defaultAttr(), nil
	}
	i := 0
	for i < len(params) {
		p := params[i]
		switch {
		case p == 0:
			attr = defaultAttr()
		case p == 1:
			attr.Bold = true
		case p == 21 || p == 22:
			attr.Bold = false
		case p == 4:
			attr.Underline = true
		case p == 24:
			attr.Underline = false
		case p == 7:
			attr.Inverse = true
		case p == 27:
			attr.Inverse = false
		case p == 39:
			attr.FG = ""
		case p == 49:
			attr.BG = ""
		case 30 <= p && p <= 37:
			attr.FG = colorIndexToHex(p-30, false)
		case 40 <= p && p <= 47:
			attr.BG = colorIndexToHex(p-40, false)
		case 90 <= p && p <= 97:
			attr.FG = colorIndexToHex(p-90, true)
		case 100 <= p && p <= 107:
			attr.BG = colorIndexToHex(p-100, true)
		case p == 38 || p == 48:
			// extended color: either 5;n (256 color) or 2;r;g;b (truecolor)
			isFG := (p == 38)
			// look ahead
			if i+1 >= len(params) {
				if true {
					// malformed; ignore per permissive behavior
					i++
					continue
				}
			}
			mode := params[i+1]
			if mode == 5 {
				// 256-color: next param is index
				if i+2 >= len(params) {
					// malformed, skip
					i += 2
					continue
				}
				idx := params[i+2]
				hex := xterm256ToHex(idx)
				if isFG {
					attr.FG = hex
				} else {
					attr.BG = hex
				}
				i += 2
			} else if mode == 2 {
				// truecolor: need next 3 params r,g,b
				if i+4 >= len(params) {
					// malformed
					i += 1
					continue
				}
				r := clamp(params[i+2], 0, 255)
				g := clamp(params[i+3], 0, 255)
				b := clamp(params[i+4], 0, 255)
				hex := fmt.Sprintf("%02x%02x%02x", r, g, b)
				if isFG {
					attr.FG = hex
				} else {
					attr.BG = hex
				}
				i += 4
			} else {
				// unknown mode; skip the indicator
				i++
			}
		default:
			// unhandled codes are ignored in permissive mode
		}
		i++
	}
	return attr, nil
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

// colorIndexToHex maps basic 0..7 colors (and bright flag) to CSS hex strings
// using common VGA-like palette (approximate).
func colorIndexToHex(idx int, bright bool) string {
	// Basic mapping similar to many terminals
	basic := [8]string{
		"000000", // black
		"aa0000", // red
		"00aa00", // green
		"aa5500", // yellow / brownish
		"0000aa", // blue
		"aa00aa", // magenta
		"00aaaa", // cyan
		"aaaaaa", // white / light gray
	}
	brightMap := [8]string{
		"555555", // bright black / dark gray
		"ff5555",
		"55ff55",
		"ffff55",
		"5555ff",
		"ff55ff",
		"55ffff",
		"ffffff",
	}
	if idx < 0 || idx > 7 {
		return ""
	}
	if bright {
		return brightMap[idx]
	}
	return basic[idx]
}

// xterm256ToHex converts a 256-color palette index to a hex string.
func xterm256ToHex(idx int) string {
	if idx < 0 {
		idx = 0
	}
	if idx < 16 {
		// standard 16 colors (we map to our colorIndexToHex)
		if idx < 8 {
			return colorIndexToHex(idx, false)
		}
		return colorIndexToHex(idx-8, true)
	}
	if idx >= 16 && idx <= 231 {
		// 6x6x6 color cube
		c := idx - 16
		r := c / 36
		g := (c % 36) / 6
		b := c % 6
		// each component maps 0..5 to values 0,95,135,175,215,255
		val := func(v int) int {
			if v == 0 {
				return 0
			}
			return 55 + v*40
		}
		return fmt.Sprintf("%02x%02x%02x", val(r), val(g), val(b))
	}
	// 232..255 grayscale
	if idx >= 232 && idx <= 255 {
		v := 8 + (idx-232)*10
		if v > 255 {
			v = 255
		}
		return fmt.Sprintf("%02x%02x%02x", v, v, v)
	}
	// fallback
	return "000000"
}

// --- HTML rendering ---

// AsHTMLLines renders each buffer line into a single HTML string (without wrapping div).
// Each contiguous run of identical attributes is wrapped in a <span style="...">.
func (d *Decoder) AsHTMLLines() []string {
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

// AsHTML returns a full HTML fragment with outer div using default colors and inner lines joined with newlines.
func (d *Decoder) AsHTML() string {
	lines := d.AsHTMLLines()
	// build default color values if possible: fallback to defaults in Decoder
	defFg := d.defaultFG
	defBg := d.defaultBG
	return `<div style="color: #` + defFg + `; background-color: #` + defBg + `;">` + strings.Join(lines, "\n") + `</div>`
}

func attrEqual(a, b Attribute) bool {
	return a.FG == b.FG && a.BG == b.BG && a.Bold == b.Bold && a.Underline == b.Underline && a.Inverse == b.Inverse
}

func buildStyle(a Attribute) string {
	// Determine effective fg/bg respecting inverse
	fg := a.FG
	bg := a.BG
	if a.Inverse {
		fg, bg = bg, fg
	}
	parts := []string{}
	if fg != "" {
		parts = append(parts, "color: #"+fg+";")
	}
	if bg != "" {
		parts = append(parts, "background-color: #"+bg+";")
	}
	if a.Bold {
		parts = append(parts, "font-weight: bold;")
	}
	if a.Underline {
		parts = append(parts, "text-decoration: underline;")
	}
	return strings.Join(parts, " ")
}
```
