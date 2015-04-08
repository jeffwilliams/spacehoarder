package dirtree

import (
  "path"
  "os"
  "io"
  "fmt"
)

type Op int

const (
  Root Op = iota
  Add
  Del
  Update
)

type OpData struct {
  Op Op
  Node *Node
}

type Filesystem interface {
  Open(name string) (file File, err error)
}

type File interface {
  io.Closer
  Readdir(count int) ([]os.FileInfo, error)
}

type OsFilesystem struct{}

func (r OsFilesystem) Open(name string) (file File, err error) {
  return os.Open(name)
}

// Build a new Dirtree, and pass along all the operations made along the way.
func Build(basepath string) (ops chan OpData, prog chan string) {
  return BuildFs(OsFilesystem{}, basepath)
}

func BuildFs(fs Filesystem, basepath string) (ops chan OpData, prog chan string) {

  ops = make(chan OpData)
  prog = make(chan string)

  go buildFs(fs, basepath, ops, prog)

  return
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
    ops <- OpData{Op: Root, Node: tree.Root()}
  }

  // Directories to process
  work := make([]*Node, 0, 1000)

  work = append(work, tree.Root())

  procDir := func(node *Node) {
    //dir, err := os.Open(node.Dir.Path)
    dir, err := fs.Open(node.Dir.Path)
    if err != nil {
      fmt.Println("Error opening directory", node.Dir.Path)
      return
    }

    fis, err := dir.Readdir(-1)
    if err != nil {
      fmt.Println("Error processing directory", node.Dir.Path)
    }

    size := node.Dir.Size
    for _, fi := range fis {
      if fi.Mode().IsRegular() {
        size += fi.Size()
      } else if fi.IsDir() {
        ch := &Node{
          Dir: Directory{
            Path: node.Dir.Path + string(os.PathSeparator) + fi.Name(),
            Basename: fi.Name(),
            Size: 0,
          },
        }
        node.Add(ch)
        work = append(work, ch)
      }
    }

    dir.Close()

    node.UpdateSize(size)
    if ops != nil {
fmt.Println("Sending add for ", node.Dir.Path)
      ops <- OpData{Op: Add, Node: node}
    }
  }

  for len(work) > 0 {

    node := work[len(work)-1]
    work = work[0:len(work)-1]

    procDir(node)

    if prog != nil {
      prog <- node.Dir.Path
    }
  }

  return tree
}


func Apply(t *Dirtree, op OpData) {
  switch op.Op {
  case Root:
    t.SetRootCopy(op.Node)
  case Add:
fmt.Println("Applying add for ", op.Node.Dir.Path)
    t.AddCopy(op.Node)
  case Del:
    t.DelCopy(op.Node)
  case Update:
    t.UpdateCopy(op.Node)
  }
}
