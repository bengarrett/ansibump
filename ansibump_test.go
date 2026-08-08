package ansibump_test

import (
	"strings"
	"testing"

	"github.com/bengarrett/ansibump"
	"github.com/nalgeon/be"
)

func TestColor(t *testing.T) {
	t.Parallel()
	const cga = ansibump.CGA16
	const xtm = ansibump.Xterm16
	blk := ansibump.CBlack
	be.Equal(t, blk.BG(), "background-color:#000;")
	be.Equal(t, blk.FG(), "color:#000;")
	colr := ansibump.Bright(ansibump.CBlack, cga)
	be.Equal(t, colr, ansibump.CDarkGray)
	colr = ansibump.Bright(ansibump.CRed, xtm) // there's no corresponding xterm color
	be.Equal(t, colr, "")
	colr = ansibump.Bright(ansibump.CGreen, cga)
	be.Equal(t, colr, ansibump.CLGreen)
	be.Equal(t, colr.BG(), "background-color:#5f5;")
	be.Equal(t, colr.FG(), "color:#5f5;")
}

func TestBasic(t *testing.T) {
	t.Parallel()
	const cga = ansibump.CGA16
	const xtm = ansibump.Xterm16
	const dp2 = ansibump.DP2
	// test basic colors
	const red = 1
	h := ansibump.XtermHex(red, cga)
	be.Equal(t, h, "a00")
	h = ansibump.XtermHex(red, xtm)
	be.Equal(t, h, "800000")
	h = ansibump.XtermHex(red, dp2)
	be.Equal(t, h, "a80000")
}

func TestColors(t *testing.T) {
	t.Parallel()
	const cga = ansibump.CGA16
	// test xterm 8-bit colors
	const black = 0
	r, g, b := ansibump.XtermColors(black)
	be.Equal(t, r, -1)
	be.Equal(t, g, -1)
	be.Equal(t, b, -1)
	h := ansibump.XtermHex(black, cga)
	be.Equal(t, h, "000")
	const grey0 = 16
	r, g, b = ansibump.XtermColors(grey0)
	be.Equal(t, r, 0)
	be.Equal(t, g, 0)
	be.Equal(t, b, 0)
	h = ansibump.XtermHex(grey0, cga)
	be.Equal(t, h, "000000")
	const lightCyan1 = 195
	r, g, b = ansibump.XtermColors(lightCyan1)
	be.Equal(t, r, 215)
	be.Equal(t, g, 255)
	be.Equal(t, b, 255)
	h = ansibump.XtermHex(lightCyan1, cga)
	be.Equal(t, h, "d7ffff")
	const red1 = 196
	r, g, b = ansibump.XtermColors(red1)
	be.Equal(t, r, 255)
	be.Equal(t, g, 0)
	be.Equal(t, b, 0)
	h = ansibump.XtermHex(red1, cga)
	be.Equal(t, h, "ff0000")
	const grey100 = 231
	r, g, b = ansibump.XtermColors(grey100)
	be.Equal(t, r, 255)
	be.Equal(t, g, 255)
	be.Equal(t, b, 255)
	h = ansibump.XtermHex(grey100, cga)
	be.Equal(t, h, "ffffff")
	const grey3 = 232
	r, g, b = ansibump.XtermColors(grey3)
	be.Equal(t, r, 8)
	be.Equal(t, g, 8)
	be.Equal(t, b, 8)
	h = ansibump.XtermHex(grey3, cga)
	be.Equal(t, h, "080808")
}

func TestRGB(t *testing.T) {
	t.Parallel()
	darkcyan := []int{38, 2, 0, 175, 135}
	s := ansibump.RGBHex(darkcyan, 0)
	be.Equal(t, s, "00af87")
	red3 := []int{48, 2, 215, 0, 0}
	s = ansibump.RGBHex(red3, 0)
	be.Equal(t, s, "d70000")
	// out of range
	s = ansibump.RGBHex(red3, 99)
	be.Equal(t, s, "")
	s = ansibump.RGBHex([]int{}, 1)
	be.Equal(t, s, "")
}

func TestCustomizer(t *testing.T) {
	t.Parallel()
	const s = "\x1b[0;34;47m\x1bc\x0c\x9b\x1b[0m"

	custom := ansibump.Customizer{
		AmigaParser: true,
	}
	r := strings.NewReader(s)
	got, err := custom.String(r)
	be.Err(t, err, nil)
	be.Equal(t, got, `<div style="color:#aaa;background-color:#000;"></div>`)

	r = strings.NewReader(s)
	custom.AmigaParser = false
	got, err = custom.String(r)
	be.Err(t, err, nil)
	be.Equal(t, got, `<div style="color:#aaa;background-color:#000;">`+
		`<span style="color:#00a;background-color:#aaa;"> `+"\u009b"+`</span>`+
		`</div>`)

	r = strings.NewReader(s)
	custom.AmigaParser = true
	custom.Color = ansibump.DP2
	got, err = custom.String(r)
	be.Err(t, err, nil)
	be.Equal(t, got, `<div style="color:#747474;background-color:#000;"></div>`)

	broken := "ABC\x1bmxyz"
	r = strings.NewReader(broken)
	custom.Strict = true
	got, err = custom.String(r)
	be.Equal(t, got, "")
	be.Err(t, err)
}
