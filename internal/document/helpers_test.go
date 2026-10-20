package document

// Test helpers over the built-in templates.

func DeployMarkdown(d DeployDoc) string {
	md, err := Deploy(MustBuiltin("deploy"), d)
	if err != nil {
		panic(err)
	}
	return md
}

func BuildTitle(key, summary string) string {
	title, err := DeployTitle(MustBuiltin("deploy"), key, summary)
	if err != nil {
		panic(err)
	}
	return title
}

func QAMarkdown(d QADoc) string {
	md, err := QA(MustBuiltin("qa"), d)
	if err != nil {
		panic(err)
	}
	return md
}

func BuildQATitle(module string, sprint int) string {
	title, err := QATitle(MustBuiltin("qa"), module, sprint, "")
	if err != nil {
		panic(err)
	}
	return title
}

func BuildQAKanbanTitle(period string) string {
	title, err := QATitle(MustBuiltin("qa"), "", 0, period)
	if err != nil {
		panic(err)
	}
	return title
}
