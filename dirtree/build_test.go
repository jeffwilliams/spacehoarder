package dirtree

import (
	"errors"
	"os"
	"testing"
	"time"
)

type TestFileInfo struct {
	name string
	size int64
	mode os.FileMode
}

func (t TestFileInfo) Name() string {
	return t.name
}

func (t TestFileInfo) Size() int64 {
	return t.size
}

func (t TestFileInfo) Mode() os.FileMode {
	return t.mode
}

func (t TestFileInfo) ModTime() time.Time {
	return time.Now()
}

func (t TestFileInfo) IsDir() bool {
	return t.mode.IsDir()
}

func (t TestFileInfo) Sys() interface{} {
	return nil
}

func NewTestFileInfo(name string, dir bool, size int64) TestFileInfo {
	mode := os.FileMode(0)
	if dir {
		mode |= os.ModeDir
	}

	return TestFileInfo{
		name: name,
		size: size,
		mode: mode,
	}
}

type TestFile []os.FileInfo

func (t TestFile) Close() error {
	return nil
}

func (t TestFile) Readdir(count int) ([]os.FileInfo, error) {
	if count > 0 {
		panic("These test cases assume the code calls Readdir with count <= 0")
	}
	return []os.FileInfo(t), nil
}

type TestFs struct {
	// Map a path to the file.
	Files map[string]TestFile
}

func (t TestFs) Open(name string) (file File, err error) {
	f, ok := t.Files[name]
	if !ok {
		return nil, errors.New("No such file " + name)
	}

	return f, nil
}

func makeTestFs() TestFs {
	/*
	  "/tmp"
	  "/tmp/a"
	  "/tmp/a/file1.txt  20"
	  "/tmp/a/file2.txt  10"
	  "/tmp/b"
	  "/tmp/b/a.txt       5"
	  "/tmp/b/dir"
	  "/tmp/b/dir/blort  30"
	*/

	// /tmp
	tmp := TestFile(make([]os.FileInfo, 0))
	tmp = append(tmp, NewTestFileInfo("a", true, 0))
	tmp = append(tmp, NewTestFileInfo("b", true, 0))

	// /tmp/a
	a := TestFile(make([]os.FileInfo, 0))
	a = append(a, NewTestFileInfo("file1.txt", false, 20))
	a = append(a, NewTestFileInfo("file2.txt", false, 10))

	// /tmp/b
	b := TestFile(make([]os.FileInfo, 0))
	b = append(b, NewTestFileInfo("a.txt", false, 5))
	b = append(b, NewTestFileInfo("dir", true, 0))

	// /tmp/b/dir
	dir := TestFile(make([]os.FileInfo, 0))
	dir = append(dir, NewTestFileInfo("blort", false, 30))

	fs := TestFs{
		Files: map[string]TestFile{
			"/tmp":             tmp,
			"/tmp/a":           a,
			"/tmp/a/file1.txt": TestFile(make([]os.FileInfo, 0)),
			"/tmp/a/file2.txt": TestFile(make([]os.FileInfo, 0)),
			"/tmp/b":           b,
			"/tmp/b/a.txt":     TestFile(make([]os.FileInfo, 0)),
			"/tmp/b/dir":       dir,
			"/tmp/b/dir/blort": TestFile(make([]os.FileInfo, 0)),
		},
	}

	return fs
}

func TestBuild(t *testing.T) {
	fs := makeTestFs()

	//tree := buildFs(fs, "/tmp", nil, nil)
	ops := make(chan OpData)
	go build(fs, "/tmp", ops, nil)

	tree := New()
	Apply(tree, ops)

	expected := map[string]int64{
		"tmp": 65,
		"a":   30,
		"b":   35,
		"dir": 30,
	}

	detected := map[string]bool{}

	tree.Root.Walk(func(n *Node) {
		//t.Logf("%s: %v\n", n.Dir.Path, n.Dir.Size)
		size, ok := expected[n.Dir.Basename]
		if !ok {
			t.Fatal("Directory with name", n.Dir.Basename, "wasn't expected")
		}
		if n.Dir.Size != size {
			t.Fatal("Directory with name", n.Dir.Basename, "should have size", size, "but has size", n.Dir.Size)
		}
		detected[n.Dir.Basename] = true
	})

	for k, _ := range expected {
		if _, ok := detected[k]; !ok {
			t.Fatal("Directory with name", k, "was not detected in the tree")
		}
	}

}
