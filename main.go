package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/emicklei/dot"
)

// example: go run main.go example | dot -Tpng > test.png && open test.png
func main() {

	// fall back to current directory if no directory given
	var dir string
	if len(os.Args) == 1 {
		dir, _ = os.Executable()
		fmt.Fprintf(os.Stderr, "warning: using current directory: '%s'\n", dir)
	} else {
		if os.Args[1] != "-h" {
			dir = os.Args[1]
			os.Args = os.Args[1:]
		}
	}

	cluster := flag.Bool("cluster", false, "cluster nodes in sub directories")
	diary := flag.Bool("diary", false, "collapse all diary entries under a single `diary.wiki` node")
	flag.Parse()

	// remap any path that contains `diary` into `diary.wiki`
	remap := make(map[string]string)
	if !*diary {
		remap["diary"] = "diary.wiki"
	}

	// setup vimwiki struct
	wiki := newWiki(dir, remap, *cluster)

	// any trailing arguments are considered directories to skip
	subDirToSkip := []string{".git"}
	for _, dir := range flag.Args() {
		subDirToSkip = append(subDirToSkip, dir)
	}

	// walk directories and build graph
	if err := wiki.Walk(subDirToSkip); err != nil {
		log.Fatalf("Error when walking directories: %v", err)
	}

	// convert to a dot-graph for visualisation
	g := wiki.Dot(dot.Directed)
	g.Attr("rankdir", "LR")
	g.Write(os.Stdout)
}
