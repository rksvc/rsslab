package utils

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	regPath    = `Software\Microsoft\Windows\CurrentVersion\Run`
	regKeyName = "rsslab"
)

func AutoStart() (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, regPath, registry.QUERY_VALUE)
	if err != nil {
		return false, err
	}
	defer k.Close()
	_, _, err = k.GetStringValue(regKeyName)
	if err == nil {
		return true, nil
	} else if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func SetAutoStart(enable bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, regPath, registry.WRITE)
	if err != nil {
		return nil
	}
	defer k.Close()
	if enable {
		var b strings.Builder
		b.WriteByte('"')
		path, err := filepath.Abs(os.Args[0])
		if err != nil {
			return err
		}
		b.WriteString(path)
		b.WriteByte('"')
		for _, arg := range os.Args[1:] {
			b.WriteByte(' ')
			if strings.ContainsRune(arg, ' ') {
				b.WriteByte('"')
				for _, c := range arg {
					if c == '"' {
						b.WriteString(`\"`)
					} else {
						b.WriteRune(c)
					}
				}
				b.WriteByte('"')
			} else {
				b.WriteString(arg)
			}
		}
		return k.SetStringValue(regKeyName, b.String())
	} else {
		return k.DeleteValue(regKeyName)
	}
}
