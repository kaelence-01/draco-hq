package updater

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// The Makefile builds the release and the updater downloads from it: a
// platform in only one of them is a binary nobody can update to.
func TestMakefilePublishesEveryPlatform(t *testing.T) {
	b, err := os.ReadFile("../../Makefile")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^PLATFORMS := (.+)$`).FindSubmatch(b)
	if m == nil {
		t.Fatal("el Makefile no define PLATFORMS")
	}
	if got := strings.Fields(string(m[1])); !slices.Equal(got, Platforms) {
		t.Errorf("Makefile %v, updater.Platforms %v", got, Platforms)
	}
}
