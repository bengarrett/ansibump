ANSIbump was originally vibe coded on 2025, Oct-26 with GTP-5 mini using the following prompt:

> "in golang create a package that converters ansi escape codes including colors and cursor movement into html"

The bot code has since been heavily modified for ANSIbump.

Some of the many changes that were hand-coded:

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
- Numerous bug fixes 😂
- Quality of life funcs to return HTML as bytes.Buffer, []bytes, or write to an io.Writer
- Updates to the current version of Go.
