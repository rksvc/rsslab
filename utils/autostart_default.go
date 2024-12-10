//go:build !windows

package utils

func AutoStart() (bool, error) {
	return false, ErrAutoStartNotImplemented
}

func SetAutoStart(enable bool) error {
	return ErrAutoStartNotImplemented
}
