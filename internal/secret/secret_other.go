//go:build !linux && !darwin

package secret

func disableEcho(fd uintptr) (func(), error) {
	return nil, errNotSupported
}
