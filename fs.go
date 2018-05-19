package spacehoarder

import (
	"fmt"
	"os"
	"syscall"
)

func GetFsDevId(path string) (uint64, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return 0, err
	}

	stat, ok := fi.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return 0, fmt.Errorf("Unable to determine filesystem device because underlying implementation does not support it")
	}

	return stat.Dev, nil
}
