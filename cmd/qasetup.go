package cmd

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/kaelence-01/draco-hq/internal/cli"
	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/profile"
	"github.com/kaelence-01/draco-hq/internal/provider"
	"github.com/kaelence-01/draco-hq/internal/qa"
)

var qaSetupFlags = []cli.Flag{
	{Long: "--jira-project", Short: "-j", Value: true, Desc: "Proyecto de Jira a configurar (sin él: lo pregunta)"},
}

// runQASetup fills qa_jira from what Jira lists: the user picks statuses and
// fields by number, never types an id.
func runQASetup(a *App, args []string) error {
	f, err := cli.Parse(args, qaSetupFlags)
	if err != nil {
		return usageError("%v", err)
	}
	if len(f.Args) > 0 {
		return usageError("argumento inesperado: %s", f.Args[0])
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	file, err := config.LoadFile()
	if err != nil {
		return err
	}
	project := f.Get("--jira-project")
	if project == "" {
		def := ""
		if file.QAJira != nil {
			def = file.QAJira.Project
		}
		if project, err = a.askValue("Proyecto de Jira (la clave de la URL del tablero)", def, true); err != nil {
			return err
		}
	}
	bundle, err := openProfile(cfg, profile.Options{AnyScope: true})
	if err != nil {
		return explainProfile(err)
	}
	src, err := bundle.QASetup()
	if err != nil {
		return err
	}
	ctx, cancel := a.context()
	defer cancel()
	s := qa.NewSetup(src)
	a.printf("\n")
	a.step(1, 2, "Leyendo los estados de "+project+" y los campos de Jira...")
	if err := s.Read(ctx, project); err != nil {
		return err
	}
	c := s.Propose(file.QAJira)

	a.printf("\n")
	a.setupSummary(s, c)
	pickStatuses := true
	switch {
	case c.Complete():
		if a.confirm("¿Guardar así? (n: cambiar)", true) {
			return a.saveSetup(file, s, c)
		}
	default:
		pickStatuses = !a.confirm("¿Usar estos estados? (n: elegirlos de la lista)", true)
	}

	a.printf("\n")
	a.step(2, 2, "Elige por número (Enter: lo marcado con *)")
	if pickStatuses {
		if err := a.pickStatuses(s, &c); err != nil {
			return err
		}
	}
	return a.pickFieldsAndSave(file, s, c)
}

// pickStatuses asks the testing status and the statuses that reached QA.
func (a *App) pickStatuses(s *qa.Setup, c *qa.Choice) error {
	a.printf("\n%s\n", a.style(bold, "Estados de "+s.Project+":"))
	names := make([]string, len(s.Statuses))
	for i, st := range s.Statuses {
		names[i] = st.Name
		if st.Done {
			names[i] += "  (finalizado)"
		}
	}
	testing := slices.IndexFunc(s.Statuses, func(st provider.Status) bool { return st.Name == c.Testing })
	reached := indexesOf(s.Statuses, func(st provider.Status) string { return st.ID }, c.Statuses)
	a.list(names, reached)
	var err error
	if testing, err = a.pickOne("Estado con el que una tarea entra a QA", len(names), testing); err != nil {
		return err
	}
	if reached, err = a.pickMany("Estados de una tarea que ya llegó a QA (separados por comas)", len(names), reached); err != nil {
		return err
	}
	c.Testing, c.Statuses = s.Statuses[testing].Name, nil
	for _, i := range reached {
		c.Statuses = append(c.Statuses, s.Statuses[i].ID)
	}
	return nil
}

// pickFieldsAndSave asks the counters and, with several, the PR field; then
// saves.
func (a *App) pickFieldsAndSave(file *config.Config, s *qa.Setup, c qa.Choice) error {
	numbers := s.FieldsOf(provider.FieldNumber)
	if len(numbers) == 0 {
		return fmt.Errorf("Jira no tiene campos numéricos personalizados: el consolidado los usa para los errores de codificación y las devoluciones")
	}
	a.printf("\n%s\n", a.style(bold, "Campos numéricos de Jira:"))
	fieldNames := make([]string, len(numbers))
	for i, fd := range numbers {
		fieldNames[i] = fd.Name + "  (" + fd.ID + ")"
	}
	a.list(fieldNames, nil)
	for _, pick := range []struct {
		label string
		id    *string
	}{
		{"Campo de errores de codificación", &c.CodingErrors},
		{"Campo de devoluciones a desarrollo", &c.DevReturns},
	} {
		def := slices.IndexFunc(numbers, func(fd provider.Field) bool { return fd.ID == *pick.id })
		i, err := a.pickOne(pick.label, len(numbers), def)
		if err != nil {
			return err
		}
		*pick.id = numbers[i].ID
	}

	prs := s.FieldsOf(provider.FieldPRStatus)
	switch len(prs) {
	case 0:
		return fmt.Errorf("Jira no tiene el campo de desarrollo (Development), del que el consolidado lee el estado del PR")
	case 1:
		c.DevStatus = prs[0].ID
		a.printf("\n")
		a.ok("Estado del PR: " + prs[0].Name + "  (" + prs[0].ID + ")")
	default:
		a.printf("\n%s\n", a.style(bold, "Campos de desarrollo de Jira:"))
		prNames := make([]string, len(prs))
		for i, fd := range prs {
			prNames[i] = fd.Name + "  (" + fd.ID + ")"
		}
		a.list(prNames, nil)
		def := slices.IndexFunc(prs, func(fd provider.Field) bool { return fd.ID == c.DevStatus })
		i, err := a.pickOne("Campo del estado del PR", len(prs), def)
		if err != nil {
			return err
		}
		c.DevStatus = prs[i].ID
	}
	return a.saveSetup(file, s, c)
}

// setupSummary prints what would be saved, by name, and where it came from.
func (a *App) setupSummary(s *qa.Setup, c qa.Choice) {
	chosen, others := s.StatusNames(c.Statuses)
	origin := "Propuesto desde tu configuración."
	if c.Suggested {
		origin = "Sugerido por draco: el estado " + config.DefaultTestingStatus + " y los finalizados. Revisa si falta alguno de los demás."
	}
	a.printf("%s\n  %s\n\n", a.style(bold, "Proyecto "+s.Project), origin)
	row := func(label, value string) {
		if value == "" {
			value = a.style(yellow, "(falta elegirlo)")
		}
		a.printf("  %-27s %s\n", label, value)
	}
	row("Entra a QA con", c.Testing)
	row(fmt.Sprintf("Ya llegó a QA (%d)", len(chosen)), strings.Join(chosen, ", "))
	row("Los demás estados", strings.Join(others, ", "))
	row("Errores de codificación", s.FieldName(c.CodingErrors))
	row("Devoluciones a desarrollo", s.FieldName(c.DevReturns))
	row("Estado del PR", s.FieldName(c.DevStatus))
	a.printf("\n")
}

func (a *App) saveSetup(file *config.Config, s *qa.Setup, c qa.Choice) error {
	q, err := s.Apply(file.QAJira, c)
	if err != nil {
		return err
	}
	file.QAJira = q
	if err := saveConfig(file); err != nil {
		return err
	}
	a.printf("\n")
	a.ok(a.style(green+bold, fmt.Sprintf("QA guardado: proyecto %s, %d estados, entra a QA con «%s»", s.Project, len(c.Statuses), c.Testing)))
	if file.QAEmail == "" {
		a.warn("falta qa_email, la cuenta que publica el consolidado: complétalo en Configuración › QA")
	}
	return nil
}

func indexesOf[T any](items []T, id func(T) string, ids []string) []int {
	var out []int
	for i, it := range items {
		if slices.Contains(ids, id(it)) {
			out = append(out, i)
		}
	}
	return out
}

// list prints items numbered from 1, with * on the marked ones.
func (a *App) list(items []string, marked []int) {
	for i, it := range items {
		mark := " "
		if slices.Contains(marked, i) {
			mark = a.style(green, "*")
		}
		a.printf("  %s %s %s\n", a.style(bold+yellow, fmt.Sprintf("[%d]", i+1)), mark, it)
	}
}

// pickOne asks for one number of a list of n; Enter takes def (-1: none).
func (a *App) pickOne(label string, n, def int) (int, error) {
	defText := ""
	if def >= 0 {
		defText = strconv.Itoa(def + 1)
	}
	for {
		v, err := a.askValue(label, defText, true)
		if err != nil {
			return 0, err
		}
		if i, err := strconv.Atoi(v); err == nil && i >= 1 && i <= n {
			return i - 1, nil
		}
		a.warn(fmt.Sprintf("ingresa un número entre 1 y %d", n))
	}
}

// pickMany asks for numbers of a list of n, separated by commas; Enter takes
// def.
func (a *App) pickMany(label string, n int, def []int) ([]int, error) {
	var defText []string
	for _, i := range def {
		defText = append(defText, strconv.Itoa(i+1))
	}
	for {
		v, err := a.askValue(label, strings.Join(defText, ","), true)
		if err != nil {
			return nil, err
		}
		if out, ok := parseIndexes(v, n); ok {
			return out, nil
		}
		a.warn(fmt.Sprintf("ingresa números entre 1 y %d separados por comas", n))
	}
}

func parseIndexes(s string, n int) ([]int, bool) {
	var out []int
	for _, v := range config.SplitList(s) {
		i, err := strconv.Atoi(v)
		if err != nil || i < 1 || i > n {
			return nil, false
		}
		if !slices.Contains(out, i-1) {
			out = append(out, i-1)
		}
	}
	return out, len(out) > 0
}
