//go:build !windows

package utils

import _ "embed"

//go:embed icon.png
var Icon []byte
