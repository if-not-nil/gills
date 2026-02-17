package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	flag "github.com/spf13/pflag"
	"golang.org/x/term"
)

var (
	// cat
	flagNumber         bool // -n --number
	flagNumberNonblank bool // -b --number-nonblank
	flagShowEnds       bool // -E --show-ends
	flagShowTabs       bool // -T --show-tabs
	flagSqueezeBlank   bool // -s --squeeze-blank

	// bat
	flagColorWhen string // --color=auto|never|always
	flagTheme     string // --theme
	flagLanguage  string // -l --language
	flagWrapMode  string // --wrap=auto|never|character
	flagWrapWidth int    // --wrap-width
	flagTabs      int    // --tabs
	flagTitles    bool   // --title
	flagTitleNum  bool   // --title-number
	flagOutput    string // -o --output

	// conclusions
	flagColor bool
	flagWrap  bool
)

func main() {
	args()

	out := os.Stdout
	if flagOutput != "" {
		f, err := os.Create(flagOutput)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cant open output file %s: %v\n", flagOutput, err)
			os.Exit(1)
		}
		defer f.Close()
		out = f
	}

	if len(flag.Args()) == 0 {
		stat, err := os.Stdin.Stat()
		if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
			catReader("<stdin>", os.Stdin, 0, out)
			return
		}
		fmt.Fprintln(os.Stderr, "no files given")
		os.Exit(1)
	}

	for n, f := range flag.Args() {
		catFile(f, n, out)
	}
}

func args() {
	// cat
	flag.BoolVarP(&flagNumber, "number", "n", false, "number all output lines")
	flag.BoolVarP(&flagNumberNonblank, "number-nonblank", "b", false, "number nonempty output lines, overrides -n")
	flag.BoolVarP(&flagShowEnds, "show-ends", "E", false, "display $ at end of each line")
	flag.BoolVarP(&flagShowTabs, "show-tabs", "T", false, "display TAB characters as ^I")
	flag.BoolVarP(&flagSqueezeBlank, "squeeze-blank", "s", false, "suppress repeated empty output lines")

	// cat combos
	var showAll bool
	flag.BoolVarP(&showAll, "show-all", "A", false, "equivalent to -TE")
	var eFlag bool
	flag.BoolVarP(&eFlag, "show-nonprinting-ends", "e", false, "equivalent to -E")
	var tFlag bool
	flag.BoolVarP(&tFlag, "show-nonprinting-tabs", "t", false, "equivalent to -T")
	var uFlag bool
	flag.BoolVarP(&uFlag, "unbuffered", "u", false, "ignored (POSIX compatibility)")

	// bat
	flag.StringVar(&flagColorWhen, "color", "auto", "when to use colors: auto, never, always")
	flag.StringVarP(&flagTheme, "theme", "S", "monokai", "set the syntax highlighting theme")
	flag.StringVarP(&flagLanguage, "language", "l", "", "explicitly set the language for syntax highlighting")
	flag.StringVar(&flagWrapMode, "wrap", "auto", "text-wrapping mode: auto, never, character")
	flag.IntVar(&flagWrapWidth, "wrap-width", 0, "wrap width (default: terminal width); implies --wrap=character")
	flag.IntVar(&flagTabs, "tabs", 8, "set the tab width")
	flag.BoolVar(&flagTitles, "title", false, "print a title header for each file")
	flag.BoolVar(&flagTitleNum, "title-number", false, "include file number in title (implies --title)")
	flag.StringVarP(&flagOutput, "output", "o", "", "write output to file instead of stdout")

	var plain bool
	flag.BoolVarP(&plain, "plain", "p", false, "disable decorations: no titles, no line numbers (color still applies)")

	var listThemes bool
	flag.BoolVar(&listThemes, "list-themes", false, "display list of supported themes")
	var listLanguages bool
	flag.BoolVar(&listLanguages, "list-languages", false, "display list of supported languages")

	flag.Parse()

	if showAll {
		flagShowEnds = true
		flagShowTabs = true
	}
	if eFlag {
		flagShowEnds = true
	}
	if tFlag {
		flagShowTabs = true
	}

	switch flagColorWhen {
	case "always":
		flagColor = true
	case "never":
		flagColor = false
	default: // "auto"
		flagColor = shouldUseColor()
	}

	if flagWrapWidth > 0 {
		flagWrap = true
	} else {
		switch flagWrapMode {
		case "character":
			flagWrap = true
		case "never":
			flagWrap = false
		default: // "auto"
			flagWrap = shouldUseColor()
		}
	}

	if flagNumberNonblank {
		flagNumber = false
	}
	if flagTitleNum {
		flagTitles = true
	}
	if plain {
		flagTitles = false
		flagTitleNum = false
		flagNumber = false
		flagNumberNonblank = false
	}

	if listThemes {
		for _, s := range styles.Names() {
			fmt.Printf("- %s\n", s)
		}
		os.Exit(0)
	}
	if listLanguages {
		for _, lx := range lexers.GlobalLexerRegistry.Lexers {
			cfg := lx.Config()
			fmt.Printf("- %-20s aliases=%-30s files=%s\n",
				cfg.Name, strings.Join(cfg.Aliases, ","), strings.Join(cfg.Filenames, ","))
		}
		os.Exit(0)
	}
}

func catFile(fpath string, n int, out io.Writer) {
	f, err := os.Open(fpath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cant open file %s: %v\n", fpath, err)
		return
	}
	defer f.Close()
	catReader(fpath, f, n, out)
}

func catReader(name string, r io.Reader, n int, out io.Writer) {
	if flagTitles && flagTitleNum {
		fmt.Fprintf(out, "== [#%d: %s] ==\n", n+1, name)
	} else if flagTitles {
		fmt.Fprintf(out, "== [%s] ==\n", name)
	}

	raw, err := io.ReadAll(r)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cant read %s: %v\n", name, err)
		return
	}

	var lexer chroma.Lexer
	if flagLanguage != "" {
		lexer = lexers.Get(flagLanguage)
		if lexer == nil {
			fmt.Fprintf(os.Stderr, "unknown language %q, falling back to autodetect\n", flagLanguage)
		}
	}
	if lexer == nil {
		lexer = lexers.Match(name)
	}
	if lexer == nil {
		lexer = lexers.Analyse(string(raw))
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}

	style := styles.Get(flagTheme)
	if style == nil {
		fmt.Fprintf(os.Stderr, "unknown theme %q\n", flagTheme)
		os.Exit(1)
	}

	useColor := flagColor && flagOutput == ""
	var formatter chroma.Formatter
	if useColor {
		formatter = formatters.TTY16m
	} else {
		formatter = formatters.NoOp
	}

	iterator, err := lexer.Tokenise(nil, string(raw))
	if err != nil {
		fmt.Fprintf(os.Stderr, "highlight error: %v\n", err)
		return
	}

	var buf strings.Builder
	if err = formatter.Format(&buf, style, iterator); err != nil {
		fmt.Fprintf(os.Stderr, "format error: %v\n", err)
		return
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	totalLines := len(lines)

	wrapWidth := flagWrapWidth
	if wrapWidth <= 0 {
		wrapWidth = termWidth()
	}

	numWidth := 0
	if flagNumber || flagNumberNonblank {
		numWidth = len(fmt.Sprintf("%d", totalLines)) + 1
	}

	lineNum := 0
	prevBlank := false
	for _, line := range lines {
		isBlank := strings.TrimSpace(stripANSI(line)) == ""

		if flagSqueezeBlank && isBlank && prevBlank {
			continue
		}
		prevBlank = isBlank

		printNum := flagNumber || (flagNumberNonblank && !isBlank)
		if printNum {
			lineNum++
		}

		displayLine := line
		if flagShowTabs {
			displayLine = strings.ReplaceAll(displayLine, "\t", "^I")
		}
		if flagShowEnds {
			displayLine = appendBeforeTrailingReset(displayLine, "$")
		}

		var outputLines []string
		if flagWrap && numWidth < wrapWidth {
			outputLines = wrapLineVisual(displayLine, wrapWidth-numWidth, flagTabs)
		} else {
			outputLines = []string{displayLine}
		}

		indent := strings.Repeat(" ", numWidth)

		for j, l := range outputLines {
			switch {
			case printNum && j == 0:
				if useColor {
					fmt.Fprintf(out, "\x1b[2;37m%*d\x1b[0m %s\n", numWidth-1, lineNum, l)
				} else {
					fmt.Fprintf(out, "%*d %s\n", numWidth-1, lineNum, l)
				}
			case (flagNumber || flagNumberNonblank) && j == 0:
				fmt.Fprintf(out, "%s%s\n", indent, l)
			default:
				if numWidth > 0 && j > 0 {
					fmt.Fprintf(out, "%s%s\n", indent, l)
				} else {
					fmt.Fprintln(out, l)
				}
			}
		}
	}
}

// wraps it and leaves ANSI alone and also handles it right
func wrapLineVisual(line string, width, tabWidth int) []string {
	if width <= 0 {
		return []string{line}
	}

	var result []string
	var cur strings.Builder
	visCol := 0

	i := 0
	for i < len(line) {
		if line[i] == '\x1b' && i+1 < len(line) && line[i+1] == '[' {
			j := i + 2
			for j < len(line) && (line[j] < 0x40 || line[j] > 0x7e) {
				j++
			}
			if j < len(line) {
				j++
			}
			cur.WriteString(line[i:j])
			i = j
			continue
		}

		r, size := utf8.DecodeRuneInString(line[i:])
		rw := runeVisualWidth(r, visCol, tabWidth)

		if visCol+rw > width && visCol > 0 {
			result = append(result, cur.String())
			cur.Reset()
			visCol = 0
		}

		cur.WriteString(line[i : i+size])
		visCol += rw
		i += size
	}

	if cur.Len() > 0 {
		result = append(result, cur.String())
	}
	if len(result) == 0 {
		result = []string{""}
	}
	return result
}

func runeVisualWidth(r rune, col, tabWidth int) int {
	if r == '\t' {
		if tabWidth <= 0 {
			tabWidth = 8
		}
		return tabWidth - (col % tabWidth)
	}
	if r < 0x20 || r == 0x7f {
		return 0
	}
	if r >= 0x1100 {
		if r <= 0x115f || r == 0x2329 || r == 0x232a ||
			(r >= 0x2e80 && r <= 0xa4cf && r != 0x303f) ||
			(r >= 0xac00 && r <= 0xd7a3) ||
			(r >= 0xf900 && r <= 0xfaff) ||
			(r >= 0xfe10 && r <= 0xfe19) ||
			(r >= 0xfe30 && r <= 0xfe6f) ||
			(r >= 0xff00 && r <= 0xff60) ||
			(r >= 0xffe0 && r <= 0xffe6) ||
			(r >= 0x1b000 && r <= 0x1b001) ||
			(r >= 0x1f300 && r <= 0x1f64f) ||
			(r >= 0x1f900 && r <= 0x1f9ff) ||
			(r >= 0x20000 && r <= 0x2fffd) ||
			(r >= 0x30000 && r <= 0x3fffd) {
			return 2
		}
	}
	return 1
}

func stripANSI(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && (s[j] < 0x40 || s[j] > 0x7e) {
				j++
			}
			if j < len(s) {
				j++
			}
			i = j
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func appendBeforeTrailingReset(line, insert string) string {
	if len(line) == 0 {
		return insert
	}
	if line[len(line)-1] == 'm' {
		j := len(line) - 2
		for j >= 0 && line[j] != '\x1b' {
			j--
		}
		if j >= 0 && j+1 < len(line) && line[j+1] == '[' {
			return line[:j] + insert + line[j:]
		}
	}
	return line + insert
}

func shouldUseColor() bool {
	if _, exists := os.LookupEnv("NO_COLOR"); exists {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func termWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 80
	}
	return w
}
