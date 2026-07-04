package ui

import (
	"embed"

	"github.com/a-h/templ"
)

//go:embed  static
var StaticFS embed.FS

var PostEmbeds = map[string]templ.Component{
	"game-of-life": GameOfLifeEmbed(),
}
