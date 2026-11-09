package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// openURL opens a document: a page in the browser, a Markdown file with its
// program. Tests replace it so nothing opens.
var openURL = nativeOpenURL

func nativeOpenURL(target string) error {
	// Only a web address or an absolute path: the history file is plain text.
	if !strings.HasPrefix(target, "https://") && !strings.HasPrefix(target, "http://") && !filepath.IsAbs(target) {
		return fmt.Errorf("%q no es una dirección web ni una ruta absoluta", target)
	}
	for _, opener := range openers() {
		path, err := exec.LookPath(opener[0])
		if err != nil {
			continue
		}
		c := exec.Command(path, append(opener[1:], target)...)
		if err := c.Start(); err != nil {
			return err
		}
		// Only to reap it: explorer.exe exits with 1 even when it opens.
		go c.Wait()
		return nil
	}
	return errors.New("no se encontró con qué abrirlo (xdg-open o wslview)")
}

// openers are the programs that open a link on this system, in order.
func openers() [][]string {
	switch runtime.GOOS {
	case "darwin":
		return [][]string{{"open"}}
	case "windows":
		return [][]string{{"rundll32", "url.dll,FileProtocolHandler"}}
	}
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		return [][]string{{"wslview"}, {"explorer.exe"}, {"xdg-open"}}
	}
	return [][]string{{"xdg-open"}}
}
