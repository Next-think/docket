package dex

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	. "polydawn.net/pogo/gosh"
	. "polydawn.net/docket/crocker"
	"strings"
)

type Graph struct {
	/*
		Absolute path to the base/working dir for of the graph git repository.
	*/
	dir string

	/*
		Cached command template for exec'ing git with this graph's cwd.
	*/
	cmd Command
}

func NewGraph(dir string) *Graph {
	dir, err := filepath.Abs(dir)
	if err != nil { panic(err); }

	return &Graph{
		dir: dir,
		cmd: Sh("git")(DefaultIO)(Opts{Cwd: dir}),
	}
}

/*

*/
func (g *Graph) GetDir() string {
	return g.dir
}

/*
Wipes uncommitted changes in the git working tree.
*/
func (g *Graph) Cleanse() {
	g.cmd("reset")()
	//Temporarily don't care if 'git checkout .' fails.
	g.cmd("checkout", ".")(Opts{OkExit:[]int{0, 1}})()
	g.cmd("clean", "-xf")()
}

// Prepares for a publish by creating a branch
func (g *Graph) PreparePublish(lineage string, ancestor string) {
	//Handle tags - currently, we discard them when dealing with a graph repo.
	lineage, _  = SplitImageName(lineage)
	ancestor, _ = SplitImageName(ancestor)

	if strings.Count(g.cmd("branch", "--list", lineage).Output(), "\n") < 1 {
		//Memo("this is a new lineage!")
		g.cmd("checkout", "-b", lineage)()
		g.cmd("rm", "*")
	} else {
		g.cmd("checkout", lineage)()
	}
}

/*
Commits a new image.  The "lineage" branch name will be extended by this new commit (or
created, if it doesn't exist), and the "ancestor" branch will also be credited as a parent
of the new commit.
*/
func (g *Graph) Publish(lineage string, ancestor string) {
	//Handle tags - currently, we discard them when dealing with a graph repo.
	lineage, _  = SplitImageName(lineage)
	ancestor, _ = SplitImageName(ancestor)

	g.cmd("add", "--all")()
	g.forceMerge(ancestor, lineage)
	g.cmd("show")(Opts{OkExit:[]int{0, 141}})()
}

func (g *Graph) forceMerge(source string, target string) {
	writeTree := g.cmd("write-tree").Output()
	writeTree = strings.Trim(writeTree, "\n")
	commitMsg := fmt.Sprintf("updated %s<<%s", target, source)
	mergeTree := g.cmd("commit-tree", writeTree, "-p", source, "-p", target, Opts{In: commitMsg}).Output()
	mergeTree = strings.Trim(mergeTree, "\n")
	g.cmd("merge", mergeTree)()
}

/*
Returns a read stream for the requested image.  Internally, the commit that the "lineage" branch ref
currently points to is opened and "image.tar" is read from.
*/
func (g *Graph) Load(lineage string) io.Reader {
	//FIXME: entirely possible to do this without doing a `git checkout`... do so
	lineage, _ = SplitImageName(lineage) //Handle tags
	g.cmd("checkout", lineage)()

	in, err := os.OpenFile(g.dir+"/image.tar", os.O_RDONLY, 0644)
	if err != nil { panic(err); }
	return in
}
