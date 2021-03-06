package dirtree

import (
	sh "github.com/jeffwilliams/spacehoarder"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Op is a type of operation on a Dirtree.
type Op int

const (
	Push Op = iota
	Pop
	AddSize
)

// OpData is an operation on a DirTree and it's corresponding data.
type OpData struct {
	Op           Op
	Path         string
	Basename     string
	Size         int64
	SizeAccurate bool
	Type         PathType
}

// Filesystem is an abstraction of a filesystem used by BuildFs.
type Filesystem interface {
	// Open opens a file with the specified path. If an error occurs opening the file
	// then err is non-nil on return.
	Open(path string) (file File, err error)

	// DeviceId returns an identifier for the device/mount that the path resides on
	DeviceId(path string) (id uint64, err error)
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

func (r OsFilesystem) DeviceId(path string) (id uint64, err error) {
	return sh.GetFsDevId(path)
}

type BuildOpts struct {
	// If the walk would cross into another filesystem, do not traverse it.
	OneFs bool
	// Include files in the output.
	IncludeFiles bool
}

var DefaultBuildOpts = &BuildOpts{
	OneFs:        true,
	IncludeFiles: false,
}

// Build builds a new Dirtree starting from the specified directory `basepath` and writes all
// the operations performed to the Dirtree to the ops channel so that a copy of the Dirtree can be
// made in a different goroutine. The paths processed are written to the channel prog.
func Build(basepath string, opts *BuildOpts) (ops chan OpData, prog chan string) {
	return BuildFs(OsFilesystem{}, basepath, opts)
}

// BuildFs builds a new Dirtree starting from the specified directory `basepath` and writes all
// the operations performed to the Dirtree to the ops channel so that a copy of the Dirtree can be
// made in a different goroutine. The paths processed are written to the channel prog.
// The Filesystem fs is used for opening files.
func BuildFs(fs Filesystem, basepath string, opts *BuildOpts) (ops chan OpData, prog chan string) {

	ops = make(chan OpData)
	prog = make(chan string)

	go build(fs, basepath, ops, prog, opts)

	return
}

// BuildSync builds a new Dirtree starting from the specified directory `basepath` and returns it when
// it's complete.
func BuildSync(basepath string, opts *BuildOpts) *Dirtree {
	ops := make(chan OpData)
	go build(OsFilesystem{}, basepath, ops, nil, opts)
	tree := New()
	tree.ApplyAll(ops)
	return tree
}

func build(fs Filesystem, basepath string, ops chan OpData, prog chan string, opts *BuildOpts) {

	if ops != nil {
		defer close(ops)
	}

	if prog != nil {
		defer close(prog)
	}

	if ops != nil {
		ops <- OpData{Op: Push, Path: basepath, Basename: filepath.Base(basepath), SizeAccurate: true}
	}

	// Directories to process
	work := make([]string, 0, 1000)

	baseDevId, err := fs.DeviceId(basepath)
	if opts.OneFs && err != nil {
		return
	}

	work = append(work, basepath)

	ticker := time.NewTicker(300 * time.Millisecond)

	procDir := func(path string) {
		accurate := true

		dir, err := fs.Open(path)
		if err != nil {
			//fmt.Println("Error opening directory", path, ":", err)
			ops <- OpData{Op: AddSize, Size: 0, SizeAccurate: false}
			return
		}

		fis, err := dir.Readdir(-1)
		if err != nil {
			//fmt.Println("Error processing directory", path)
			accurate = false
		}

		size := int64(0)
		for _, fi := range fis {
			fpath := path + string(os.PathSeparator) + fi.Name()

			if fi.Mode().IsRegular() {
				if opts.IncludeFiles {
					ops <- OpData{Op: Push, Size: fi.Size(), Path: fpath, Basename: filepath.Base(fpath), SizeAccurate: true, Type: PathTypeFile}
					//ops <- OpData{Op: AddSize, Size: fi.Size(), SizeAccurate: true}
				} else {
					size += fi.Size()
				}
			} else if fi.IsDir() {

				if opts.OneFs {
					devId, err := fs.DeviceId(fpath)

					if err == nil && baseDevId != devId {
						continue
					}
				}

				ops <- OpData{Op: Push, Path: fpath, Basename: filepath.Base(fpath), SizeAccurate: true, Type: PathTypeDir}
				work = append(work, fpath)
			}

			// Send a progress update if this is taking a long time
			select {
			case <-ticker.C:
				if prog != nil {
					prog <- fpath
				}
			default:
			}
		}

		dir.Close()

		ops <- OpData{Op: AddSize, Size: size, SizeAccurate: accurate}
	}

	for len(work) > 0 {
		// Refactor below; use the same code as in Apply.
		path := work[len(work)-1]
		work = work[0 : len(work)-1]

		ops <- OpData{Op: Pop}

		procDir(path)

		if prog != nil {
			prog <- path
		}
	}

	ticker.Stop()
}

// Same as Apply, but a copy of each OpData in ops is written to outops
func ApplyAndDup(t *Dirtree, ops chan OpData, outops chan OpData) {
	applyOps := make(chan OpData)
	go t.ApplyAll(applyOps)

	defer close(applyOps)
	defer close(outops)

	for op := range ops {
		applyOps <- op
		outops <- op
	}
}
