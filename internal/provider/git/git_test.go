package git

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestExplainGitErrorGivesActionableMessage(t *testing.T) {
	cases := map[string]string{
		"fatal: bad object abc1234":                       "no existe en el repo local",
		"fatal: not a git repository (or any parent)":     "no es un repositorio Git",
		"fatal: ambiguous argument 'abc'":                 "ambiguo",
		"fatal: unknown revision or path not in the tree": "no existe en el repo local",
	}
	for stderr, want := range cases {
		got := explainGitError(stderr, "abc1234", "/repos/api")
		if !strings.Contains(got, want) {
			t.Errorf("explainGitError(%q) no menciona %q:\n%s", stderr, want, got)
		}
	}
}

func TestExplainGitErrorFallsBackToRawStderr(t *testing.T) {
	got := explainGitError("fatal: something entirely new", "abc1234", "/repos/api")
	if !strings.Contains(got, "something entirely new") {
		t.Errorf("un stderr desconocido debe propagarse tal cual: %s", got)
	}
}

func TestParseLog(t *testing.T) {
	out := "a1b2c3d4" + logSep + "a1b2c3d" + logSep + "Filtro de pagos" + logSep + "Ana" + logSep + "2026-09-21T10:00:00-05:00\n" +
		"linea rota\n"
	got := parseLog(out)
	if len(got) != 1 {
		t.Fatalf("esperaba 1 commit, obtuve %+v", got)
	}
	c := got[0]
	if c.Hash != "a1b2c3d4" || c.ShortHash != "a1b2c3d" || c.Subject != "Filtro de pagos" || c.Author != "Ana" || c.Date.Day() != 21 {
		t.Errorf("commit = %+v", c)
	}
}

func TestLocalReadsATemporaryRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git no está instalado")
	}
	dir := t.TempDir()
	gitIn := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Ana", "GIT_AUTHOR_EMAIL=ana@example.test",
			"GIT_COMMITTER_NAME=Ana", "GIT_COMMITTER_EMAIL=ana@example.test")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	write := func(name string) {
		t.Helper()
		if err := os.MkdirAll(dir+"/app", 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dir+"/"+name, []byte(name), 0644); err != nil {
			t.Fatal(err)
		}
	}
	gitIn("init", "-q")
	write("app/Pago.php")
	gitIn("add", ".")
	gitIn("commit", "-q", "-m", "Primer commit")
	gitIn("checkout", "-q", "-b", "filtro-pagos")
	write("app/Cobro.php")
	write("app/Pago.php.bak")
	gitIn("add", ".")
	gitIn("commit", "-q", "-m", "Segundo commit")

	ctx := context.Background()
	commits, err := Local{}.RecentCommits(ctx, dir, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 2 || commits[0].Subject != "Segundo commit" || commits[0].Author != "Ana" {
		t.Fatalf("commits = %+v", commits)
	}
	if branch, err := (Local{}).CurrentBranch(ctx, dir); err != nil || branch != "filtro-pagos" {
		t.Errorf("rama = %q, %v", branch, err)
	}

	files, err := Local{}.ChangedFiles(ctx, dir, []string{commits[0].Hash, commits[1].Hash, commits[0].Hash})
	if err != nil {
		t.Fatal(err)
	}
	want := "app/Cobro.php app/Pago.php.bak app/Pago.php"
	if strings.Join(files, " ") != want {
		t.Errorf("archivos = %v, esperaba %s", files, want)
	}

	if _, err := (Local{}).ChangedFiles(ctx, dir, []string{"0000000"}); err == nil || !strings.Contains(err.Error(), "no existe") {
		t.Errorf("un hash inexistente debe dar un error accionable: %v", err)
	}
}
