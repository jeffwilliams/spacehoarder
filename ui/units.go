package ui

import "fmt"

func FancySize(size int64) string {
	units := []string{"", "K", "M", "G", "T", "P", "E", "Z"}

	f := float64(size)

	i := 0
	for ; i < len(units); i++ {
		if f < 1024.0 {
			break
		}
		f /= 1024.0
	}

	return fmt.Sprintf("%.1f%sB", f, units[i])
}
