package cli

import (
	"reflect"
	"strings"
	"testing"
)

var genFlags = []Flag{
	{Long: "--issue", Short: "-i", Value: true},
	{Long: "--commit-backend", Short: "-b", Value: true},
	{Long: "--space", Short: "-s", Value: true},
	{Long: "--dry-run"},
}

func TestParseAllForms(t *testing.T) {
	for _, args := range [][]string{
		{"-i", "OPS-7", "-b", "a1b2c3d", "--dry-run"},
		{"--issue", "OPS-7", "--commit-backend=a1b2c3d", "--dry-run"},
		{"-i=OPS-7", "-b=a1b2c3d", "--dry-run"},
	} {
		p, err := Parse(args, genFlags)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if p.Get("--issue") != "OPS-7" || p.Get("--commit-backend") != "a1b2c3d" || !p.Has("--dry-run") || p.Has("--space") {
			t.Errorf("%v → %+v", args, p.Values)
		}
	}
}

func TestParseKeepsPositionals(t *testing.T) {
	p, err := Parse([]string{"add", "pagos", "--dry-run", "--", "--no-es-flag"}, genFlags)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(p.Args, []string{"add", "pagos", "--no-es-flag"}) || !p.Has("--dry-run") {
		t.Errorf("positionals = %v, flags = %v", p.Args, p.Values)
	}
}

func TestParseBooleanDoesNotEatTheNextArgument(t *testing.T) {
	p, err := Parse([]string{"--dry-run", "scan"}, genFlags)
	if err != nil || !reflect.DeepEqual(p.Args, []string{"scan"}) {
		t.Errorf("un flag sin valor no consume el siguiente argumento: %+v %v", p, err)
	}
}

func TestParseErrors(t *testing.T) {
	cases := map[string][]string{
		"flag desconocido: --isue": {"--isue", "OPS-7"},
		"flag desconocido: -x":     {"-x"},
		"falta el valor de -i":     {"-i"},
		"--dry-run no lleva valor": {"--dry-run=si"},
	}
	for want, args := range cases {
		if _, err := Parse(args, genFlags); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%v: esperaba %q, obtuve %v", args, want, err)
		}
	}
}

func TestShortFlagsArePerCommand(t *testing.T) {
	qa := []Flag{{Long: "--sprint", Short: "-s", Value: true}}
	if p, _ := Parse([]string{"-s", "12"}, qa); p.Get("--sprint") != "12" {
		t.Errorf("-s en qa es --sprint: %v", p.Values)
	}
	if p, _ := Parse([]string{"-s", "DEV"}, genFlags); p.Get("--space") != "DEV" {
		t.Errorf("-s en generate es --space: %v", p.Values)
	}
}

func TestParseOptionalValue(t *testing.T) {
	flags := []Flag{{Long: "--output", Short: "-o", Value: true, Optional: true}, {Long: "--repo", Short: "-r", Value: true}}
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"-o"}, ""},
		{[]string{"-o", "-r", "web"}, ""},
		{[]string{"-o", "r.json"}, "r.json"},
		{[]string{"-o=r.json"}, "r.json"},
	}
	for _, c := range cases {
		p, err := Parse(c.args, flags)
		if err != nil || !p.Has("--output") || p.Get("--output") != c.want {
			t.Errorf("%v → %+v %v, want %q", c.args, p, err, c.want)
		}
	}
}

func TestFindByNameOrAlias(t *testing.T) {
	cmds := []Command{{Names: []string{"generate", "gen", "g"}}, {Names: []string{"qa"}}}
	if c := Find(cmds, "g"); c == nil || c.Names[0] != "generate" {
		t.Errorf("un alias encuentra el comando: %+v", c)
	}
	if Find(cmds, "gener") != nil {
		t.Error("un prefijo no es un nombre")
	}
}
