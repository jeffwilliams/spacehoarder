package dirtree

import (
	"fmt"
	"io"
	"os"
	"path"
	"time"
)

// Op is a type of operation on a Dirtree.
type Op int

const (
	// Root sets the root of the tree.
	Root Op = iota
	// Add adds a node to a Dirtree.
	Add
	// Del deletes a node from a Dirtree.
	Del
	// Update modifies the contents of a node in a Dirtree.
	Update
)

// OpData is an operation on a DirTree and it's corresponding data.
type OpData struct {
	Op   Op
	Node *Node
	Size int64
}

// Filesystem is an abstraction of a filesystem used by BuildFs.
type Filesystem interface {
	// Open opens a file with the specified path. If an error occurs opening the file
	// then err is non-nil on return.
	Open(path string) (file File, err error)
}

// File is an abstraction of a file.
type File interface {
	io.Closer
	Readdir(count int) ([]os.FileInfo, error)
}

// OsFilesystem is a Filesystem that performs as expected; that is,
// it opens files from the local filesystem.
type OsFilesystem struct{}

// Open opens the file with the specified path.
func (r OsFilesystem) Open(path string) (file File, err error) {
	return os.Open(path)
}

// Build builds a new Dirtree starting from the specified directory `basepath` and writes all
// the operations performed to the Dirtree to the ops channel so that a copy of the Dirtree can be
// made in a different goroutine. The paths processed are written to the channel prog.
func Build(basepath string) (ops chan OpData, prog chan string) {
	return BuildFs(OsFilesystem{}, basepath)
}

// BuildFs builds a new Dirtree starting from the specified directory `basepath` and writes all
// the operations performed to the Dirtree to the ops channel so that a copy of the Dirtree can be
// made in a different goroutine. The paths processed are written to the channel prog.
// The Filesystem fs is used for opening files.
func BuildFs(fs Filesystem, basepath string) (ops chan OpData, prog chan string) {

	ops = make(chan OpData)
	prog = make(chan string)

	go buildFs(fs, basepath, ops, prog)

	return
}

// BuildSync builds a new Dirtree starting from the specified directory `basepath` and returns it when
// it's complete.
func BuildSync(basepath string) *Dirtree {
	return buildFs(OsFilesystem{}, basepath, nil, nil)
}

func buildFs(fs Filesystem, basepath string, ops chan OpData, prog chan string) *Dirtree {

	if ops != nil {
		defer close(ops)
	}

	if prog != nil {
		defer close(prog)
	}

	tree := New()

	tree.Root().Dir.Path = basepath
	tree.Root().Dir.Basename = path.Base(basepath)

	if ops != nil {
		ops <- OpData{Op: Root, Node: tree.Root(), Size: 0}
	}

	// Directories to process
	work := make([]*Node, 0, 1000)

	work = append(work, tree.Root())

	updateSize := func(node *Node, size int64) {
		node.UpdateSize(size)
		if ops != nil {
			ops <- OpData{Op: Update, Node: node, Size: size}
		}
	}

	ticker := time.NewTicker(300 * time.Millisecond)

	procDir := func(node *Node) {
		dir, err := fs.Open(node.Dir.Path)
		if err != nil {
			fmt.Println("Error opening directory", node.Dir.Path, ":", err)
			return
		}

		fis, err := dir.Readdir(-1)
		if err != nil {
			fmt.Println("Error processing directory", node.Dir.Path)
		}

		size := node.Dir.Size
		for _, fi := range fis {
			path := node.Dir.Path + string(os.PathSeparator) + fi.Name()

			if fi.Mode().IsRegular() {
				size += fi.Size()
			} else if fi.IsDir() {
				ch := &Node{
					Dir: Directory{
						Path:     path,
						Basename: fi.Name(),
						Size:     0,
					},
				}
				node.Add(ch)
				if ops != nil {
					ops <- OpData{Op: Add, Node: ch, Size: 0}
				}
				work = append(work, ch)
			}

			// Send a progress update if this is taking a long time
			select {
			case <-ticker.C:
				prog <- path
			default:
			}
		}

		dir.Close()

		updateSize(node, size)
	}

	for len(work) > 0 {
		node := work[len(work)-1]
		work = work[0 : len(work)-1]

		procDir(node)

		if prog != nil {
			prog <- node.Dir.Path
		}
	}

	ticker.Stop()

	return tree
}

// Apply applies the operation `op` to the Dirtree `t`.
// This is meant to be used to apply the operations output by
// Build and BuildFs for creating a duplicate tree.
func Apply(t *Dirtree, op OpData) {
	switch op.Op {
	case Root:
		t.SetRootCopy(op.Node, op.Size)
	case Add:
		t.AddCopy(op.Node, op.Size)
	case Del:
		t.DelCopy(op.Node)
	case Update:
		t.UpdateCopy(op.Node, op.Size)
	}
}
