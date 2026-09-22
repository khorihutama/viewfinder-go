//go:build windows

package native

import (
	"syscall"
)

var dll = syscall.NewLazyDLL("viewfinder_native.dll")
