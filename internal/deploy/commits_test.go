package deploy

import (
	"testing"
	"time"

	"github.com/kaelence-01/draco-hq/internal/provider"
)

func TestCommitMatches(t *testing.T) {
	c := provider.Commit{Hash: "a1b2c3d4e5f6", ShortHash: "a1b2c3d", Subject: "Agrega filtro de Pagos", Author: "Ana Pérez", Email: "ana@example.test",
		Date: time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC)}
	for query, want := range map[string]bool{
		"":                       true,
		"a1b2":                   true,
		"A1B2C3D4E5":             true,
		"b2c3":                   false,
		"pagos":                  true,
		"ana 2026-11 filtro":     true,
		"ana 2026-10":            false,
		"example.test":           true,
		"pérez  pagos":           true,
		"luis":                   false,
		"filtro a1b2c3d 2026-11": true,
	} {
		if got := CommitMatches(c, query); got != want {
			t.Errorf("CommitMatches(%q) = %v, se esperaba %v", query, got, want)
		}
	}
}

func TestMine(t *testing.T) {
	c := provider.Commit{Email: "Ana@Example.test"}
	if !Mine(c, "ana@example.test") || Mine(c, "luis@example.test") || Mine(provider.Commit{}, "") {
		t.Error("Mine compara el correo sin distinguir mayúsculas, y un correo vacío no es de nadie")
	}
}

func TestMentionsIssue(t *testing.T) {
	for text, want := range map[string]bool{
		"OPS-7: totales del reporte":  true,
		"[ops-7] totales":             true,
		"Corrige OPS-7.":              true,
		"Merge feature/OPS-7-reporte": true,
		"OPS-70: otra tarea":          false,
		"XOPS-7 no es la tarea":       false,
		"OPS-70 y luego OPS-7":        true,
		"Sin tarea":                   false,
	} {
		if got := MentionsIssue(text, "OPS-7"); got != want {
			t.Errorf("MentionsIssue(%q) = %v, se esperaba %v", text, got, want)
		}
	}
	if MentionsIssue("OPS-7", "") {
		t.Error("sin clave no hay mención")
	}
}

func TestLooksLikeHash(t *testing.T) {
	for s, want := range map[string]bool{
		"a1b2c3d": true, "A1B2C3D4E5F6": true, "a1b2c3": false, "pagos-api": false, "g1b2c3d": false,
		"a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2": true, "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c": false,
	} {
		if got := LooksLikeHash(s); got != want {
			t.Errorf("LooksLikeHash(%q) = %v, se esperaba %v", s, got, want)
		}
	}
}
