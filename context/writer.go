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

type Writer struct {
	out         io.Writer
	indir       string
	blockSep    string
	forceInline int
	topLevel    int

	DefaultExternalFigureSize string // used to constrain externalfigure size when neigher its width nor height is specified
}

func NewWriter(w io.Writer, indir string) *Writer {
	return &Writer{out: w, indir: indir, topLevel: 1}
}

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

func (w *Writer) resolveImageTarget(url string) string {
	if !filepath.IsAbs(url) {
		a, err := filepath.Abs(filepath.Join(w.indir, url))
		if err == nil {
			url = a
		}
	}
	return filepath.ToSlash(url)
}

func (w *Writer) wr(s string) {
	fmt.Fprint(w.out, s)
}

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

func (w *Writer) handleAdmonition(ll pandoc.InlineList) bool {
	if len(ll) < 3 {
		return false
	}

	f, ok := ll[0].(*pandoc.Formatted)
	if !ok {
		return false
	}
	_, ok = ll[1].(*pandoc.Space)
	if !ok {
		return false
	}

	if f.Fmt == pandoc.Strong {
		if len(f.Content) == 1 {
			if s, ok := f.Content[0].(*pandoc.Str); ok && len(s.Text) > 2 && s.Text[0] == '!' {
				p := s.Text[1:]
				ft := "NOTE"
				if p[0] == '!' {
					p = p[1:]
					ft = "WARNING"
				}
				w.wr("\\start" + ft + "{" + EscapeStr(p) + "}\n")
				w.WriteInlines(ll[2:])
				w.wr("\n\\stop" + ft)
				return true
			}
		}
	}
	return false
}

func (w *Writer) writeBlock(b pandoc.Block) {
	switch b := b.(type) {
	case *pandoc.Plain:
		w.WriteInlines(b.Inlines)
		w.blockSep = "\n\n"

	case *pandoc.Para:
		if !w.handleAdmonition(b.Inlines) {
			w.WriteInlines(b.Inlines)
		}
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
		w.wr("\\startblockquote")
		w.blockSep = "\n"
		w.WriteBlocks(b.Blocks)
		w.wr("\n\\stopblockquote")
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

func (w *Writer) WriteBlocks(bb []pandoc.Block) {
	for _, b := range bb {
		w.wr(w.blockSep)
		w.writeBlock(b)
	}
}

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

func EscapeStr(s string) string {
	if strings.ContainsAny(s, "\\~%$#{}") {
		s = escaper.Replace(s)
	}
	return s
}

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
