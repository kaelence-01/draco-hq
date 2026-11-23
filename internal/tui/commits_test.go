package tui

import (
	"strings"
	"testing"

	"github.com/kaelence-01/draco-hq/internal/provider"
)

// suggestions are backend commits of other branches: two mention OPS-7, one
// of them by Ana, and one names OPS-70.
var suggestions = []provider.Commit{
	{ShortHash: "7d6c5b4", Subject: "OPS-7: totales del reporte", Author: "Ana", Email: "ana@example.test", Date: fakeDay.AddDate(0, 0, 1)},
	{ShortHash: "8a7b6c5", Subject: "ops-7 ajuste de redondeo", Author: "Luis", Email: "luis@example.test", Date: fakeDay.AddDate(0, 0, 1)},
	{ShortHash: "3e2d1c0", Subject: "OPS-70: otra tarea", Author: "Luis", Email: "luis@example.test", Date: fakeDay.AddDate(0, 0, 1)},
}

func TestCommitsSearchByMessageAuthorAndDate(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toCommits(t, "OPS-7")
	typeText(e.m, "/luis")
	out := view(e.m)
	if !strings.Contains(out, "5c4b3a2") || strings.Contains(out, "a1b2c3d") || !strings.Contains(out, "/ luis█") {
		t.Errorf("la búsqueda por autor deja solo sus commits, mientras se escribe:\n%s", out)
	}
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "filtro: luis") || !strings.Contains(out, "1 de 3") {
		t.Errorf("enter deja el filtro aplicado y la cuenta:\n%s", out)
	}
	send(e.m, keyEsc)
	if out := view(e.m); !strings.Contains(out, "a1b2c3d") || !strings.Contains(out, "Commits de OPS-7") {
		t.Errorf("esc quita el filtro sin salir:\n%s", out)
	}
	typeText(e.m, "/ana valida")
	if out := view(e.m); !strings.Contains(out, "9f8e7d6") || strings.Contains(out, "a1b2c3d") {
		t.Errorf("cada palabra tiene que estar:\n%s", out)
	}
	send(e.m, keyEsc)
	typeText(e.m, "/2026-09")
	if out := view(e.m); !strings.Contains(out, "Ningún commit coincide.") {
		t.Errorf("la fecha también se busca:\n%s", out)
	}
}

func TestCommitsPasteALineOfGitLogInTheSearch(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toCommits(t, "OPS-7")
	typeText(e.m, "/")
	send(e.m, paste("9f8e7d6 Valida rango máximo\n"))
	out := view(e.m)
	if !strings.Contains(out, "/ 9f8e7d6█") || strings.Contains(out, "a1b2c3d") || !strings.Contains(out, "ARCHIVOS (2)") {
		t.Errorf("de la línea pegada queda el hash, y se leen los archivos del commit que encuentra:\n%s", out)
	}
	if got := e.git.read["/code/pagos-api"]; strings.Join(got, ",") != "9f8e7d6" {
		t.Errorf("al pegar se leen los archivos del commit bajo el cursor: %v", got)
	}
}

func TestCommitsEnterOnAHashMarksOrAddsIt(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toCommits(t, "OPS-7")
	typeText(e.m, "/9f8e7d6")
	send(e.m, keyEnter)
	out := view(e.m)
	if !strings.Contains(out, "✓ 9f8e7d6 marcado") || !strings.Contains(out, "Backend (1)") || !strings.Contains(out, "a1b2c3d") {
		t.Errorf("un hash de la lista se marca y la lista vuelve completa:\n%s", out)
	}
	typeText(e.m, "/abcdef1")
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "abcdef1") || !strings.Contains(out, "(a mano)") || !strings.Contains(out, "Backend (2)") {
		t.Errorf("un hash que no está se agrega a mano:\n%s", out)
	}
	send(e.m, keySpace)
	if out := view(e.m); strings.Contains(out, "abcdef1") || !strings.Contains(out, "Backend (1)") {
		t.Errorf("espacio quita el hash a mano:\n%s", out)
	}
}

func TestCommitsMineNeedsTheIdentity(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toCommits(t, "OPS-7")
	send(e.m, press('m', "m"))
	if out := view(e.m); !strings.Contains(out, "Sin user.email en el repo") || !strings.Contains(out, "5c4b3a2") {
		t.Errorf("sin identidad, m lo explica y no filtra:\n%s", out)
	}

	e = newGenerateApp(t, nil)
	e.git.email = "ANA@example.test"
	e.toCommits(t, "OPS-7")
	send(e.m, press('m', "m"))
	out := view(e.m)
	if !strings.Contains(out, "a1b2c3d") || !strings.Contains(out, "9f8e7d6") || strings.Contains(out, "5c4b3a2") || !strings.Contains(out, "m: solo míos") || !strings.Contains(out, "2 de 3") {
		t.Errorf("m deja los commits del user.email del clon:\n%s", out)
	}
	send(e.m, press('m', "m"))
	if out := view(e.m); !strings.Contains(out, "5c4b3a2") {
		t.Errorf("m otra vez los muestra todos:\n%s", out)
	}
}

func TestCommitsSuggestWhatMentionsTheIssue(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.git.email, e.git.others = "ana@example.test", suggestions
	e.toCommits(t, "OPS-7")
	out := view(e.m)
	if !strings.Contains(out, "[x]★7d6c5b4") || !strings.Contains(out, "[ ]★8a7b6c5") {
		t.Errorf("los que mencionan la tarea van con estrella, y solo los propios marcados:\n%s", out)
	}
	if strings.Index(out, "8a7b6c5") > strings.Index(out, "a1b2c3d") || strings.Contains(out, "3e2d1c0") {
		t.Errorf("los sugeridos van arriba y OPS-70 no es OPS-7:\n%s", out)
	}
	if !strings.Contains(out, "★ menciona OPS-7") || !strings.Contains(out, "Backend (1)") {
		t.Errorf("el detalle dice por qué se sugiere:\n%s", out)
	}
	send(e.m, keyEnter)
	if got := e.git.read["/code/pagos-api"]; strings.Join(got, ",") != "7d6c5b4" {
		t.Errorf("el sugerido marcado va al documento: %v", got)
	}
}

func TestCommitsFromTheBarMarkOnlyItsHashes(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.git.email, e.git.others = "ana@example.test", suggestions
	e.resize(160, 40)
	runBar(e.m, "g -i OPS-7 -b a1b2c3d")
	out := view(e.m)
	if !strings.Contains(out, "Backend (1)") || !strings.Contains(out, "✓ a1b2c3d") || !strings.Contains(out, "7d6c5b4") {
		t.Errorf("con -b no se marca nada más:\n%s", out)
	}
}

func TestCommitsReadMoreAtTheEnd(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.git.older = 150
	e.toCommits(t, "OPS-7")
	if out := view(e.m); !strings.Contains(out, "100 leídos") {
		t.Fatalf("se lee una página de 100:\n%s", out)
	}
	for range 100 {
		send(e.m, keyDown)
	}
	if out := view(e.m); !strings.Contains(out, "153 leídos") || !strings.Contains(out, "Commit viejo 97") {
		t.Errorf("↓ en la última fila lee la página siguiente:\n%s", out)
	}
	for range 60 {
		send(e.m, keyDown)
	}
	if out := view(e.m); !strings.Contains(out, "153 leídos") || !strings.Contains(out, "Commit viejo 150") {
		t.Errorf("sin más commits, ↓ se queda en el último:\n%s", out)
	}

	e = newGenerateApp(t, nil)
	e.git.older = 150
	e.toCommits(t, "OPS-7")
	typeText(e.m, "/viejo 150")
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "Ningún commit coincide. ↓ lee 100 más.") {
		t.Fatalf("sin coincidencias en lo leído, ofrece leer más:\n%s", out)
	}
	send(e.m, keyDown)
	if out := view(e.m); !strings.Contains(out, "Commit viejo 150") || !strings.Contains(out, "1 de 153") {
		t.Errorf("la búsqueda sigue en lo que se lee:\n%s", out)
	}
}

func TestCommitsOfEveryBranch(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.git.others = []provider.Commit{{ShortHash: "1f2e3d4", Subject: "Ajuste en otra rama", Author: "Luis", Date: fakeDay.AddDate(0, 0, 2)}}
	e.toCommits(t, "OPS-7")
	send(e.m, keySpace)
	if out := view(e.m); strings.Contains(out, "1f2e3d4") {
		t.Fatalf("la rama actual no trae commits de otras:\n%s", out)
	}
	send(e.m, press('a', "a"))
	out := view(e.m)
	if !strings.Contains(out, "1f2e3d4") || !strings.Contains(out, "pagos-api · todas las ramas") || !strings.Contains(out, "a: todas las ramas") || !strings.Contains(out, "Backend (1)") {
		t.Errorf("a lee todas las ramas y lo marcado se queda:\n%s", out)
	}
	send(e.m, press('a', "a"))
	if out := view(e.m); strings.Contains(out, "1f2e3d4") || !strings.Contains(out, "pagos-api · develop") {
		t.Errorf("a otra vez vuelve a la rama:\n%s", out)
	}
}
