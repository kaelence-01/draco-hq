package config

// QAState is how far the config is from publishing the QA report.
type QAState int

const (
	QAUnset      QAState = iota // nothing of qa_jira is set: QA is optional
	QAIncomplete                // qa_jira is set but Validate reports problems
	QANoAccount                 // qa_jira is valid but qa_email is empty: nobody could publish
	QAReady
)

// QAState reports the state of the QA settings.
func (c *Config) QAState() QAState {
	switch {
	case c.QAJira == nil:
		return QAUnset
	case c.QAJira.Validate() != nil:
		return QAIncomplete
	case c.QAEmail == "":
		return QANoAccount
	}
	return QAReady
}

// HasQAReport reports whether qa_report names anybody for the report header.
func (c *Config) HasQAReport() bool {
	return c.QAReport != nil && *c.QAReport != (QAReportConfig{})
}
