package backlog

import (
	"reflect"
	"testing"
)

func mustPatterns(t *testing.T, entries ...string) []modulePattern {
	t.Helper()
	patterns, errs := parseModulePatterns(entries)
	if len(errs) > 0 {
		t.Fatalf("patrones inválidos: %v", errs)
	}
	return patterns
}

func TestModuleOf(t *testing.T) {
	patterns := mustPatterns(t,
		"Facturacion=resources/views/reportesFac/**",
		"Inventario=src/app/almacen/**",
		"app/Http/{modulo}/**",
		"src/app/{modulo}/**",
	)
	for path, want := range map[string]string{
		"app/Http/Pagos/BusinessLogic/X.php":                   "Pagos",
		"app/Http/CuentasPorCobrar/Controllers/Y.php":          "CuentasPorCobrar",
		"src/app/cuentas-por-cobrar/pages/z.component.ts":      "CuentasPorCobrar",
		"src/app/pagos/services/cobro-pag.service.ts":          "Pagos",
		"resources/views/reportesFac/reporteMensual.blade.php": "Facturacion", // fixed name, listed first
		"src/app/almacen/components/x.component.ts":            "Inventario",
		"app/Http/Controller.php":                              "", // a file under app/Http is not a module
		"app/Helpers/helper.php":                               "",
	} {
		if got := moduleOf(patterns, path); got != want {
			t.Errorf("moduleOf(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestModuleOfFirstMatchWins(t *testing.T) {
	patterns := mustPatterns(t, "FAC=app/Http/Facturacion/**/Fac*", "app/Http/{modulo}/**")

	if got := moduleOf(patterns, "app/Http/Facturacion/BusinessLogic/FacItemBusinessLogic.php"); got != "FAC" {
		t.Errorf("la regla fija listada primero debe ganar: %q", got)
	}
	if got := moduleOf(patterns, "app/Http/Facturacion/BusinessLogic/ClienteBusinessLogic.php"); got != "Facturacion" {
		t.Errorf("lo que no es Fac* cae en la regla general: %q", got)
	}
}

func TestParseModulePatternsRejectsAmbiguousEntries(t *testing.T) {
	for _, bad := range []string{
		"app/**/{modulo}/x", // {modulo} after ** is ambiguous
		"app/{modulo}{modulo}/x",
		"app/Http/**",        // no {modulo} and no fixed name
		"=app/Http/**",       // empty fixed name
		"Pagos=app/{modulo}", // fixed name with {modulo}
	} {
		if _, errs := parseModulePatterns([]string{bad}); len(errs) != 1 {
			t.Errorf("%q debe ser inválido", bad)
		}
	}
}

func TestModuleDisplayName(t *testing.T) {
	for in, want := range map[string]string{"pagos": "Pagos", "cuentas-por-cobrar": "CuentasPorCobrar", "Pagos": "Pagos", "ñandú_mod": "ÑandúMod"} {
		if got := moduleDisplayName(in); got != want {
			t.Errorf("moduleDisplayName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestModuleKey(t *testing.T) {
	if ModuleKey("Cuentas-Por-Cobrar") != ModuleKey("cuentasporcobrar") || ModuleKey("Pagos") != "pagos" {
		t.Error("ModuleKey debe ignorar mayúsculas, guiones y espacios")
	}
}

func TestCountModulesAndFilter(t *testing.T) {
	rep := &Report{Findings: []Finding{
		{ID: "1", Module: "Pagos"}, {ID: "2", Module: "Facturacion"},
		{ID: "3", Module: "Facturacion"}, {ID: "4", Module: ""},
	}}
	rep.Total = len(rep.Findings)
	rep.Modules = countModules(rep.Findings)

	want := []ModuleCount{{"Facturacion", 2}, {"Pagos", 1}, {"", 1}} // sin módulo, al final
	if !reflect.DeepEqual(rep.Modules, want) {
		t.Errorf("countModules: %#v", rep.Modules)
	}

	rep.FilterModules([]string{"pagos"})
	if rep.Total != 1 || rep.Findings[0].ID != "1" || !reflect.DeepEqual(rep.ModuleFilter, []string{"pagos"}) {
		t.Errorf("FilterModules: total=%d findings=%#v filtro=%#v", rep.Total, rep.Findings, rep.ModuleFilter)
	}
	if len(rep.Modules) != 3 {
		t.Error("el resumen por módulo debe seguir siendo el del scan completo")
	}
}
