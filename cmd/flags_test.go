package cmd

import (
	"reflect"
	"strings"
	"testing"
)

var genFlags = []flag{
	{long: "--issue", short: "-i", value: true},
	{long: "--commit-backend", short: "-b", value: true},
	{long: "--space", short: "-s", value: true},
	{long: "--dry-run"},
}

func TestParseFlagsAllForms(t *testing.T) {
	for _, args := range [][]string{
		{"-i", "OPS-7", "-b", "a1b2c3d", "--dry-run"},
		{"--issue", "OPS-7", "--commit-backend=a1b2c3d", "--dry-run"},
		{"-i=OPS-7", "-b=a1b2c3d", "--dry-run"},
	} {
		p, err := parseFlags(args, genFlags)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if p.get("--issue") != "OPS-7" || p.get("--commit-backend") != "a1b2c3d" || !p.has("--dry-run") || p.has("--space") {
			t.Errorf("%v → %+v", args, p.values)
		}
	}
}

func TestParseFlagsKeepsPositionals(t *testing.T) {
	p, err := parseFlags([]string{"add", "pagos", "--dry-run", "--", "--no-es-flag"}, genFlags)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(p.args, []string{"add", "pagos", "--no-es-flag"}) || !p.has("--dry-run") {
		t.Errorf("positionals = %v, flags = %v", p.args, p.values)
	}
}

func TestParseFlagsBooleanDoesNotEatTheNextArgument(t *testing.T) {
	p, err := parseFlags([]string{"--dry-run", "scan"}, genFlags)
	if err != nil || !reflect.DeepEqual(p.args, []string{"scan"}) {
		t.Errorf("un flag sin valor no consume el siguiente argumento: %+v %v", p, err)
	}
}

func TestParseFlagsErrors(t *testing.T) {
	cases := map[string][]string{
		"flag desconocido: --isue": {"--isue", "OPS-7"},
		"flag desconocido: -x":     {"-x"},
		"falta el valor de -i":     {"-i"},
		"--dry-run no lleva valor": {"--dry-run=si"},
	}
	for want, args := range cases {
		if _, err := parseFlags(args, genFlags); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%v: esperaba %q, obtuve %v", args, want, err)
		}
	}
}

func TestShortFlagsArePerCommand(t *testing.T) {
	qa := []flag{{long: "--sprint", short: "-s", value: true}}
	if p, _ := parseFlags([]string{"-s", "12"}, qa); p.get("--sprint") != "12" {
		t.Errorf("-s en qa es --sprint: %v", p.values)
	}
	if p, _ := parseFlags([]string{"-s", "DEV"}, genFlags); p.get("--space") != "DEV" {
		t.Errorf("-s en generate es --space: %v", p.values)
	}
}

func TestParseFlagsOptionalValue(t *testing.T) {
	flags := []flag{{long: "--output", short: "-o", value: true, optional: true}, {long: "--repo", short: "-r", value: true}}
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
		p, err := parseFlags(c.args, flags)
		if err != nil || !p.has("--output") || p.get("--output") != c.want {
			t.Errorf("%v → %+v %v, want %q", c.args, p, err, c.want)
		}
	}
}
