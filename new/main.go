package main

// sample -
//

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func example() {
	fmt.Fprintln(os.Stderr, `new -rwx install - creates a new file called install with rwx perms for all groups
new -rw install - creates install with rw-rw-rw-
new -rwxRWxrwx install - rwx--xrwx
new -Rrxxx install - -wxrwxrwx (R disables read for user, lowercase enable for group+other)
new -rwx build/ - creates a build/ dir with rwx for all groups
new build/asdf - creates build/ dir and asdf inside with no perms specified (0000)
new -rwx src/main/asdf - recursively creates src/ and main/ then asdf with rwxrwxrwx
new -x asdf - if asdf exists, adds x for all groups; if new, creates with --x--x--x
new -X asdf - removes x from all groups on existing file
new -rwx asdf nasdf - creates both asdf and nasdf with rwxrwxrwx
new -rwx asdf -rw nasdf - asdf gets rwxrwxrwx, nasdf gets rw-rw-rw-package main
new go run . aaa/addsaf/asdf/asd/f - created aaa/addsaf/asdf/asd/f (0000) `)

}

type BitState int

const (
	Unspecified BitState = iota
	On
	Off
)

type Perms struct {
	read    BitState
	write   BitState
	execute BitState
}

type Request struct {
	name      string
	is_dir    bool
	perms     [3]Perms
	overwrite bool
	has_perms bool
}

func apply_perms(base fs.FileMode, perms [3]Perms) fs.FileMode {
	res := base
	for i, p := range perms {
		shift := uint(6 - i*3)
		apply := func(s BitState, bit fs.FileMode) {
			switch s {
			case On:
				res |= bit << shift
			case Off:
				res &^= bit << shift
			}
		}
		apply(p.read, 4)
		apply(p.write, 2)
		apply(p.execute, 1)
	}
	return res
}

func (req *Request) parse_flag(flag string) {
	state := func(f rune) BitState {
		if f >= 'a' && f <= 'z' {
			return On
		}
		return Off
	}

	n_used := map[rune]uint{'r': 0, 'w': 0, 'x': 0}
	for _, f := range flag[1:] {
		lower := f | ' '
		if n_used[lower] >= 3 {
			fmt.Printf("flag '%c' used too many times\n", lower)
			usage()
		}
		mask := &req.perms[n_used[lower]]
		switch lower {
		case 'r':
			mask.read = state(f)
		case 'w':
			mask.write = state(f)
		case 'x':
			mask.execute = state(f)
		case 'o':
			req.overwrite = state(f) == On
		default:
			fmt.Printf("unknown flag character: %c\n", f)
			usage()
		}
		n_used[lower]++
	}

	// if a bit was only specified once, copy it to groups 1 and 2
	type field struct {
		get func(*Perms) BitState
		set func(*Perms, BitState)
	}
	fields := []field{
		{
			func(p *Perms) BitState { return p.read },
			func(p *Perms, v BitState) { p.read = v },
		},
		{
			func(p *Perms) BitState { return p.write },
			func(p *Perms, v BitState) { p.write = v },
		},
		{
			func(p *Perms) BitState { return p.execute },
			func(p *Perms, v BitState) { p.execute = v },
		},
	}
	letters := []rune{'r', 'w', 'x'}
	for i, l := range letters {
		if n_used[l] == 1 {
			v := fields[i].get(&req.perms[0])
			fields[i].set(&req.perms[1], v)
			fields[i].set(&req.perms[2], v)
		}
	}

	req.has_perms = true
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: new [-rwxRWX...] <path> [<path> ...]
you may be looking for 'new --example'

in params (-rwxRWX...):
	lowercase = enable bit, uppercase = disable bit
	r/R = read, w/W = write, x/X = execute, o/O = overwrite
  repeat a letter up to 3 times for user/group/other
  a single -x fans out to all three groups
  path with trailing / creates a directory
  nested paths (a/b/c) create intermediate directories`)
	os.Exit(1)
}

func create(req Request) {
	name := req.name
	if name == "" {
		fmt.Println("empty filename")
		usage()
	}

	if strings.HasSuffix(name, "/") {
		req.is_dir = true
		name = strings.TrimSuffix(name, "/")
	}

	dir_perm_from := func(perms [3]Perms, has_perms bool) fs.FileMode {
		if !has_perms {
			return 0o755
		}
		forced := perms
		for i := range forced {
			forced[i].execute = On
		}
		return apply_perms(0, forced)
	}

	perm := apply_perms(0, req.perms)
	dir_perm := dir_perm_from(req.perms, req.has_perms)

	// ensure parent dihs exist
	dir := filepath.Dir(name)
	if dir != "." {
		if err := os.MkdirAll(dir, dir_perm); err != nil {
			fmt.Printf("error creating parent dirs for %q: %v\n", name, err)
			os.Exit(1)
		}
	}

	if req.is_dir {
		if err := os.MkdirAll(name, apply_perms(0, req.perms)); err != nil {
			fmt.Printf("error creating dir %q: %v\n", name, err)
			os.Exit(1)
		}
		fmt.Printf("created dir  %s (%04o)\n", name, dir_perm)
		return
	}

	info, err := os.Stat(name)
	if err == nil {
		// file exists
		if !req.has_perms {
			fmt.Printf("nothing to do for existing file %q (no flags given)\n", name)
			return
		}
		perm = apply_perms(info.Mode().Perm(), req.perms)
		if err := os.Chmod(name, perm); err != nil {
			fmt.Printf("error chmod %q: %v\n", name, err)
			os.Exit(1)
		}
		fmt.Printf("updated     %s (%04o)\n", name, perm)
		return
	}

	// new file
	f, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
	if err != nil {
		fmt.Printf("error creating %q: %v\n", name, err)
		os.Exit(1)
	}
	f.Close()
	fmt.Printf("created     %s (%04o)\n", name, perm)
}

func main() {
	args := os.Args[1:]
	if slices.Contains(args, "--example") {
		example()
		os.Exit(1)
	}

	if len(args) == 0 {
		usage()
	}

	var pending_req *Request
	had_flag := false

	for _, a := range args {
		if len(a) == 0 {
			continue
		}
		if a[0] == '-' {
			pending_req = &Request{}
			pending_req.parse_flag(a)
			had_flag = true
		} else {
			if pending_req == nil {
				pending_req = &Request{}
			}
			req := *pending_req
			req.name = a
			create(req)
			had_flag = false
		}
	}

	if had_flag {
		fmt.Println("flag given with no filename")
		usage()
	}
}
