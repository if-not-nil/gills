package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
)

func usage() {
	fmt.Fprintln(os.Stderr, "usage: choice [flags] [choices]")
	fmt.Fprintln(os.Stderr, "output: random item(s) from args or stdin (one item per line)")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "flags:")
	fmt.Fprintln(os.Stderr, "  -i[N]        random integer from 0 to N (default 1)")
	fmt.Fprintln(os.Stderr, "  -f[N]        random float from 0 to N (default 1)")
	fmt.Fprintln(os.Stderr, "  -n K         pick K items (default 1)")
	fmt.Fprintln(os.Stderr, "  -nu K        pick K unique items (no repeats)")
	fmt.Fprintln(os.Stderr, "  -s           shuffle: print all choices in random order")
	fmt.Fprintln(os.Stderr, "  -d DELIM     join output with DELIM instead of newlines")
	fmt.Fprintln(os.Stderr, "  -x VAL       exclude VAL from choices (repeatable)")
	fmt.Fprintln(os.Stderr, "  -S SEED      seed for reproducible output")
	fmt.Fprintln(os.Stderr, "  -c           print count of choices and exit")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "weighted choices: suffix with :N e.g. 'a:3 b:1'")
	os.Exit(1)
}

func example() {
	fmt.Fprintln(os.Stderr, "examples:")
	fmt.Fprintln(os.Stderr, "  choice a b c                    pick one of a, b, c")
	fmt.Fprintln(os.Stderr, "  choice -n3 a b c d e            pick 3 (with replacement)")
	fmt.Fprintln(os.Stderr, "  choice -nu3 a b c d e           pick 3 unique")
	fmt.Fprintln(os.Stderr, "  choice -s a b c d               shuffle all")
	fmt.Fprintln(os.Stderr, "  choice -n3 -d, a b c d e        pick 3, comma-separated")
	fmt.Fprintln(os.Stderr, "  choice -x b a b c               pick from a, c only")
	fmt.Fprintln(os.Stderr, "  choice 'rare:1' 'common:9'      weighted pick")
	fmt.Fprintln(os.Stderr, "  choice -S42 a b c               reproducible output")
	fmt.Fprintln(os.Stderr, "  choice -i100                    random int 0-100")
	fmt.Fprintln(os.Stderr, "  choice -f3.14                   random float 0-3.14")
	fmt.Fprintln(os.Stderr, "  choice -c a b c d               print 4")
	fmt.Fprintln(os.Stderr, "  cat words.txt | choice          pick random line from file")
	fmt.Fprintln(os.Stderr, "  cat words.txt | choice -s       shuffle file lines")
	fmt.Fprintln(os.Stderr, "  cat words.txt | choice -nu5     pick 5 unique lines")
	fmt.Fprintln(os.Stderr, "  cat words.txt | choice a b      pick from file lines + a, b")
	os.Exit(1)
}

func isatty() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return true
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func stdinLines() []string {
	var lines []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

type item struct {
	value  string
	weight int
}

func parseItems(args []string) []item {
	items := make([]item, 0, len(args))
	for _, a := range args {
		// split on last colon to allow values like "http://foo:3"
		idx := strings.LastIndex(a, ":")
		if idx > 0 {
			w, err := strconv.Atoi(a[idx+1:])
			if err == nil && w > 0 {
				items = append(items, item{a[:idx], w})
				continue
			}
		}
		items = append(items, item{a, 1})
	}
	return items
}

func expandWeights(items []item) []string {
	var out []string
	for _, it := range items {
		for i := 0; i < it.weight; i++ {
			out = append(out, it.value)
		}
	}
	return out
}

func printResults(results []string, delim string) {
	if delim == "" {
		for _, r := range results {
			fmt.Println(r)
		}
	} else {
		fmt.Println(strings.Join(results, delim))
	}
}

func main() {
	args := os.Args[1:]

	if len(args) > 0 && (args[0] == "-example") {
		example()
		os.Exit(1)
	}
	// -i and -f: numeric range, no choices needed
	if len(args) >= 1 && strings.HasPrefix(args[0], "-i") {
		var upper int64 = 1
		if len(args[0]) > 2 {
			var err error
			upper, err = strconv.ParseInt(args[0][2:], 10, 64)
			if err != nil || upper < 1 {
				usage()
			}
		}
		fmt.Println(rand.Int63n(upper))
		return
	}
	if len(args) >= 1 && strings.HasPrefix(args[0], "-f") {
		var upper float64 = 1
		if len(args[0]) > 2 {
			var err error
			upper, err = strconv.ParseFloat(args[0][2:], 64)
			if err != nil || upper <= 0 {
				usage()
			}
		}
		fmt.Println(rand.Float64() * upper)
		return
	}

	// parse flags
	var (
		pickN     int = 1
		unique    bool
		shuffle   bool
		delim     string
		excludes  []string
		seed      int64
		hasSeed   bool
		countOnly bool
	)

	rest := []string{}
	i := 0
	for i < len(args) {
		a := args[i]
		switch {
		case a == "-s":
			shuffle = true
		case a == "-c":
			countOnly = true
		case strings.HasPrefix(a, "-nu"):
			unique = true
			if len(a) > 3 {
				n, err := strconv.Atoi(a[3:])
				if err != nil || n < 1 {
					usage()
				}
				pickN = n
			} else {
				i++
				if i >= len(args) {
					usage()
				}
				n, err := strconv.Atoi(args[i])
				if err != nil || n < 1 {
					usage()
				}
				pickN = n
			}
		case strings.HasPrefix(a, "-n"):
			if len(a) > 2 {
				n, err := strconv.Atoi(a[2:])
				if err != nil || n < 1 {
					usage()
				}
				pickN = n
			} else {
				i++
				if i >= len(args) {
					usage()
				}
				n, err := strconv.Atoi(args[i])
				if err != nil || n < 1 {
					usage()
				}
				pickN = n
			}
		case a == "-d":
			i++
			if i >= len(args) {
				usage()
			}
			delim = args[i]
		case a == "-x":
			i++
			if i >= len(args) {
				usage()
			}
			excludes = append(excludes, args[i])
		case strings.HasPrefix(a, "-S"):
			if len(a) > 2 {
				s, err := strconv.ParseInt(a[2:], 10, 64)
				if err != nil {
					usage()
				}
				seed = s
				hasSeed = true
			} else {
				i++
				if i >= len(args) {
					usage()
				}
				s, err := strconv.ParseInt(args[i], 10, 64)
				if err != nil {
					usage()
				}
				seed = s
				hasSeed = true
			}
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(os.Stderr, "unknown flag: %s\n", a)
			usage()
		default:
			rest = append(rest, a)
		}
		i++
	}

	// collect choices
	rawChoices := rest
	if len(rawChoices) == 0 {
		if isatty() {
			fmt.Fprintln(os.Stderr, "no choices given")
			usage()
		}
		rawChoices = stdinLines()
	} else if !isatty() {
		rawChoices = append(rawChoices, stdinLines()...)
	}

	// parse weights
	items := parseItems(rawChoices)

	// apply excludes
	if len(excludes) > 0 {
		excSet := make(map[string]bool, len(excludes))
		for _, e := range excludes {
			excSet[e] = true
		}
		filtered := items[:0]
		for _, it := range items {
			if !excSet[it.value] {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}

	if len(items) == 0 {
		fmt.Fprintln(os.Stderr, "no choices available")
		os.Exit(1)
	}

	// -c: just print count
	if countOnly {
		fmt.Println(len(items))
		return
	}

	// seed rng
	var rng *rand.Rand
	if hasSeed {
		rng = rand.New(rand.NewSource(seed))
	} else {
		rng = rand.New(rand.NewSource(rand.Int63()))
	}

	// expand weights into flat pool
	pool := expandWeights(items)

	// shuffle: print everything in random order (weights respected by repetition)
	if shuffle {
		rng.Shuffle(len(pool), func(i, j int) {
			pool[i], pool[j] = pool[j], pool[i]
		})
		// deduplicate while preserving shuffle order (weights just biased position)
		seen := make(map[string]bool)
		var result []string
		for _, v := range pool {
			if !seen[v] {
				seen[v] = true
				result = append(result, v)
			}
		}
		printResults(result, delim)
		return
	}

	// pick N unique
	if unique {
		if pickN > len(items) {
			fmt.Fprintf(os.Stderr, "not enough unique choices: need %d, have %d\n", pickN, len(items))
			os.Exit(1)
		}
		// shuffle pool, take first pickN unique values
		rng.Shuffle(len(pool), func(i, j int) {
			pool[i], pool[j] = pool[j], pool[i]
		})
		seen := make(map[string]bool)
		var result []string
		for _, v := range pool {
			if !seen[v] {
				seen[v] = true
				result = append(result, v)
			}
			if len(result) == pickN {
				break
			}
		}
		printResults(result, delim)
		return
	}

	// pick N (with replacement)
	result := make([]string, pickN)
	for j := 0; j < pickN; j++ {
		result[j] = pool[rng.Intn(len(pool))]
	}
	printResults(result, delim)
}
