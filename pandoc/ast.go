package pandoc

type KeyVal struct {
	Key string
	Val string
}

type Attr struct {
	Identifier string
	Classes    []string
	KeyVals    []*KeyVal
}

func (a *Attr) KeyValMap() map[string]string {
	ret := make(map[string]string, len(a.KeyVals))
	for _, kv := range a.KeyVals {
		ret[kv.Key] = kv.Val
	}
	return ret
}

func (a *Attr) HasClass(s string) bool {
	for _, c := range a.Classes {
		if c == s {
			return true
		}
	}
	return false
}

type Block interface {
}

type BlockList []Block

type Plain struct {
	Inlines InlineList
}

type Para struct {
	Inlines InlineList
}

type LineBlock struct {
	Lines []InlineList
}

type CodeBlock struct {
	Attr Attr
	Text string
}

type RawBlock struct {
	Format string
	Text   string
}

type BlockQuote struct {
	Blocks BlockList
}

type OrderedList struct {
	StartNumber int
	NumberStyle string
	NumberDelim string
	Items       []BlockList
}

type BulletList struct {
	Items []BlockList
}

type DefinitionItem struct {
	Term        InlineList
	Definitions []BlockList
}

type DefinitionList struct {
	Items []*DefinitionItem
}

type Header struct {
	Level   int
	Attr    Attr
	Inlines InlineList
}

type HorizontalRule struct {
}

type ColSpec struct {
	Alignment string
	ColWidth  float32
}

type Table struct {
	Attr         Attr
	ShortCaption InlineList
	Caption      BlockList
	ColSpecs     []*ColSpec
	Head         TableHeadOrFoot
	Bodies       []*TableBody
	Foot         TableHeadOrFoot
}

type TableHeadOrFoot struct {
	Attr Attr
	Rows []*Row
}

type TableBody struct {
	Attr           Attr
	RowHeadColumns int
	Rows1          []*Row
	Rows2          []*Row
}

type Row struct {
	Attr  Attr
	Cells []*Cell
}

type Cell struct {
	Attr      Attr
	Alignment string
	RowSpan   int
	ColSpan   int
	Blocks    BlockList
}

type Div struct {
	Attr   Attr
	Blocks BlockList
}

type Null struct {
}

type Inline interface {
}

type InlineList []Inline

type Str struct {
	Text string
}

type InlineFmt int

const (
	Emph = InlineFmt(iota)
	Underline
	Strong
	Strikeout
	Superscript
	Subscript
	SmallCaps
)

func (f InlineFmt) String() string {
	switch f {
	case Emph:
		return "Emph"
	case Underline:
		return "Underline"
	case Strong:
		return "Strong"
	case Strikeout:
		return "Strikeout"
	case Superscript:
		return "Superscript"
	case Subscript:
		return "Subscript"
	case SmallCaps:
		return "SmallCaps"
	default:
		return "UnknownSpan"
	}
}

type Formatted struct {
	Fmt     InlineFmt
	Content InlineList
}

type QuoteType int

type Quoted struct {
	QuoteType string // SingleQuote or DoubleQuote
	Content   InlineList
}

type Cite struct {
	Id      string
	Prefix  InlineList
	Suffix  InlineList
	Mode    string // AuthorInText, SuppressAuthor, NormalCitation
	NoteNum int
	Hash    int
	Content InlineList
}

type Code struct {
	Attr Attr
	Text string
}

type Space struct {
}

type SoftBreak struct {
}

type LineBreak struct {
}

type Math struct {
	Type string // DisplayMath, InlineMath
	Text string
}

type RawInline struct {
	Format string // Text
	Text   string
}

type Target struct {
	URL   string
	Title string
}

type Link struct {
	Attr    Attr
	Content InlineList
	Target  Target
}

type Image struct {
	Attr    Attr
	Content InlineList
	Target  Target
}

type Note struct {
	Blocks BlockList
}

type Span struct {
	Attr    Attr
	Content InlineList
}
