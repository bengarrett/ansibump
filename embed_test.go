package ansibump_test

import (
	"bufio"
	"bytes"
	_ "embed"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/bengarrett/ansibump"
	"github.com/nalgeon/be"
)

// Embed the test file directly into the binary.
//
//go:embed testdata/ZII-RTXT
var testAnsiArt []byte

var update = flag.Bool("update", false, "update golden files") //nolint:gochecknoglobals

func checkGolden(t *testing.T, name string, actual []byte) {
	t.Helper()

	// Normalize CR LF to LF to avoid OS-specific diffs
	actual = bytes.ReplaceAll(actual, []byte("\r\n"), []byte("\n"))

	goldenPath := filepath.Join("testdata", name+".golden")

	if *update {
		err := os.WriteFile(goldenPath, actual, 0o644) //nolint:gosec
		be.Err(t, err, nil)
		t.Logf("updated golden file: %s", goldenPath)
		return
	}

	expected, err := os.ReadFile(goldenPath)
	be.Err(t, err, nil)

	expected = bytes.ReplaceAll(expected, []byte("\r\n"), []byte("\n"))

	if !bytes.Equal(actual, expected) {
		t.Fatalf("output does not match golden file %s\nGot:\n%s\nWant:\n%s", goldenPath, actual, expected)
	}
}

// Cell represents a single terminal cell with character and style attributes.
type Cell struct {
	Rune rune
	Attr ansibump.Attribute
}

// Canvas represents an 80-column terminal screen buffer.
type Canvas struct {
	Width  int
	Height int
	X      int
	Y      int
	Grid   [][]Cell
}

func NewCanvas(t *testing.T, width, height int) *Canvas {
	t.Helper()
	grid := make([][]Cell, height)
	for i := range grid {
		grid[i] = make([]Cell, width)
	}
	return &Canvas{
		Width:  width,
		Height: height,
		Grid:   grid,
	}
}

// WriteRune places a rune on the canvas and advances the cursor.
func (c *Canvas) WriteRune(t *testing.T, r rune, attr ansibump.Attribute) {
	t.Helper()
	if r == '\n' {
		c.X = 0
		c.Y++
		c.ensureHeight(t)
		return
	}
	if r == '\r' {
		c.X = 0
		return
	}

	c.ensureHeight(t)

	if c.X >= c.Width {
		c.X = 0
		c.Y++
		c.ensureHeight(t)
	}

	c.Grid[c.Y][c.X] = Cell{Rune: r, Attr: attr}
	c.X++
}

// String serializes the grid into plain text lines, stripping empty trailing rows.
func (c *Canvas) String(t *testing.T) string {
	t.Helper()
	var buf strings.Builder

	// Find last non-empty row
	lastNonEmptyRow := -1

	for y, row := range slices.Backward(c.Grid) {
		for _, cell := range slices.Backward(row) {
			if cell.Rune != 0 && cell.Rune != ' ' {
				lastNonEmptyRow = y
				break
			}
		}
		if lastNonEmptyRow != -1 {
			break
		}
	}

	for y := 0; y <= lastNonEmptyRow; y++ {
		rowRunes := make([]rune, len(c.Grid[y]))
		for x, cell := range c.Grid[y] {
			if cell.Rune == 0 {
				rowRunes[x] = ' '
			} else {
				rowRunes[x] = cell.Rune
			}
		}
		line := strings.TrimRight(string(rowRunes), " ")
		buf.WriteString(line)
		buf.WriteRune('\n')
	}

	return strings.TrimRight(buf.String(), "\n")
}

func (c *Canvas) ensureHeight(t *testing.T) {
	t.Helper()
	for c.Y >= len(c.Grid) {
		c.Grid = append(c.Grid, make([]Cell, c.Width))
	}
}

func TestRenderZIIRTXT(t *testing.T) {
	t.Parallel()

	d := ansibump.NewDecoder(nil)
	br := bufio.NewReader(bytes.NewReader(testAnsiArt))
	canvas := NewCanvas(t, 80, 25)

	var (
		cur      ansibump.Attribute
		lineWrap bool
	)

	for {
		b, err := br.ReadByte()
		if errors.Is(err, io.EOF) {
			break
		}
		be.Err(t, err, nil)

		if b == 0x1b { // ESC byte
			cur, err = d.ExportParseCSI(br, cur, &lineWrap)
			be.Err(t, err, nil)
			continue
		}

		canvas.WriteRune(t, rune(b), cur)
	}

	rendered := canvas.String(t)
	checkGolden(t, "ZII-RTXT", []byte(rendered))
}

func TestAnsiToHTMLGolden(t *testing.T) {
	t.Parallel()

	c := ansibump.Customizer{}
	r := bytes.NewReader(testAnsiArt)
	p, err := c.Bytes(r)
	be.Err(t, err, nil)

	checkGolden(t, "ZII-RTXT.html", p)
}
