package deploy

import (
	"strings"
	"unicode"

	"github.com/kaelence-01/draco-hq/internal/provider"
)

// CommitMatches reports whether every word of query is in the commit: at the
// start of its hash, or anywhere in its message, author, email or date
// (2006-01-02), ignoring case.
func CommitMatches(c provider.Commit, query string) bool {
	hashes := []string{strings.ToLower(c.Hash), strings.ToLower(c.ShortHash)}
	text := strings.ToLower(strings.Join([]string{c.Subject, c.Author, c.Email, c.Date.Format("2006-01-02")}, "\n"))
	for _, w := range strings.Fields(strings.ToLower(query)) {
		if !strings.Contains(text, w) && !hasPrefixAny(hashes, w) {
			return false
		}
	}
	return true
}

func hasPrefixAny(list []string, prefix string) bool {
	for _, s := range list {
		if s != "" && strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

// Mine reports whether the commit was authored with email, ignoring case; an
// empty email is nobody's.
func Mine(c provider.Commit, email string) bool {
	return email != "" && strings.EqualFold(c.Email, email)
}

// MentionsIssue reports whether text names the issue key as a whole word,
// ignoring case: OPS-7 is in "ops-7: fix" but not in "OPS-70".
func MentionsIssue(text, key string) bool {
	text, key = strings.ToLower(text), strings.ToLower(key)
	if key == "" {
		return false
	}
	for from := 0; ; {
		i := strings.Index(text[from:], key)
		if i < 0 {
			return false
		}
		start, end := from+i, from+i+len(key)
		if !wordRune(text[:start], true) && !wordRune(text[end:], false) {
			return true
		}
		from = start + 1
	}
}

// wordRune reports whether the rune next to a match continues its word: the
// last one of before (a letter or digit) or the first one of after (a digit).
func wordRune(s string, before bool) bool {
	if s == "" {
		return false
	}
	if before {
		r := []rune(s)[len([]rune(s))-1]
		return unicode.IsLetter(r) || unicode.IsDigit(r)
	}
	return unicode.IsDigit([]rune(s)[0])
}

// LooksLikeHash reports whether s can be an abbreviated commit hash: 7 to 40
// hexadecimal digits.
func LooksLikeHash(s string) bool {
	if len(s) < 7 || len(s) > 40 {
		return false
	}
	for _, r := range s {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return false
		}
	}
	return true
}
