//go:build darwin || linux || freebsd || netbsd || openbsd

package baseline

import "syscall"

func backgroundProcessAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
