# ANSIbump
[![Go Reference](https://pkg.go.dev/badge/github.com/bengarrett/ansibump.svg)](https://pkg.go.dev/github.com/bengarrett/ansibump)
[![Go Report Card](https://goreportcard.com/badge/github.com/bengarrett/ansibump)](https://goreportcard.com/report/github.com/bengarrett/ansibump)

ANSIbump takes texts encoded with ANSI escape codes and transforms it into a HTML fragment for use in a template or webpage.

See the [reference documentation](https://pkg.go.dev/github.com/bengarrett/ansibump) for usage, and examples, including changing the character sets and the color palette.

ANSIbump was created for and is in use on the website archive Defacto2, home to [thousands of ANSI](https://defacto2.net/files/ansi) texts and artworks that are now rendered as HTML.

Some ANSI highlights that are rendered by ANSIbump:

- [The Game Gallery BBS ad](https://defacto2.net/f/ba2bcbb) from 1985 that's an ANSI rendition of Pacman.
- [Epsilon Nine BBS ad](https://defacto2.net/f/ac22cda) from 1990 of an orbiting space station.
- [Bart Simpson](http://localhost:1323/f/ac29cac) was a popular subject in 1991.
- [Superheros drawings](https://defacto2.net/f/b62d8a3) were very popular themes in the mid-1990s.

#### Quick usage

```go
package main

import (
	"log"
	"os"

	"github.com/bengarrett/ansibump"
)

func main() {
	file, _ := os.Open("file.ans")
	defer file.Close()
	const columns = 80
	_, _ = ansibump.WriteTo(file, os.Stdout, columns)
}
```

#### Not supported or known issues

- ANSI.SYS blinking, [for example](https://defacto2.net/f/a922ed8). CSS blinking uses a [lot of boilerplate](https://github.com/bengarrett/RetroTxt/blob/main/ext/css/text_colors_blink.css) for each color.
- No auto-detection for Amiga controls, `0x30 0x20 0x70` [example](https://defacto2.net/f/a92327d).
- [CR causing newlines](https://defacto2.net/f/a522a2a) and unsupported ESC `[=7l`, `[=7h`.

#### Important

ANSIbump was initially vibe coded, you can see the [original prompt, output, and read the manual changes](https://github.com/bengarrett/ansibump/blob/main/docs/vibe.md).

### Similar projects

- [Deark](https://github.com/jsummers/deark) is a utility that can output ANSI to HTML or an image.
- [ansipants](https://github.com/demozoo/ansipants) a Python module and utility for converting ANSI art to HTML.
- [RetroTxt](https://docs.retrotxt.com/) a web browser extension that renders ANSI as HTML in a tab.
- [Ansilove](https://github.com/ansilove) is a collection of tools to convert ANSI to images.
- [Ultimate Oldschool PC Font Pack](https://int10h.org/oldschool-pc-fonts/) offers various retro DOS and PC fonts.

