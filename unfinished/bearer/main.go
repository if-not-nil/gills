package main

import (
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/fatih/color"
	"github.com/go-git/go-git/v6"
	"github.com/mattn/go-isatty"
)

func main() {
	var flag_color = flag.Bool("color", true, "color output?")
	flag.Parse()

	color.NoColor = !isatty.IsTerminal(os.Stdout.Fd()) || !*flag_color
	home_dir, _ := os.UserHomeDir()
	root_path := path.Join(home_dir, "bearer")

	root, err := os.ReadDir(root_path)

	if err != nil {
		fmt.Println("cant open root dir")
		os.Exit(1)
	}

	projects := map[string][]os.DirEntry{}

	for _, dir := range root {
		dir_path := path.Join(root_path, dir.Name())
		dir_path = path.Clean(dir_path)

		_, err := git.PlainOpen(dir_path)
		if err != nil {
			color.New(color.FgHiBlack).Printf("%s is not a git repo, skipping...\n", dir_path)
			continue
		}
		fmt.Printf("%s!!!\n", dir_path)
		// log.Println(repo.Log(&git.LogOptions{All: true}))
	}
	fmt.Println(projects)
}
