# Markdown Element Overview

A short tour of the markdown elements goldmark-pdf renders. Each section corresponds to one element family.

## Headings

`#` through `######` map to H1–H6. The renderer's `DefaultStyles` sizes them 24, 22, 20, 18, 16, 14 points respectively.

# Heading 1

## Heading 2

### Heading 3

#### Heading 4

##### Heading 5

###### Heading 6

## Inline formatting

Body text supports **bold**, *italic*, ***bold italic***, `inline code`, and ~~strikethrough~~ (the last requires the `extension.Strikethrough` extension, which is enabled in the example renderer).

Adjacent formats compose: a **bold word with *italic inside* and back**, or a link with [**bold inside the link text**](https://example.com/).

A line ending with two spaces forces a hard break,  
so this sentence starts on a new line within the same paragraph.

## Links

External links: [goldmark](https://github.com/yuin/goldmark) and [gofpdf](https://github.com/phpdave11/gofpdf). Inline-code link text also works: [`pdf.New()`](https://pkg.go.dev/github.com/stephenafamo/goldmark-pdf#New).

## Lists

Unordered:

- First item
- Second item
- Third item with **bold** and a [link](https://example.com/)

Ordered:

1. Step one
2. Step two
3. Step three

Nested mix (bullets under numbers):

1. Top-level numbered item
   - Nested bullet
   - Another nested bullet
2. Second numbered item
   1. Nested numbered
   2. Another nested numbered

## Blockquotes

> A blockquote is rendered with the `Blockquote` style from
> `DefaultStyles` — a slightly smaller font than body text by default.
>
> Blockquotes can span multiple paragraphs and contain **inline formatting**.

## Code blocks

Fenced code blocks pick up syntax highlighting from chroma. The default
theme is `monokai`.

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, PDF!")
}
```

A shell session:

```sh
$ go run ./examples
2026/05/19 22:56:59 sample [fpdf]  → examples/sample.fpdf.pdf
2026/05/19 22:56:59 sample [gopdf] → examples/sample.gopdf.pdf
```

Indented code (four-space prefix) also works without a language tag:

    untagged := "no syntax highlighting here"
    fmt.Println(untagged)

When a chroma token's text is wider than the remaining line, the renderer
wraps it onto a continuation row. Each continuation row must redraw the left
gutter so the code-block background extends all the way to the left edge

```bash
usage: cbench.py [-h] -cpp CPP [-hwroot HWROOT] [-rundir RUNDIR] [-plusArgsVlog PLUSARGSVLOG] [-plusArgsVcs PLUSARGSVCS] [-plusArgsSim PLUSARGSSIM]
                 [-disableSelfCheck] [-nocomp] [-dump] [-iss] [-rtl_syn] [-rtl_fpga]
```

## Horizontal rule

A `---` on its own line draws a horizontal rule:

---

The rule separates this paragraph from the section above.

## Wrapping behavior

The renderer wraps long paragraphs at the right margin. The following block
is a single paragraph long enough to demonstrate wrapping across several
visual lines: Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed
do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad
minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex
ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate
velit esse cillum dolore eu fugiat nulla pariatur.

See [`tables.md`](./tables.md) for table rendering and [`sample.md`](./sample.md)
for a longer end-to-end document.
