package ansibump_test

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"github.com/bengarrett/ansibump"
	"golang.org/x/text/encoding/charmap"
)

func ExampleBuffer() {
	const ansi = "\x1b[0m\x1b[5;33;42mHI\x1b[0m"

	// use cga palette with codepage 437
	r := strings.NewReader(ansi)
	customizer := ansibump.Customizer{
		Width:       80,
		AmigaParser: false,
		Strict:      false,
		Color:       ansibump.CGA16,
		CharSet:     charmap.CodePage437,
	}
	s, _ := customizer.String(r)
	fmt.Printf("%q\n", s)

	// use xterm palette with codepage 437
	r = strings.NewReader(ansi)
	customizer.Color = ansibump.Xterm16
	s, _ = customizer.String(r)
	fmt.Printf("%q\n", s)
	// Output: "<div style=\"color:#aaa;background-color:#000;\"><span style=\"color:#a50;background-color:#0a0;\">HI</span></div>"
	// "<div style=\"color:#c0c0c0;background-color:#000;\"><span style=\"color:#808000;background-color:#008000;\">HI</span></div>"
}

func ExampleBuffer_codepage() {
	const ansi = "\x1b[0;34;47m\xae\xaf\x1b[0m"
	customizer := ansibump.Customizer{
		Width:       80,
		AmigaParser: false,
		Strict:      false,
		Color:       ansibump.CGA16,
		CharSet:     charmap.CodePage437,
	}
	// using Code Page 437
	r := strings.NewReader(ansi)
	s, _ := customizer.String(r)
	fmt.Printf("%q\n", s)

	// using Latin 1 (ISO-8859-1)
	r = strings.NewReader(ansi)
	customizer.CharSet = charmap.ISO8859_1
	s, _ = customizer.String(r)
	fmt.Printf("%q\n", s)
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
