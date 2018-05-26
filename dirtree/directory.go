package dirtree

type PathType uint8

const (
	PathTypeDir PathType = iota
	PathTypeFile
)

type PathInfo struct {
	Path         string
	Basename     string
	Size         int64
	SizeAccurate bool
	Type         PathType
}
