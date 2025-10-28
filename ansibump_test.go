package ansibump_test

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/bengarrett/ansibump"
	"github.com/nalgeon/be"
	"golang.org/x/text/encoding/charmap"
)

func ExampleBuffer() {
	const ansi = "\x1b[0m\x1b[5;33;42mHI\x1b[0m"

	// use cga palette with codepage 437
	const cga = ansibump.CGA16
	r := strings.NewReader(ansi)
	buf, _ := ansibump.Buffer(r, 80, false, cga, charmap.CodePage437)
	fmt.Printf("%q\n", buf.String())

	// use xterm palette with codepage 437
	const xterm = ansibump.Xterm16
	r = strings.NewReader(ansi)
	buf, _ = ansibump.Buffer(r, 80, false, xterm, charmap.CodePage437)
	fmt.Printf("%q\n", buf.String())
	// Output: "<div style=\"color:#aaa;background-color:#000;\"><span style=\"color:#a50;background-color:#0a0;\">HI</span></div>"
	// "<div style=\"color:#c0c0c0;background-color:#000;\"><span style=\"color:#808000;background-color:#008000;\">HI</span></div>"
}

func ExampleBuffer_codepage() {
	const ansi = "\x1b[0;34;47m\xae\xaf\x1b[0m"
	const cga = ansibump.CGA16
	// using Code Page 437
	r := strings.NewReader(ansi)
	buf, _ := ansibump.Buffer(r, 80, false, cga, charmap.CodePage437)
	fmt.Printf("%q\n", buf.String())

	// using Latin 1 (ISO-8859-1)
	r = strings.NewReader(ansi)
	buf, _ = ansibump.Buffer(r, 80, false, cga, charmap.ISO8859_1)
	fmt.Printf("%q\n", buf.String())
	// Output: "<div style=\"color:#aaa;background-color:#000;\"><span style=\"color:#00a;background-color:#aaa;\">«»</span></div>"
	// "<div style=\"color:#aaa;background-color:#000;\"><span style=\"color:#00a;background-color:#aaa;\">®¯</span></div>"
}

func ExampleBytes() {
	const ansi = "\x1b[0m\x1b[5;30;42mHI\x1b[0m"
	r := strings.NewReader(ansi)
	p, _ := ansibump.Bytes(r, 80)
	fmt.Printf("%q", p)
	// Output: "<div style=\"color:#aaa;background-color:#000;\"><span style=\"color:#000;background-color:#0a0;\">HI</span></div>"
}

func ExampleString() {
	const ansi = "\x1b[0m\n\x1b[1;34m\x02\x1b[0m \x1b[1;34mA\x1b[36mN\x1b[33mS\x1b[37mI\x1b[35mbump\x1b[0;33m\x1b[37m"
	r := strings.NewReader(ansi)
	s, _ := ansibump.String(r, 80)
	fmt.Printf("%q", s)
	// Output: "<div style=\"color:#aaa;background-color:#000;\">\n<span style=\"color:#55f;\">\x02</span><span style=\"color:#aaa;\"> </span><span style=\"color:#55f;\">A</span><span style=\"color:#5ff;\">N</span><span style=\"color:#ff5;\">S</span><span style=\"color:#fff;\">I</span><span style=\"color:#f5f;\">bump</span></div>"
}

func ExampleString_xterm256() {
	const ansi = "\x1b[0m\x1b[38;5;93mPurple\x1b[0m \x1b[38;5;94mOrange4\x1b[0m"
	r := strings.NewReader(ansi)
	s, _ := ansibump.String(r, 80)
	fmt.Printf("%q", s)
	// Output: "<div style=\"color:#aaa;background-color:#000;\"><span style=\"color:#8700ff;\">Purple</span><span style=\"color:#aaa;\"> </span><span style=\"color:#875f00;\">Orange4</span></div>"
}

func ExampleString_rgb() {
	const ansi = "\x1b[0m\x1b[38;2;135;0;255;48;2;135;95;0mPurple on Orange4\x1b[0m"
	r := strings.NewReader(ansi)
	s, _ := ansibump.String(r, 80)
	fmt.Printf("%q", s)
	// Output: "<div style=\"color:#aaa;background-color:#000;\"><span style=\"color:#8700ff;background-color:#875f00;\">Purple on Orange4</span></div>"
}

func ExampleWriteTo() {
	const ansi = "\x1b[0m\x1b[5;30;42mHI\x1b[0m"
	input := strings.NewReader(ansi)
	var b bytes.Buffer
	output := bufio.NewWriter(&b)
	cnt, _ := ansibump.WriteTo(input, output, 80)
	output.Flush()
	fmt.Printf("%d bytes written\n%q", cnt, b.String())
	// Output: 110 bytes written
	// "<div style=\"color:#aaa;background-color:#000;\"><span style=\"color:#000;background-color:#0a0;\">HI</span></div>"
}

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
	// test basic colors
	const red = 1
	h := ansibump.XtermHex(red, cga)
	be.Equal(t, h, "a00")
	h = ansibump.XtermHex(red, xtm)
	be.Equal(t, h, "800000")
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
