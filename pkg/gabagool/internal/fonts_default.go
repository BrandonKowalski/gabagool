//go:build !nodefaultfont

package internal

import _ "embed"

//go:embed embedded_fonts/HackGenConsoleNF-Bold.ttf
var defaultFont []byte
