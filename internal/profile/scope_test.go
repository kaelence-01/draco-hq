package profile

import (
	"testing"

	"github.com/kaelence-01/draco-hq/internal/config"
)

func TestScope(t *testing.T) {
	confluence := &config.Profile{Docs: config.ProviderConfig{Provider: "confluence", Space: "DEV"}}
	markdown := &config.Profile{Docs: config.ProviderConfig{Provider: "markdown", Path: "/docs/despliegues"}}
	cases := []struct {
		name string
		p    *config.Profile
		o    Options
		want string
	}{
		{"el space del perfil", confluence, Options{}, "DEV"},
		{"el space del proyecto antes que el del perfil", confluence, Options{ProjectSpace: "PAG"}, "PAG"},
		{"el scope pedido antes que todos", confluence, Options{Scope: "OPS", ProjectSpace: "PAG"}, "OPS"},
		{"sin space en ningún lado", &config.Profile{Docs: config.ProviderConfig{Provider: "confluence"}}, Options{}, ""},
		{"la carpeta del perfil Markdown", markdown, Options{}, "/docs/despliegues"},
		{"Markdown ignora el space del proyecto", markdown, Options{ProjectSpace: "PAG"}, "/docs/despliegues"},
		{"Markdown con otra carpeta pedida", markdown, Options{Scope: "/otra"}, "/otra"},
		{"un destino desconocido", &config.Profile{Docs: config.ProviderConfig{Provider: "otro"}}, Options{ProjectSpace: "PAG"}, ""},
	}
	for _, c := range cases {
		if got := Scope(c.p, c.o); got != c.want {
			t.Errorf("%s: se esperaba %q y se obtuvo %q", c.name, c.want, got)
		}
	}
}
