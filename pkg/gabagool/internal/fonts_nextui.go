//go:build !nonextuifonts

package internal

import _ "embed"

//go:embed embedded_fonts/nextui/RoundedMplus1cNerdFont-Bold.ttf
var nextUIFont1 []byte

//go:embed embedded_fonts/nextui/BPreplayNerdFont-Bold.ttf
var nextUIFont2 []byte
