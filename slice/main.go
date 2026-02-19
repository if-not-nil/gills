package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"unicode"

	flag "github.com/spf13/pflag"
)

func main() {
	zeroIndexed := flag.BoolP("zero-indexed", "z", false, "use 0-based indexing (default is 1-based)")
	flagExamples := flag.Bool("example", false, "show usage examples and exit")
	flag.Usage = usage
	flag.Parse()
	if *flagExamples {
		example()
		os.Exit(1)
	}

	args := flag.Args()
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}

	for _, arg := range args {
		filename, lower, upper, err := parse_arg(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "slice: %v\n", err)
			os.Exit(1)
		}

		content, err := read_source(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "slice: can't read %q: %v\n", filename, err)
			os.Exit(1)
		}

		result := slice_content(content, lower, upper, *zeroIndexed)
		os.Stdout.Write(result)
	}
}

type Kind int

const (
	k_char Kind = iota
	k_line
	k_byte
	k_word
)

type Bound struct {
	Kind  Kind
	N     uint64
	IsEnd bool
}

// SI (1000 based), IEC (1024 based), 'b' for bytes
var sizeSuffixes = map[string]uint64{
	"b":  1,
	"kb": 1000, "kB": 1000, "KB": 1000,
	"mb": 1000 * 1000, "mB": 1000 * 1000, "MB": 1000 * 1000,
	"gb": 1000 * 1000 * 1000, "gB": 1000 * 1000 * 1000, "GB": 1000 * 1000 * 1000,
	"kib": 1024, "KiB": 1024, "Ki": 1024,
	"mib": 1024 * 1024, "MiB": 1024 * 1024, "Mi": 1024 * 1024,
	"gib": 1024 * 1024 * 1024, "GiB": 1024 * 1024 * 1024, "Gi": 1024 * 1024 * 1024,
}

func usage() {
	fmt.Fprintln(os.Stderr, `slice — print a slice of a file or stdin
you may be looking for 'slice --example'

usage:
  slice [flags] <file>[<lower>:<upper>]
  slice [flags] <file>[<n>]           first n units  (e.g. [10l] = first 10 lines)
  slice [flags] -[<lower>:<upper>]    read from stdin

bound syntax:
  <n>          characters (default)
  <n>c         characters
  <n>l         lines
  <n>w         words
  <n>b         bytes
  <n>kb / kB   kilobytes  (1000)
  <n>KiB / Ki  kibibytes  (1024)
  <n>mb / MB   megabytes  (1000^2)
  <n>MiB / Mi  mebibytes  (1024^2)
  <n>gb / GB   gigabytes  (1000^3)
  <n>GiB / Gi  gibibytes  (1024^3)
  <n>_<p>      n * 10^p chars  (e.g. 1_3 = 1000)

flags:`)
	flag.PrintDefaults()
}

func example() {
	fmt.Fprintln(os.Stderr,
`* slice main.go[1:] # or slice main.go [1]
  whole file starting from the first character 
* slice -[1:]
  standard input from the first character 
* slice main.go[1:] -z # zero-indexed
  whole file starting from the second character
* slice main.go[1l:20l]
  lines 1-20 
* slice main.go[1w:20w]
  words 1-20 
* slice main.go[1:20]
  characters 1-20 
* slice main.go[1_3:2_3]
  characters 1*(10^3)-1*(10^3), so 1000-2000
* slice main.go[1b:20b]
  bytes 1-20 
* slice main.go[1kb:20kb]
  kilobytes 1-20. accepted are kB 
  accepted are b 512, kB 1000, K 1024, MB 1000*1000, M 1024*1024`)
}

func parse_bound(s string) (Bound, error) {
	if s == "" {
		return Bound{IsEnd: true}, nil
	}

	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}

	var n uint64 = 1
	var err error
	if i > 0 {
		n, err = strconv.ParseUint(s[:i], 10, 64)
		if err != nil {
			return Bound{}, fmt.Errorf("invalid number in %q: %w", s, err)
		}
	}

	suffix := s[i:]

	if suffix == "" || suffix == "c" {
		return Bound{Kind: k_char, N: n}, nil
	}

	switch suffix {
	case "l":
		return Bound{Kind: k_line, N: n}, nil
	case "w":
		return Bound{Kind: k_word, N: n}, nil
	}

	if mult, ok := sizeSuffixes[suffix]; ok {
		return Bound{Kind: k_byte, N: n * mult}, nil
	}

	if strings.HasPrefix(suffix, "_") {
		power, err := strconv.ParseInt(suffix[1:], 10, 64)
		if err != nil {
			return Bound{}, fmt.Errorf("'_' must be followed by an integer, got %q", suffix[1:])
		}
		return Bound{Kind: k_char, N: n * uint64(math.Pow10(int(power)))}, nil
	}

	return Bound{}, fmt.Errorf("unknown suffix %q", suffix)
}

func parse_arg(arg string) (filename string, lower, upper Bound, err error) {
	filename, rest, ok := strings.Cut(arg, "[")
	if !ok {
		err = fmt.Errorf("%q is missing '[': expected file[lower:upper]", arg)
		return
	}

	if !strings.HasSuffix(rest, "]") {
		err = fmt.Errorf("%q is missing closing ']'", arg)
		return
	}
	inner := rest[:len(rest)-1]

	// file[nS] into [1S : nS], like [10l] = [1l:10l]
	if !strings.Contains(inner, ":") {
		var b Bound
		b, err = parse_bound(inner)
		if err != nil {
			return
		}
		lower = Bound{Kind: b.Kind, N: 1}
		upper = b
		return
	}

	parts := strings.SplitN(inner, ":", 2)
	if lower, err = parse_bound(parts[0]); err != nil {
		err = fmt.Errorf("lower bound: %w", err)
		return
	}
	if upper, err = parse_bound(parts[1]); err != nil {
		err = fmt.Errorf("upper bound: %w", err)
		return
	}
	return
}

// byte position of the start of unit n 0-based
func start_offset(content []byte, kind Kind, n uint64) int {
	s := string(content)
	switch kind {
	case k_byte:
		if int(n) > len(content) {
			return len(content)
		}
		return int(n)

	case k_char:
		var count uint64
		for i := range s {
			if count == n {
				return i
			}
			count++
		}
		return len(content)

	case k_word:
		var count uint64
		inWord := false
		for i, r := range s {
			if !unicode.IsSpace(r) {
				if !inWord {
					inWord = true
					if count == n {
						return i
					}
					count++
				}
			} else {
				inWord = false
			}
		}
		return len(content)

	case k_line:
		if n == 0 {
			return 0
		}
		var count uint64
		for i, r := range s {
			if r == '\n' {
				count++
				if count == n {
					return i + 1
				}
			}
		}
		return len(content)
	}
	return 0
}

// if isLower then inclusive else exclusive (after last unit)
func bound_to_byte_offset(content []byte, b Bound, zeroIndexed bool, isLower bool) int {
	if b.IsEnd {
		return len(content)
	}

	n := b.N
	if !zeroIndexed {
		if n > 0 {
			n--
		}
	}

	if isLower {
		return start_offset(content, b.Kind, n)
	}
	return start_offset(content, b.Kind, n+1)
}

func slice_content(content []byte, lower, upper Bound, zeroIndexed bool) []byte {
	start := bound_to_byte_offset(content, lower, zeroIndexed, true)
	end := bound_to_byte_offset(content, upper, zeroIndexed, false)

	if start < 0 {
		start = 0
	}
	if end > len(content) {
		end = len(content)
	}
	if start >= end {
		return nil
	}
	return content[start:end]
}

func read_source(filename string) ([]byte, error) {
	if filename != "-" {
		var buf []byte
		r := bufio.NewReader(os.Stdin)
		tmp := make([]byte, 32*1024)
		for {
			n, err := r.Read(tmp)
			buf = append(buf, tmp[:n]...)
			if err != nil {
				break
			}
		}
		return buf, nil
	}
	return os.ReadFile(filename)
}
