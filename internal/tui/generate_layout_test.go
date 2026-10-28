package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func (e *generateEnv) resize(width, height int) {
	e.m.Update(tea.WindowSizeMsg{Width: width, Height: height})
}

// bottomEdge is the body line just above the rule of the footer.
func bottomEdge(lines []string) string { return lines[len(lines)-3] }

func TestGenerateFormShowsWhatFollows(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.resize(160, 40)
	lines := strings.Split(view(e.m), "\n")
	assertFits(t, lines, 160, 40)
	out := strings.Join(lines, "\n")
	for _, want := range []string{
		"TAREA Y PROYECTO", "LO QUE SIGUE", "Tarea  ›  Commits  ›  Vista previa  ›  Ubicación  ›  Publicar",
		"Backend  pagos-api  /code/pagos-api", "Frontend pagos-web  /code/pagos-web", "Destino  markdown · despliegues",
		"La tarea se lee de jira y los commits,",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	form, next := lineWith(t, lines, "TAREA Y PROYECTO"), lineWith(t, lines, "LO QUE SIGUE")
	if strings.Index(next, "LO QUE SIGUE") < strings.Index(form, "TAREA Y PROYECTO")+40 {
		t.Errorf("lo que sigue va a la derecha del formulario:\n%s", out)
	}

	e.m.top().(*generate).project = "facturacion"
	out = view(e.m)
	if !strings.Contains(out, "Backend  facturacion-api  /code/facturacion-api") || !strings.Contains(out, "Frontend —") {
		t.Errorf("la caja sigue al proyecto elegido; sin frontend, un guion:\n%s", out)
	}

	e.resize(80, 24)
	if out := view(e.m); strings.Contains(out, "LO QUE SIGUE") || !strings.Contains(out, "Tarea") {
		t.Errorf("en 80 columnas solo va el formulario:\n%s", out)
	}
}

func TestCommitsShowTheFilesOfTheCommitBesideTheList(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.resize(160, 40)
	e.toCommits(t, "OPS-7")
	lines := strings.Split(view(e.m), "\n")
	assertFits(t, lines, 160, 40)
	out := strings.Join(lines, "\n")
	for _, want := range []string{
		"OPS-7", "✓ Tarea  ›  Commits  ›  Vista previa  ›  Ubicación  ›  Publicar",
		"HASH", "MENSAJE", "AUTOR", "FECHA", "ARCH.", "Agrega filtro por rango de fechas", "Ana", "2026-10-19",
		"a1b2c3d Agrega filtro por rango de fechas", "Ana · 2026-10-19 · develop", "ARCHIVOS (2)", "app/Http/Pagos/FiltroController.php", "routes/api.php",
		"SELECCIONADOS  0 commits · 0 archivos   ninguno: marca con espacio",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	if at := strings.Index(lineWith(t, lines, "ARCHIVOS (2)"), "ARCHIVOS (2)"); at < 90 {
		t.Errorf("el detalle va a la derecha de la lista: columna %d", at)
	}
	strip := 0
	for i, l := range lines {
		if strings.Contains(l, "SELECCIONADOS") {
			strip = i
		}
	}
	if !strings.Contains(lines[strip-2], "╰") || !strings.Contains(bottomEdge(lines), "╰") {
		t.Errorf("las cajas llegan hasta la franja y la franja hasta el pie:\n%s", out)
	}
}

func TestCommitsReadTheFilesOfEachCommitOnce(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toCommits(t, "OPS-7")
	dir := "/code/pagos-api"
	if e.git.calls[dir] != 1 || strings.Join(e.git.read[dir], ",") != "a1b2c3d" {
		t.Fatalf("al llegar la lista se leen los archivos del primer commit: %d %v", e.git.calls[dir], e.git.read[dir])
	}
	send(e.m, keyDown)
	if e.git.calls[dir] != 2 || strings.Join(e.git.read[dir], ",") != "9f8e7d6" {
		t.Errorf("al bajar el cursor se lee el commit nuevo, de un solo hash: %d %v", e.git.calls[dir], e.git.read[dir])
	}
	send(e.m, keyUp)
	send(e.m, keyDown)
	if e.git.calls[dir] != 2 {
		t.Errorf("un commit ya leído no se vuelve a leer: %d llamadas", e.git.calls[dir])
	}
	typeText(e.m, "hf1e2d3c")
	send(e.m, keyEnter)
	if e.git.calls[dir] != 3 || strings.Join(e.git.read[dir], ",") != "f1e2d3c" {
		t.Errorf("un hash a mano también se lee: %d %v", e.git.calls[dir], e.git.read[dir])
	}
	if out := view(e.m); !strings.Contains(out, "(a mano)") || !strings.Contains(out, "ARCHIVOS (2)") {
		t.Errorf("el hash a mano muestra sus archivos:\n%s", out)
	}
}

func TestCommitsStripCountsFilesOncePerSide(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.resize(160, 40)
	e.toCommits(t, "OPS-7")
	send(e.m, keySpace)
	out := view(e.m)
	for _, want := range []string{"1 commit · 2 archivos", "✓ a1b2c3d Agrega filtro por rango", "frontend: ninguno"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	send(e.m, keyDown)
	send(e.m, keySpace)
	if out := view(e.m); !strings.Contains(out, "2 commits · 2 archivos") {
		t.Errorf("dos commits del mismo lado con los mismos archivos cuentan cada archivo una vez:\n%s", out)
	}
	send(e.m, tea.KeyPressMsg{Code: tea.KeyTab})
	send(e.m, keySpace)
	out = view(e.m)
	if !strings.Contains(out, "3 commits · 3 archivos") || strings.Contains(out, "ninguno") || !strings.Contains(out, "✓ e4f5a6b Filtro de fechas") {
		t.Errorf("la franja suma los dos repos, y cada lado cuenta sus archivos:\n%s", out)
	}
	send(e.m, tea.KeyPressMsg{Code: tea.KeyTab})
	send(e.m, keySpace)
	if out := view(e.m); !strings.Contains(out, "2 commits · 3 archivos") || !strings.Contains(out, "Backend (1)") {
		t.Errorf("desmarcar baja la cuenta:\n%s", out)
	}
}

func TestBarPresetReadsTheFilesOfTheMarkedHashes(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.resize(160, 40)
	runBar(e.m, "g -i OPS-7 -b a1b2c3d,ffff999 -f e4f5")
	out := view(e.m)
	if !strings.Contains(out, "3 commits · 3 archivos") || !strings.Contains(out, "Backend (2)") || !strings.Contains(out, "✓ ffff999 (a mano)") {
		t.Errorf("los hashes de la barra llegan marcados y con sus archivos leídos, sin pulsar nada:\n%s", out)
	}
}

func TestCommitsGiveWayWithoutRoom(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toCommits(t, "OPS-7")
	e.resize(110, 34)
	out := view(e.m)
	if !strings.Contains(out, "Agrega filtro por rango de fechas") || strings.Contains(out, "AUTOR") {
		t.Errorf("a 110 columnas el mensaje gana el sitio de autor y fecha, que quedan en el detalle:\n%s", out)
	}

	e.resize(80, 24)
	lines := strings.Split(view(e.m), "\n")
	assertFits(t, lines, 80, 24)
	out = strings.Join(lines, "\n")
	for _, want := range []string{"OPS-7", "▸ Backend", "HASH", "a1b2c3d", "ARCHIVOS app/Http/Pagos/FiltroController.php, routes/api.php", "MARCADOS 0 commits · 0 archivos", "Un commit de otra rama se agrega con h."} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "╭") || strings.Contains(out, "SELECCIONADOS") {
		t.Errorf("en 80 columnas el detalle baja y no hay cajas:\n%s", out)
	}
	send(e.m, keySpace)
	if out := view(e.m); !strings.Contains(out, "MARCADOS 1 commit · 2 archivos") {
		t.Errorf("marcar actualiza la cuenta:\n%s", out)
	}
}

func TestCommitsSecondaryKeysAreInTheFullHelp(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toCommits(t, "OPS-7")
	lines := strings.Split(view(e.m), "\n")
	short := lines[len(lines)-1]
	for _, want := range []string{"↑↓ elegir", "espacio marcar", "tab repo", "enter continuar", "? más"} {
		if !strings.Contains(short, want) {
			t.Errorf("el pie lleva %q: %q", want, short)
		}
	}
	if strings.Contains(short, "hash a mano") {
		t.Errorf("h solo está en la ayuda completa (y en la línea bajo la lista): %q", short)
	}
	e.resize(80, 24)
	lines = strings.Split(view(e.m), "\n")
	if short := lines[len(lines)-1]; !strings.Contains(short, "esc volver") || !strings.Contains(short, "? más") || strings.Contains(short, "…") {
		t.Errorf("a 80 columnas el pie cabe completo, con ? más: %q", short)
	}
	e.resize(110, 34)
	send(e.m, keyHelp)
	if full := squash(view(e.m)); !strings.Contains(full, "h hash a mano") || !strings.Contains(full, "↑/k subir") {
		t.Errorf("la ayuda completa lleva todas:\n%s", full)
	}
}

func TestLocationsShowASummaryBesideTheList(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toPreview(t, "")
	send(e.m, keyEnter)
	e.resize(160, 40)
	lines := strings.Split(view(e.m), "\n")
	assertFits(t, lines, 160, 40)
	out := strings.Join(lines, "\n")
	for _, want := range []string{
		"¿Dónde se crea?", "✓ Tarea  ›  ✓ Commits  ›  ✓ Vista previa  ›  Ubicación  ›  Publicar", "/ filtrar", "▸ ./ (raíz de la carpeta)",
		"RESUMEN", "Título", "Documento de Despliegue - OPS-7", "Destino  markdown · ./ (raíz de la carpeta)", "Página   nueva, junto a «./ (raíz de la carpeta)»",
		"Después  enlace remoto en OPS-7 (jira)", "enter pasa a confirmar; nada se publica todavía.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	if at := strings.Index(lineWith(t, lines, "RESUMEN"), "RESUMEN"); at < 90 {
		t.Errorf("el resumen va a la derecha de la lista: columna %d", at)
	}
	if !strings.Contains(bottomEdge(lines), "╰") {
		t.Errorf("las cajas llegan hasta el pie:\n%s", out)
	}

	typeText(e.m, "/zzz")
	if out := view(e.m); !strings.Contains(out, "Página   nueva, junto a «—»") || !strings.Contains(out, "/ zzz") {
		t.Errorf("sin lugar elegido el resumen no inventa uno:\n%s", out)
	}
}

func TestLocationsSummaryGoesUnderTheListInNarrowTerminals(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toPreview(t, "")
	send(e.m, keyEnter)
	e.resize(80, 24)
	lines := strings.Split(view(e.m), "\n")
	assertFits(t, lines, 80, 24)
	out := strings.Join(lines, "\n")
	for _, want := range []string{"/ filtrar", "▸ ./ (raíz de la carpeta)", "Título", "Después  enlace remoto en OPS-7"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "╭") || strings.Contains(out, "¿Dónde se crea?") {
		t.Errorf("en 80 columnas no hay cajas ni línea de pasos:\n%s", out)
	}
}

func TestPublishShowsTheStepsAndKeepsItsText(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toPreview(t, "")
	send(e.m, keyEnter)
	send(e.m, keyEnter)
	e.resize(160, 40)
	out := view(e.m)
	for _, want := range []string{"✓ Tarea  ›  ✓ Commits  ›  ✓ Vista previa  ›  ✓ Ubicación  ›  Publicar", "Crear junto a «./ (raíz de la carpeta)»", "¿Publicar? enter confirma", "$ draco g -i OPS-7 -b a1b2c3d -p pagos"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "✓ Documento creado") || !strings.Contains(out, "╭") {
		t.Errorf("el resultado va en una caja:\n%s", out)
	}
}

func TestPreparingShowsTheSteps(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.resize(160, 40)
	e.toCommits(t, "OPS-7")
	send(e.m, keySpace)
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "Preparar documento") || !strings.Contains(out, "✓ Tarea  ›  ✓ Commits  ›  Vista previa") {
		t.Errorf("preparar el documento lleva la línea de pasos:\n%s", out)
	}
}
