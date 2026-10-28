//go:build !windows

package state

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func tryLockRegistration(file *os.File) (bool, error) {
	err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
		return false, nil
	}
	return err == nil, err
}
