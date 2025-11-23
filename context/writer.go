package context

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/adnsv/go-pandoc"
)

// Writer converts Pandoc AST elements to ConTeXt markup. It maintains state during
// conversion including block separation, inline mode, and heading levels.
type Writer struct {
	out         io.Writer // Output writer for ConTeXt markup
	indir       string    // Input directory for resolving relative paths
	blockSep    string    // Separator to insert before next block
	forceInline int       // Counter for forcing inline image placement
	topLevel    int       // Top-level heading mapping (0=part, 1=chapter, 2=section)

	DefaultExternalFigureSize string // Default size constraint for external figures
}

// NewWriter creates a new Writer instance that writes ConTeXt markup to w.
// The indir parameter specifies the input directory for resolving relative image paths.
// The default top-level heading is set to chapter (level 1).
func NewWriter(w io.Writer, indir string) *Writer {
	return &Writer{out: w, indir: indir, topLevel: 1}
}

// SetTopLevelDivision sets the top-level heading division for Markdown level 1 headings.
// Valid values are "part" (level 0), "chapter" (level 1), or "section" (level 2).
// Invalid values default to "section".
func (w *Writer) SetTopLevelDivision(s string) {
	switch s {
	case "part":
		w.topLevel = 0
	case "chapter":
		w.topLevel = 1
	case "section":
		w.topLevel = 2
	default:
		w.topLevel = 2
	}
}

// resolveImageTarget converts a relative image URL to an absolute path.
// Relative paths are resolved relative to the input directory.
func (w *Writer) resolveImageTarget(url string) string {
	if !filepath.IsAbs(url) {
		a, err := filepath.Abs(filepath.Join(w.indir, url))
		if err == nil {
			url = a
		}
	}
	return filepath.ToSlash(url)
}

// wr writes a string to the output writer.
func (w *Writer) wr(s string) {
	fmt.Fprint(w.out, s)
}

// makeHeading generates the appropriate ConTeXt heading command for a given Markdown
// heading level. It adjusts the level based on the top-level division setting.
func (w *Writer) makeHeading(lvl int) string {
	lvl += w.topLevel
	switch lvl {
	case 1:
		return "\\part"
	case 2:
		return "\\chapter"
	default:
		if lvl < 3 || lvl > 10 {
			return ""
		}
		return "\\" + strings.Repeat("sub", lvl-3) + "section"
	}
}

// writeRow outputs a table row with the specified style. It handles cell alignment
// and column spanning using ConTeXt's xtable system.
func (w *Writer) writeRow(table *pandoc.Table, row *pandoc.Row, style string) {
	if style != "" {
		style = "[" + style + "]"
	}
	w.wr("\\startxrow" + style)
	for i, c := range row.Cells {
		styles := []string{}
		alignmentHack := ""
		if i < len(table.ColSpecs) {
			switch table.ColSpecs[i].Alignment {
			case "AlignCenter":
				alignmentHack = "\u200B" // zero-width space
				styles = append(styles, "align=center")
			case "AlignRight":
				alignmentHack = "\u200B" // zero-width space
				styles = append(styles, "align=flushright")
			}
		}
		sty := ""
		if len(styles) > 0 {
			sty = "[" + strings.Join(styles, ",") + "]"
		}
		w.wr("\n\\startxcell" + sty)
		w.blockSep = "\n" + alignmentHack
		w.WriteBlocks(c.Blocks)
		w.wr("\n\\stopxcell")
	}
	w.wr("\n\\stopxrow")
}

// writeTable converts a Pandoc table to ConTeXt's xtable format. It handles table
// headers, bodies, footers, and captions.
func (w *Writer) writeTable(table *pandoc.Table) {
	w.wr("\\startplacetable[")
	if table.Caption == nil {
		w.wr("location={here,none}")
	} else {
		w.wr("title={")
		// looks pandoc produces a single Plain element here
		w.blockSep = ""
		w.WriteBlocks(table.Caption)
		w.wr("}")
	}
	w.wr("]")
	w.forceInline++
	w.wr("\n\\startxtable")
	if len(table.Head.Rows) > 0 {
		w.wr("\n\\startxtablehead")
		for _, r := range table.Head.Rows {
			w.wr("\n")
			w.writeRow(table, r, "head")
		}
		w.wr("\n\\stopxtablehead")
	}

	for j, tb := range table.Bodies {
		// todo: figure out how exactly pandoc works here
		w.wr("\n\\startxtablebody")
		for i, r := range tb.Rows2 {
			w.wr("\n")

			last := j == len(table.Bodies)-1 && i == len(tb.Rows2)-1

			if last {
				w.writeRow(table, r, "lastbody")
			} else {
				w.writeRow(table, r, "body")
			}
		}
		w.wr("\n\\stopxtablebody")
	}

	if len(table.Foot.Rows) > 0 {
		w.wr("\n\\startxtablefoot")
		for _, r := range table.Foot.Rows {
			w.wr("\n")
			w.writeRow(table, r, "foot")
		}
		w.wr("\n\\stopxtablefoot")
	}
	w.wr("\n\\stopxtable")
	w.forceInline--
	w.wr("\n\\stopplacetable")
}

// writeDiv processes Pandoc Div blocks with special class handling for layout features.
// Supports HSTACK (horizontal layout), narrower (text narrowing), combination
// (figure combination), and columns (multi-column layout).
func (w *Writer) writeDiv(div *pandoc.Div) {
	kv := div.Attr.KeyValMap()
	w.blockSep = ""

	if div.Attr.HasClass("HSTACK") {
		w.wr("\\startxtable\\startxrow\\startxcell")
		w.blockSep = "\n"
		for _, b := range div.Blocks {
			if _, ok := b.(*pandoc.HorizontalRule); ok {
				w.wr("\n\\stopxcell\\startxcell")
				w.blockSep = "\n"
				continue
			}
			w.wr(w.blockSep)
			w.writeBlock(b)
		}

		w.wr("\n\\stopxcell\\stopxrow\\stopxtable")
		return
	}

	n := kv["narrower"]
	if n != "" {
		w.wr("\\startnarrow[middle=" + n + "]\n")
	}

	if c := kv["combination"]; c != "" {
		w.wr("\\startcombination[" + c + "]")
		w.blockSep = "\n"
		w.forceInline++
		w.WriteBlocks(div.Blocks)
		w.forceInline--
		w.wr("\n\\stopcombination")
	} else if c = kv["columns"]; c != "" {
		w.wr("\\startcolumns[" + c + "]")
		w.blockSep = "\n"
		w.WriteBlocks(div.Blocks)
		w.wr("\n\\stopcolumns")
	} else {
		w.WriteBlocks(div.Blocks)
	}

	if n != "" {
		w.wr("\n\\stopnarrow")
	}
}

// handleAlert detects GitHub-style alert syntax in blockquotes:
// > [!NOTE]
// > Content here
func (w *Writer) handleAlert(blocks []pandoc.Block) bool {
	if len(blocks) == 0 {
		return false
	}

	// First block should be a paragraph
	para, ok := blocks[0].(*pandoc.Para)
	if !ok {
		return false
	}

	// Check if first inline matches [!TYPE] pattern
	if len(para.Inlines) == 0 {
		return false
	}

	str, ok := para.Inlines[0].(*pandoc.Str)
	if !ok {
		return false
	}

	// Check for [!TYPE] pattern
	text := str.Text
	if !strings.HasPrefix(text, "[!") || !strings.HasSuffix(text, "]") {
		return false
	}

	// Extract alert type
	alertType := text[2 : len(text)-1]

	// Map GitHub alert types to ConTeXt environments
	var envType string
	var title string
	var icon string
	var color string
	switch strings.ToUpper(alertType) {
	case "NOTE":
		envType = "NOTE"
		title = "Note"
		icon = "\\NoteIcon"
		color = "AlertNoteColor"
	case "TIP":
		envType = "TIP"
		title = "Tip"
		icon = "\\TipIcon"
		color = "AlertTipColor"
	case "IMPORTANT":
		envType = "IMPORTANT"
		title = "Important"
		icon = "\\ImportantIcon"
		color = "AlertImportantColor"
	case "WARNING":
		envType = "WARNING"
		title = "Warning"
		icon = "\\WarningIcon"
		color = "AlertWarningColor"
	case "CAUTION":
		envType = "CAUTION"
		title = "Caution"
		icon = "\\CautionIcon"
		color = "AlertCautionColor"
	default:
		return false
	}

	// Output ConTeXt environment with icon and styled heading
	w.wr("\\start" + envType)
	w.wr("{\\color[" + color + "]{" + icon + "\\space\\raise1.5pt\\hbox{\\ss\\bf " + title + "}}}")
	w.wr("\n\\blank[small]\n")

	// Write remaining content from first paragraph (after [!TYPE])
	if len(para.Inlines) > 1 {
		w.blockSep = "\n"
		w.WriteInlines(para.Inlines[1:])
	}

	// Write remaining blocks
	if len(blocks) > 1 {
		w.blockSep = "\n"
		w.WriteBlocks(blocks[1:])
	}

	w.wr("\n\\stop" + envType)
	return true
}

// writeBlock converts a single Pandoc block element to ConTeXt markup.
// It handles paragraphs, code blocks, lists, tables, headings, and other block-level elements.
func (w *Writer) writeBlock(b pandoc.Block) {
	switch b := b.(type) {
	case *pandoc.Plain:
		w.WriteInlines(b.Inlines)
		w.blockSep = "\n\n"

	case *pandoc.Para:
		w.WriteInlines(b.Inlines)
		w.blockSep = "\n\n"

	case *pandoc.LineBlock:
		for i, ll := range b.Lines {
			if i > 0 {
				w.wr("\n")
			}
			w.WriteInlines(ll)
		}
		w.blockSep = "\n\n"

	case *pandoc.CodeBlock:
		w.wr("\\starttyping")
		if len(b.Attr.Classes) > 0 {
			lang := b.Attr.Classes[0]
			w.wr("[option=" + lang + "]")
		}
		w.wr("\n")
		w.wr(b.Text)
		w.wr("\n\\stoptyping")
		w.blockSep = "\n\n"

	case *pandoc.BlockQuote:
		// Check for GitHub-style alerts first
		if !w.handleAlert(b.Blocks) {
			// Fall back to standard blockquote
			w.wr("\\startblockquote")
			w.blockSep = "\n"
			w.WriteBlocks(b.Blocks)
			w.wr("\n\\stopblockquote")
		}
		w.blockSep = "\n\n"

	case *pandoc.OrderedList:
		w.wr("\\startitemize[n,packed][stopper=.]")
		for _, bb := range b.Items {
			w.wr("\n\\item\n")
			w.blockSep = ""
			w.WriteBlocks(bb)
			w.blockSep = "\n\n"
		}
		w.wr("\n\\stopitemize")
		w.blockSep = "\n\n"

	case *pandoc.BulletList:
		w.wr("\\startitemize")
		for _, bb := range b.Items {
			w.wr("\n\\item\n")
			w.blockSep = ""
			w.WriteBlocks(bb)
			w.blockSep = "\n\n"
		}
		w.wr("\n\\stopitemize")
		w.blockSep = "\n\n"

	case *pandoc.DefinitionList:
		for i, item := range b.Items {
			if i > 0 {
				w.wr("\n\n")
			}
			w.wr("\\startdescription{")
			w.WriteInlines(item.Term)
			w.wr("}")
			w.blockSep = "\n"
			for _, bb := range item.Definitions {
				w.WriteBlocks(bb)
			}
			w.wr("\n\\stopdescription")
			w.blockSep = "\n\n"
		}

	case *pandoc.Header:
		w.wr(w.makeHeading(b.Level))
		if b.Attr.Identifier != "" {
			w.wr("[" + b.Attr.Identifier + "]")
		}
		w.wr("{")
		w.WriteInlines(b.Inlines)
		w.wr("}")
		w.blockSep = "\n\n"

	case *pandoc.HorizontalRule:
		w.wr("\\thinrule")
		w.blockSep = "\n\n"

	case *pandoc.Table:
		w.writeTable(b)
		w.blockSep = "\n\n"

	case *pandoc.Div:
		w.writeDiv(b)
		w.blockSep = "\n\n"

	case *pandoc.RawBlock:
		if b.Format == "tex" {
			w.wr(b.Text)
			w.blockSep = "\n\n"
		}

	default:
		w.blockSep = ""
	}
}

// WriteBlocks converts a sequence of Pandoc blocks to ConTeXt markup, inserting
// appropriate block separators between elements.
func (w *Writer) WriteBlocks(bb []pandoc.Block) {
	for _, b := range bb {
		w.wr(w.blockSep)
		w.writeBlock(b)
	}
}

// writeExternalFigure generates a ConTeXt \externalfigure command for an image.
// It handles image paths, size constraints, and offset positioning.
func (w *Writer) writeExternalFigure(img *pandoc.Image) {
	fn := img.Target.URL
	fn = w.resolveImageTarget(fn)
	ext := strings.ToLower(filepath.Ext(fn))

	kv := img.Attr.KeyValMap()
	if dx, dy := kv["dx"], kv["dy"]; dx != "" || dy != "" {
		w.wr("\\offset[")
		if dx != "" {
			w.wr(fmt.Sprintf("x=%s", dx))
			if dy != "" {
				w.wr(",")
			}
		}
		if dy != "" {
			w.wr(fmt.Sprintf("y=%s", dy))
		}
		w.wr("]")
	}

	w.wr("{\\externalfigure[")
	w.wr(fn)
	w.wr("]")
	attrs := []string{}
	attrs = append(attrs, "conversion=mp")

	haveWidth := false
	haveHeight := false
	if len(kv) > 0 {
		if s := kv["width"]; s != "" {
			n, u, err := splitNumUnits(s)
			if err == nil {
				switch u {
				case "%":
					u = "\\textwidth"
					n /= 100.0
				case "inch":
					u = "in"
				}
				attrs = append(attrs, fmt.Sprintf("width=%f%s", n, u))
			}
			haveWidth = true
		}
		if s := kv["height"]; s != "" {
			n, u, err := splitNumUnits(s)
			if err == nil {
				switch u {
				case "%":
					u = "\\textheight"
					n /= 100.0
				case "inch":
					u = "in"
				}
				attrs = append(attrs, fmt.Sprintf("height=%f%s", n, u))
			}
			haveWidth = true
		}
	}
	if !haveWidth && !haveHeight && ext != ".svg" && w.DefaultExternalFigureSize != "" {
		attrs = append(attrs, w.DefaultExternalFigureSize)
	}
	if len(attrs) > 0 {
		w.wr("[" + strings.Join(attrs, ",") + "]")
	}
	w.wr("}")
}

// writeImage generates a ConTeXt \placefigure command for a floating figure with caption.
// For inline images, use writeExternalFigure instead.
func (w *Writer) writeImage(img *pandoc.Image) {
	options := []string{}

	if len(img.Content) == 0 {
		options = append(options, "none")
	}

	if opts := img.Attr.KeyValMap()["options"]; opts != "" {
		options = append(options, strings.Split(opts, ",")...)
	}

	w.wr("\\placefigure")

	// options
	w.wr("[" + strings.Join(options, ",") + "]")

	// references
	w.wr("[" + img.Attr.Identifier + "]")

	// title
	w.wr("{")
	w.WriteInlines(img.Content)
	w.wr("}")

	// image
	w.writeExternalFigure(img)
}

// FlattenInlines converts a list of inline elements to a plain string with ConTeXt escaping.
// This is used for generating link text and other contexts where formatted text is needed
// as a simple string.
func FlattenInlines(ll pandoc.InlineList) string {
	buf := &bytes.Buffer{}
	for _, l := range ll {
		switch l := l.(type) {
		case *pandoc.Space:
			buf.WriteString(" ")
		case *pandoc.SoftBreak:
			buf.WriteString("\n")
		case *pandoc.LineBreak:
			buf.WriteString("\n")
		case *pandoc.Str:
			buf.WriteString(EscapeStr(l.Text))
		case *pandoc.Formatted:
			buf.WriteString(FlattenInlines(l.Content))
		case *pandoc.Quoted:
			if l.QuoteType == "SingleQuote" {
				buf.WriteString("\\quote{")
			} else if l.QuoteType == "DoubleQuote" {
				buf.WriteString("\\quotation{")
			} else {
				buf.WriteString("{")
			}
			buf.WriteString(FlattenInlines(l.Content))
			buf.WriteString("}")
		case *pandoc.RawInline:
			buf.WriteString(l.Text)
		}
	}
	return buf.String()
}

// WriteInlines converts a list of Pandoc inline elements to ConTeXt markup.
// It handles text, formatting, links, images, math, and other inline elements.
func (w *Writer) WriteInlines(ll pandoc.InlineList) {
	for _, l := range ll {
		switch l := l.(type) {
		case *pandoc.Space:
			w.wr(" ")

		case *pandoc.SoftBreak:
			w.wr("\n")

		case *pandoc.LineBreak:
			w.wr("\\crlf\n")

		case *pandoc.Str:
			w.wr(EscapeStr(l.Text))

		case *pandoc.Formatted:
			w.wr(contextFmt(l.Fmt))
			w.WriteInlines(l.Content)
			w.wr("}")

		case *pandoc.Quoted:
			if l.QuoteType == "SingleQuote" {
				w.wr("\\quote{")
			} else if l.QuoteType == "DoubleQuote" {
				w.wr("\\quotation{")
			} else {
				w.wr("{")
			}
			w.WriteInlines(l.Content)
			w.wr("}")

		case *pandoc.Code:
			if strings.ContainsAny(l.Text, "\\~%$#{}") {
				w.wr("\\mono{")
				w.wr(EscapeStr(l.Text))
				w.wr("}")
			} else {
				w.wr("\\type{")
				w.wr(l.Text)
				w.wr("}")
			}

		case *pandoc.Math:
			if l.Type == "DisplayMath" {
				w.wr("\\startformula ")
				w.wr(l.Text)
				w.wr(" \\stopformula")
			} else {
				w.wr("$")
				w.wr(l.Text)
				w.wr("$")
			}

		case *pandoc.RawInline:
			w.wr(l.Text)

		case *pandoc.Image:
			kvs := l.Attr.KeyValMap()
			if kvs["placement"] == "inline" || w.forceInline > 0 {
				w.writeExternalFigure(l)
			} else {
				w.writeImage(l)
			}

		case *pandoc.Link:
			{
				c := FlattenInlines(l.Content)
				if c != "" {
					if strings.HasSuffix(c, "\\#") {
						c = strings.TrimSuffix(c, "\\#")
						w.wr("\\in{")
						w.wr(EscapeStr(c))
						w.wr("}[")
						s := strings.TrimPrefix(l.Target.URL, "#")
						w.wr(s)
						w.wr(("]"))
						break
					}
				}

				w.wr("\\goto{")
				w.WriteInlines(l.Content)
				w.wr("}[url(")
				s := strings.TrimPrefix(l.Target.URL, "#")
				w.wr(s)
				w.wr((")]"))
			}

			// todo Link, Cite, Span

		}

	}
}

var escaper = strings.NewReplacer(
	`#`, `\#`,
	`$`, `\$`,
	`%`, `\letterpercent{}`,
	`&`, `\&`,
	`\`, `\letterbackslash{}`,
	`{`, `\{`,
	`}`, `\}`,
	`~`, `\~`,
)

// EscapeStr escapes special ConTeXt characters in a string. It handles backslash,
// hash, dollar, percent, ampersand, braces, and tilde characters.
func EscapeStr(s string) string {
	if strings.ContainsAny(s, "\\~%$#{}") {
		s = escaper.Replace(s)
	}
	return s
}

// contextFmt converts a Pandoc inline format type to the corresponding ConTeXt command.
// It handles emphasis, bold, underline, strikeout, superscript, subscript, and small caps.
func contextFmt(f pandoc.InlineFmt) string {
	switch f {
	case pandoc.Emph:
		return "{\\em "
	case pandoc.Underline:
		return "\\underbar{"
	case pandoc.Strong:
		return "{\\bf "
	case pandoc.Strikeout:
		return "\\overstrike{"
	case pandoc.Superscript:
		return "\\high{"
	case pandoc.Subscript:
		return "\\low{"
	case pandoc.SmallCaps:
		return "{\\sc "
	default:
		return "{"
	}
}

// splitNumUnits parses a size string into a numeric value and unit.
// Supported units are %, px, cm, mm, in, inch, pt, and em.
func splitNumUnits(s string) (n float32, u string, err error) {
	for _, p := range []string{"%", "px", "cm", "mm", "in", "inch", "pt", "em"} {
		if strings.HasSuffix(s, p) {
			u = p
			s = s[:len(s)-len(p)]
			break
		}
	}
	v, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 0, "", err
	}
	n = float32(v)
	return
}
