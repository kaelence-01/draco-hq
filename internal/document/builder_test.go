package document

import (
	"reflect"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"

	"github.com/kaelence-01/draco-hq/internal/provider"
)

func TestBuildTitleSanitizesCharactersConfluenceRejects(t *testing.T) {
	cases := []struct {
		summary, want string
	}{
		{
			"Modal para Acción y Validaciones",
			"Documento de Despliegue - OPS-1999 - Modal para Acción y Validaciones",
		},
		{
			"Ajuste FAC/Pagos: revisión [urgente] | v2",
			"Documento de Despliegue - OPS-1999 - Ajuste FAC-Pagos- revisión (urgente) - v2",
		},
		{
			"   espacios alrededor   ",
			"Documento de Despliegue - OPS-1999 - espacios alrededor",
		},
	}
	for _, c := range cases {
		if got := BuildTitle("OPS-1999", c.summary); got != c.want {
			t.Errorf("BuildTitle(%q)\n  = %q\n  esperaba %q", c.summary, got, c.want)
		}
	}
}

func TestBuildTitleUsesThePrefixSearchesRelyOn(t *testing.T) {
	// FindLastDeployDoc filters by this prefix; without it the location picker is empty.
	if got := BuildTitle("OPS-1", "x"); !strings.HasPrefix(got, "Documento de Despliegue") {
		t.Errorf("el título perdió el prefijo esperado: %q", got)
	}
}

func TestCommitFileURLPerHost(t *testing.T) {
	const (
		repo   = "https://bitbucket.org/mi-org/pagos-api"
		commit = "27cefd86"
		file   = "app/Models/Fac.php"
	)
	cases := []struct {
		name, host, want string
	}{
		{
			name: "bitbucket conserva el ancla por archivo",
			host: "https://bitbucket.org",
			want: repo + "/commits/" + commit + "#chg-" + file,
		},
		{
			name: "github usa /commit singular y sin ancla",
			host: "https://github.com",
			want: repo + "/commit/" + commit,
		},
		{
			name: "gitlab usa /-/commit",
			host: "https://gitlab.com",
			want: repo + "/-/commit/" + commit,
		},
		{
			name: "host desconocido no genera link roto",
			host: "https://git.empresa.com",
			want: "",
		},
		{
			name: "host vacío no genera link",
			host: "",
			want: "",
		},
	}
	for _, c := range cases {
		if got := commitFileURL(c.host, repo, commit, file); got != c.want {
			t.Errorf("%s: commitFileURL(%q) = %q, esperaba %q", c.name, c.host, got, c.want)
		}
	}
}

func TestCommitFileURLNeedsRepoAndCommit(t *testing.T) {
	if got := commitFileURL("https://bitbucket.org", "", "abc", "f.php"); got != "" {
		t.Errorf("sin repoURL esperaba \"\", obtuve %q", got)
	}
	if got := commitFileURL("https://bitbucket.org", "https://x/y", "", "f.php"); got != "" {
		t.Errorf("sin commit esperaba \"\", obtuve %q", got)
	}
}

func TestDeployMarkdownOmitsEmptySides(t *testing.T) {
	md := DeployMarkdown(DeployDoc{
		IssueKey:     "OPS-1999",
		BackendRepo:  "pagos-api",
		BackendFiles: map[string][]string{"app": {"Fac.php"}},
	})
	if strings.Contains(md, "Frontend:") {
		t.Errorf("sin archivos de frontend no debe haber sección de Frontend:\n%s", md)
	}
	if !strings.Contains(md, "### Proyectos y formularios - Backend:") {
		t.Errorf("faltó la sección de Backend:\n%s", md)
	}
}

func checklist(md string) []string {
	var out []string
	for _, line := range strings.Split(md, "\n") {
		if step, ok := strings.CutPrefix(line, "- [ ] "); ok {
			out = append(out, step)
		}
	}
	return out
}

func TestChecklistComesFromConfig(t *testing.T) {
	got := checklist(DeployMarkdown(DeployDoc{Checklist: []string{"Pasar API", "php artisan migrate --force", "Invalidar CDN"}}))
	want := []string{"Pasar API", "php artisan migrate --force", "Invalidar CDN"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("checklist = %#v, esperaba %#v", got, want)
	}
}

func TestChecklistFallsBackToGenericDefault(t *testing.T) {
	got := checklist(DeployMarkdown(DeployDoc{}))
	if !reflect.DeepEqual(got, defaultChecklist) {
		t.Errorf("checklist = %#v, esperaba el default %#v", got, defaultChecklist)
	}
	for _, step := range got {
		if strings.Contains(step, "artisan") {
			t.Errorf("el default no debe asumir un stack: %q", step)
		}
	}
}

func TestChecklistSkipsBlankSteps(t *testing.T) {
	if got := checklist(DeployMarkdown(DeployDoc{Checklist: []string{"Pasar API", "   ", "", "Invalidar CDN"}})); len(got) != 2 {
		t.Errorf("esperaba 2 pasos, obtuve %#v", got)
	}
}

func TestMDTextEscapesMarkup(t *testing.T) {
	if got := mdText("a|b *c* <x> [d](e) &amp;\nfin"); got != `a\|b \*c\* \<x\> \[d\](e) \&amp; fin` {
		t.Errorf("mdText = %q", got)
	}
}

// cells counts the cells of every table row, as goldmark parses them.
func cells(t *testing.T, md string) []int {
	t.Helper()
	src := []byte(md)
	doc := goldmark.New(goldmark.WithExtensions(extension.Table)).Parser().Parse(text.NewReader(src))
	var out []int
	_ = gast.Walk(doc, func(n gast.Node, entering bool) (gast.WalkStatus, error) {
		if entering && (n.Kind() == east.KindTableRow || n.Kind() == east.KindTableHeader) {
			out = append(out, n.ChildCount())
		}
		return gast.WalkContinue, nil
	})
	return out
}

func TestValuesCannotBreakTheFilesTable(t *testing.T) {
	md := DeployMarkdown(DeployDoc{
		IssueKey:     "OPS-1",
		BackendRepo:  "pagos|api",
		BackendFiles: map[string][]string{"app|x": {"a|b.php", "<script>.php"}},
		VCSHost:      "https://bitbucket.org", VCSOrg: "mi-org", BackendCommit: "abc",
	})
	five := 0
	for _, n := range cells(t, md) {
		switch n {
		case 5:
			five++
		case 2:
		default:
			t.Fatalf("una fila tiene %d celdas:\n%s", n, md)
		}
	}
	if five != 2 {
		t.Errorf("la tabla de archivos debía tener encabezado y una fila, tiene %d filas de 5 celdas:\n%s", five, md)
	}
}

func TestQAMarkdownMarksAndUnknown(t *testing.T) {
	md := QAMarkdown(QADoc{Sprint: 3, Module: "Pagos", Tasks: []provider.QAIssue{
		{URL: "https://example.test/browse/OPS-1", HasCodingErrors: true, DeployDocUnknown: true, Observations: "monto | sin validar"},
	}})
	for _, want := range []string{"| ✖ | ✔ | ? | ✖ |", `monto \| sin validar`, "### Observación", "### Consideraciones", "| **Sprint** | 3 |"} {
		if !strings.Contains(md, want) {
			t.Errorf("falta %q en:\n%s", want, md)
		}
	}
	for _, n := range cells(t, md) {
		if n != 2 && n != 7 && n != 1 {
			t.Errorf("fila con %d celdas:\n%s", n, md)
		}
	}
}

func TestOrDashRendersMissingConfigVisibly(t *testing.T) {
	for _, in := range []string{"", "   ", "\t"} {
		if got := orDash(in); got != "—" {
			t.Errorf("orDash(%q) = %q, esperaba —", in, got)
		}
	}
	if got := orDash("Nombre Real"); got != "Nombre Real" {
		t.Errorf("orDash alteró un valor presente: %q", got)
	}
}
