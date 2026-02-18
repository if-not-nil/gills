package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
)

type Kind int

const (
	k_char Kind = iota
	k_line
	k_byte
	k_size
)

type Bound struct {
	Kind Kind
	N    int64
}

func usage() {
	fmt.Println("wrong")
}

func parse_arg(arg string) (name string, lower Bound, upper Bound) {
	parts := strings.Split(arg, "[")
	if len(parts) != 2 {
		fmt.Println("not enough parts")
		usage()
		os.Exit(1)
	}
	filename := parts[0]
	bounds_raw := strings.Trim(parts[1], "[]")
	// i know it doesnt matter but it makes sense to not accept it like that
	if len(bounds_raw) != len(parts[1])-1 {
		fmt.Println(filename, bounds_raw, parts[1])
		fmt.Println("cant split right")
		usage()
		os.Exit(1)
	}

	bounds_str := strings.Split(bounds_raw, ":")
	bounds := slices.Repeat([]Bound{{}}, len(bounds_str))
	for bound_i, bound := range bounds_str {
		fmt.Println(bound, ":")
		until := 0
		var numbers []rune
		for i, c := range bound {
			if c >= '0' && c <= '9' {
				// fmt.Println("hit")
				numbers = append(numbers, c)
			} else {
				// fmt.Println("not hit")
				until = i
				break
			}
		}
		if len(numbers) == 0 {
			numbers = []rune{'1'}
		}
		prefix, err := strconv.ParseInt(string(numbers), 10, 64)
		if err != nil {
			fmt.Println("not a number that you just gave me (shouldnt occur and defaults to once)")
			usage()
		}
		suffix := bound[until:]
		if until == 0 {
			suffix = "c"
		}

		fmt.Println(prefix, suffix, until)

		var kind Kind
		switch suffix {
		case "c":
			kind = k_char
		case "l":
			kind = k_line
		case "b":
			kind = k_byte
		default:
			if suffix[0] == '_' {
				power, err := strconv.ParseInt(suffix[1:], 10, 64)
				if err != nil {
					fmt.Println("_ prefixes powers and must be an integer")
					usage()
				}
				prefix = prefix * int64(math.Pow10(int(power)))
			} else {
				// filesizes
			}
		}
		bounds[bound_i].Kind = kind
		bounds[bound_i].N = prefix
	}
	return filename, bounds[0], bounds[1]
}


func main() {
	flag.Parse()
	for _, arg := range flag.Args() {
		fmt.Println(parse_arg(arg))
		fmt.Println("todo: file reading")
	}
}
