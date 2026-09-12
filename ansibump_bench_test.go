//nolint:gochecknoglobals
package ansibump_test

import (
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/bengarrett/ansibump"
)

// Benchmarks for various Match patterns, to run:
// go test -bench=Benchmark -benchmem

var testMatches = [...]ansibump.Color{
	ansibump.DPWhite, ansibump.CBlack, ansibump.CCyan, "hello", ansibump.XTeal, ansibump.CYellow, ansibump.DPRed, ansibump.XBlue, ansibump.XRed,
}

func BenchmarkColors(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		for _, s := range testMatches {
			_ = s.BG()
			_ = s.FG()
		}
	}
}

type Color string

const (
	CBlack    Color = "000" // cga black
	CRed      Color = "a00" // red
	CGreen    Color = "0a0" // green
	CBrown    Color = "a50" // yellow
	CBlue     Color = "00a" // blue
	CMagenta  Color = "a0a" // magenta
	CCyan     Color = "0aa" // cyan
	CGray     Color = "aaa" // white
	CDarkGray Color = "555" // bright black
	CLRed     Color = "f55" // bright red
	CLGreen   Color = "5f5" // bright green
	CYellow   Color = "ff5" // bright yellow
	CLBlue    Color = "55f" // bright blue
	CLMagenta Color = "f5f" // bright magenta
	CLCyan    Color = "5ff" // bright cyan
	CWhite    Color = "fff" // bright white
)

type Colors [16]Color

func CGA() Colors {
	return Colors{
		CBlack, CRed, CGreen, CBrown, CBlue, CMagenta, CCyan, CGray,
		CDarkGray, CLRed, CLGreen, CYellow, CLBlue, CLMagenta, CLCyan, CWhite,
	}
}

var cgaColors = Colors{
	CBlack, CRed, CGreen, CBrown, CBlue, CMagenta, CCyan, CGray,
	CDarkGray, CLRed, CLGreen, CYellow, CLBlue, CLMagenta, CLCyan, CWhite,
}

func CGA01() Colors {
	return cgaColors
}

func bright(c ansibump.Color, standard Colors) Color { //nolint:unparam
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

func BenchmarkBright00(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		for _, c := range testMatches {
			_ = bright(c, CGA())
		}
	}
}

func BenchmarkBright01(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		for _, c := range testMatches {
			_ = bright(c, CGA01())
		}
	}
}

const (
	ansi0 = "\x1b[0m\x1b[5;33;42mHI\x1b[0m"
	ansi1 = "\x1b[0;34;47m\xae\xaf\x1b[0m"
	ansi2 = "\x1b[0m\x1b[5;30;42mHI\x1b[0m"
	ansi3 = "\x1b[0m\n\x1b[1;34m\x02\x1b[0m \x1b[1;34mA\x1b[36mN\x1b[33mS\x1b[37mI\x1b[35mbump\x1b[0;33m\x1b[37m"
	ansi4 = "\x1b[0m\x1b[38;5;93mPurple\x1b[0m \x1b[38;5;94mOrange4\x1b[0m"
	ansi5 = "\x1b[0m\x1b[38;2;135;0;255;48;2;135;95;0mPurple on Orange4\x1b[0m"
	ansi6 = "\x1b[0m\x1b[5;30;42mHI\x1b[0m"
)

var ansis = []string{ansi0, ansi1, ansi2, ansi3, ansi4, ansi5, ansi6}

func BenchmarkBytes(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		for _, ansi := range ansis {
			r := strings.NewReader(ansi)
			_, _ = ansibump.Bytes(r, 80)
		}
	}
}

func BenchmarkString(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		for _, ansi := range ansis {
			r := strings.NewReader(ansi)
			_, _ = ansibump.String(r, 80)
		}
	}
}

func BenchmarkHex00(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		for code := range 255 {
			r, g, b := ansibump.XtermColors(code)
			const format = "%02x%02x%02x"
			_ = fmt.Sprintf(format, r, g, b)
		}
	}
}

func BenchmarkHex01(b *testing.B) {
	// This is the most performant and also idiomatic
	b.ReportAllocs()

	const last = 255
	for b.Loop() {
		for code := range 255 {
			r, g, b := ansibump.XtermColors(code)
			src := [3]byte{
				uint8(max(0, min(last, r))),
				uint8(max(0, min(last, g))),
				uint8(max(0, min(last, b))),
			}
			var dst [6]byte
			hex.Encode(dst[:], src[:])
			_ = string(dst[:])
		}
	}
}

func BenchmarkHex02(b *testing.B) {
	b.ReportAllocs()
	const hexTable = "0123456789abcdef"
	for b.Loop() {
		for code := range 255 {
			r, g, b := ansibump.XtermColors(code)
			r &= 0xff
			g &= 0xff
			b &= 0xff

			buf := [6]byte{
				hexTable[r>>4], hexTable[r&0x0f],
				hexTable[g>>4], hexTable[g&0x0f],
				hexTable[b>>4], hexTable[b&0x0f],
			}
			_ = string(buf[:])
		}
	}
}
