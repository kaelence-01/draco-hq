package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// saveDialog asks where to save a file with the system's own dialog and
// returns the chosen path, or "" when there is no dialog or it was canceled.
// Tests replace it: in WSL it would open a real Windows window.
var saveDialog = nativeSaveDialog

func nativeSaveDialog(defaultName string) string {
	switch runtime.GOOS {
	case "darwin":
		return osascriptDialog(defaultName)
	case "windows":
		return powershellDialog("powershell", defaultName)
	default:
		if p := zenityDialog(defaultName); p != "" {
			return p
		}
		return wslPath(powershellDialog("powershell.exe", defaultName))
	}
}

// output runs a dialog program by its absolute path (P-005); "" if it is not
// installed or the user canceled.
func output(name string, args ...string) string {
	path, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	out, err := exec.Command(path, args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func zenityDialog(defaultName string) string {
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return ""
	}
	cwd, _ := os.Getwd()
	return output("zenity", "--file-selection", "--save", "--confirm-overwrite",
		"--title=Guardar como...", "--filename="+filepath.Join(cwd, defaultName),
		"--file-filter=Archivos de texto (*.txt) | *.txt")
}

func powershellDialog(exe, defaultName string) string {
	// The name goes inside a single-quoted PowerShell string: '' is its escape.
	script := fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms
$d = New-Object System.Windows.Forms.SaveFileDialog
$d.Title = 'Guardar documento'
$d.Filter = 'Archivos de texto (*.txt)|*.txt'
$d.FileName = '%s'
$d.InitialDirectory = [System.Environment]::GetFolderPath('Desktop')
if ($d.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { $d.FileName }`, strings.ReplaceAll(defaultName, "'", "''"))
	return output(exe, "-NoProfile", "-NonInteractive", "-Command", script)
}

// wslPath turns a Windows path chosen from WSL into its /mnt/<drive> form.
func wslPath(p string) string {
	if len(p) >= 2 && p[1] == ':' {
		return "/mnt/" + strings.ToLower(p[:1]) + strings.ReplaceAll(p[2:], `\`, "/")
	}
	return p
}

func osascriptDialog(defaultName string) string {
	name := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(defaultName)
	return output("osascript", "-e", fmt.Sprintf(`POSIX path of (choose file name with prompt "Guardar documento:" default name "%s")`, name))
}
