package dirtree

type Directory struct {
	Path         string
	Basename     string
	Size         int64
	SizeAccurate bool
}
