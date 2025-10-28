# ANSIbump
[![Go Reference](https://pkg.go.dev/badge/github.com/bengarrett/ansibump.svg)](https://pkg.go.dev/github.com/bengarrett/ansibump)
[![Go Report Card](https://goreportcard.com/badge/github.com/bengarrett/ansibump)](https://goreportcard.com/report/github.com/bengarrett/ansibump)

ANSIbump takes texts encoded with ANSI escape codes and transforms it into a HTML fragment for use in a template or webpage.

See the [reference documentation](https://pkg.go.dev/github.com/bengarrett/ansibump) for usage, and examples, including changing the character sets and the color palette.

ANSIbump was created for and is in use on the website archive Defacto2, home to [thousands of ANSI](https://defacto2.net/files/ansi) texts and artworks that are now rendered in HTML.

Some ANSI highlights that are rendered by ANSIbump:

- [The Game Gallery BBS ad](https://defacto2.net/f/ba2bcbb) from 1985 that's an ANSI rendition of Pacman.
- [Epsilon Nine BBS ad](https://defacto2.net/f/ac22cda) from 1990 of an orbiting space station.
- [Bart Simpson](https://defacto2.net/f/ac29cac) was a popular subject in 1991.
- [Comic character drawings](https://defacto2.net/f/b22decc) were very popular in the 1990s.

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

#### HTML

ANSIbump will output a [`<div>`](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/div) "content division" element containing colors, styles, newlines, and text.
- The div element should be used within a [`<pre>`](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/pre) "preformatted text" element.
- Most ANSI text will want a custom monospaced font, [Cascadia Mono](https://github.com/microsoft/cascadia-code) handles all the [CodePage 437](https://en.wikipedia.org/wiki/Code_page_437) characters. 
- Or use the [IBM VGA font](https://int10h.org/oldschool-pc-fonts/fontlist/font?ibm_vga_8x16) for a more authentic recreation,
 either font will require a CSS [`@font-face`](https://developer.mozilla.org/en-US/docs/Web/CSS/@font-face) rule and [`font-family`](https://developer.mozilla.org/en-US/docs/Web/CSS/font-family) property.

```html
<html>
  <head>
    <title>Quick usage</title>
  </head>
  <style>
    @font-face {
      font-family: cascadia-mono;
      src: url(CascadiaMono.woff2) format("woff2");
    }
    pre {
      font-family: cascadia-mono, monospace, serif;
    }
  </style>
  <body>
    <pre><!--- ansibump output ---><div style="color:#aaa;background-color:#000;">   <span style="color:#a50;background-color:#0a0;">HI‼︎</span>   </div>
    </pre>
  </body>
</html>
```

#### Not supported or known issues

- ANSI.SYS blinking, [for example](https://defacto2.net/f/a922ed8). CSS blinking uses a [lot of boilerplate](https://github.com/bengarrett/RetroTxt/blob/main/ext/css/text_colors_blink.css) for each color.
- No auto-detection for system and programming controls, [example](https://defacto2.net/f/a92327d).

#### Sauce metadata

ANSIbump doesn't parse any SAUCE metadata, however this can be done with a separate [bengarrett/sauce](https://github.com/bengarrett/sauce) package.

#### Important

ANSIbump was initially vibe coded, you can see the [original prompt, output, and read of the many needed manual changes](https://github.com/bengarrett/ansibump/blob/main/docs/vibe.md).

### Similar projects

- [Deark](https://github.com/jsummers/deark) is a utility that can output ANSI to HTML or an image.
- [ansipants](https://github.com/demozoo/ansipants) a Python module and utility for converting ANSI art to HTML.
- [RetroTxt](https://docs.retrotxt.com/) a web browser extension that renders ANSI as HTML in a tab.
- [Ansilove](https://github.com/ansilove) is a collection of tools to convert ANSI to images.
- [Ultimate Oldschool PC Font Pack](https://int10h.org/oldschool-pc-fonts/) offers various retro DOS and PC fonts.

