package web

import (
	"embed"
	t "html/template"
	"io"
	"net/http"

	"github.com/gofiber/template"
	"github.com/gofiber/template/html/v2"
)

//go:embed graphicarts javascripts stylesheets
var Assets embed.FS

//go:embed *.html
var Templates embed.FS

var Engine = html.Engine{
	Engine: template.Engine{
		Left:       "{%",
		Right:      "%}",
		Directory:  "/",
		Extension:  ".html",
		LayoutName: "embed",
		FileSystem: http.FS(Templates),
		Funcmap: map[string]any{
			"inline": func(svg string) t.HTML {
				svgFile, err := Assets.Open("graphicarts/" + svg)
				if err != nil {
					panic(err)
				}
				content, err := io.ReadAll(svgFile)
				if err != nil {
					panic(err)
				}
				return t.HTML(content)
			},
		},
	},
}
