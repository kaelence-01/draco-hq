package document

import "strings"

// mdSpecial are the characters that could start markup, close a table cell,
// open HTML or an entity when interpolated into Markdown.
const mdSpecial = "\\`*_[]<>|~#!&"

// mdText escapes s as inline Markdown text; line breaks fold to spaces.
func mdText(s string) string {
	var b strings.Builder
	for _, r := range strings.Join(strings.Fields(s), " ") {
		if strings.ContainsRune(mdSpecial, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// mdURL makes u safe as a link destination.
func mdURL(u string) string {
	return strings.NewReplacer(" ", "%20", "(", "%28", ")", "%29", "<", "%3C", ">", "%3E", "|", "%7C").Replace(u)
}

func link(text, url string) string { return "[" + mdText(text) + "](" + mdURL(url) + ")" }

// card renders an URL as an autolink, which destinations show as a card;
// "—" when there is none.
func card(url string) string {
	if strings.TrimSpace(url) == "" {
		return "—"
	}
	return "<" + mdURL(url) + ">"
}

// orDash renders an unset value as an em dash, so a missing config key is
// visibly empty in the published report instead of silently absent.
func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return mdText(s)
}
