package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/profile"
)

func init() {
	register(&command{run: runInit, help: initHelp, summary: "Configura credenciales, el space y un proyecto"}, "init")
}

const tokenURL = "https://id.atlassian.com/manage-profile/security/api-tokens"

func runInit(a *App, args []string) error {
	if _, err := names(args, 0, 0, "draco init"); err != nil {
		return err
	}
	path, err := config.ConfigPath()
	if err != nil {
		return err
	}
	cfg, err := config.LoadFile()
	if err != nil {
		return err
	}

	a.title("draco init", "Configuración inicial")
	if cfg.IsEmpty() {
		if imported, err := a.offerGtt(); imported || err != nil {
			return err
		}
	}
	if _, err := os.Stat(path); err == nil {
		a.warn("Ya existe una configuración en " + path)
		a.printf("    Se conservan sus proyectos, perfiles y ajustes; se piden de nuevo las credenciales.\n")
		if !a.confirm("  ¿Continuar?", false) {
			a.printf("Cancelado.\n")
			return nil
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	a.printf("  Se guarda en %s, legible solo por tu usuario.\n\n", path)

	a.section("Cuenta de Atlassian")
	if cfg.AtlassianEmail, err = a.askValue("Email", cfg.AtlassianEmail, true); err != nil {
		return err
	}
	if err := a.askToken(cfg); err != nil {
		return err
	}
	for {
		url, err := a.askValue("URL del sitio (ej: https://empresa.atlassian.net)", cfg.BaseURL, true)
		if err != nil {
			return err
		}
		if cfg.BaseURL, err = config.NormalizeBaseURL(url); err == nil {
			break
		}
		a.warn(err.Error())
	}

	if len(cfg.Profiles) == 0 {
		if cfg.ConfluenceSpaceKey, err = a.askValue("Space de Confluence de los documentos (ej: DEV)", cfg.ConfluenceSpaceKey, false); err != nil {
			return err
		}
	} else {
		a.printf("\n  Perfiles existentes: %s (ver draco profile list).\n", strings.Join(cfg.ProfileNames(), ", "))
	}

	if err := saveConfig(cfg); err != nil {
		return err
	}
	a.printf("\n")
	a.ok("Configuración guardada")

	if err := a.verifyAccount(cfg); err != nil {
		return err
	}

	a.printf("\n")
	if len(cfg.Projects) == 0 {
		a.printf("  Generar un documento necesita un proyecto: dónde están tus repos.\n")
	}
	if a.confirm("¿Configurar un proyecto ahora?", true) {
		if err := a.addProject(cfg, ""); err != nil {
			return err
		}
	}
	a.printf("\n")
	a.ok(a.style(green+bold, "Listo.") + " Prueba: draco g -i <TAREA> -b <commit> --dry-run")
	return nil
}

// offerGtt imports gtt's config, if there is one and the user wants it, and
// reports whether it did; gtt's file is never changed.
func (a *App) offerGtt() (bool, error) {
	gtt, from, err := config.GttConfig()
	if err != nil {
		a.warn("hay una configuración de gtt, pero no se pudo leer: " + err.Error())
		return false, nil
	}
	if gtt == nil {
		return false, nil
	}
	a.printf("  Se encontró la configuración de gtt en %s.\n", from)
	if !a.confirm("  ¿Importarla? La de gtt no se modifica", true) {
		a.printf("\n")
		return false, nil
	}
	if err := saveConfig(gtt); err != nil {
		return true, err
	}
	a.ok("Configuración importada de gtt")
	for _, note := range config.ImportNotes(gtt) {
		a.printf("    %s\n", note)
	}
	if err := a.verifyAccount(gtt); err != nil {
		return true, err
	}
	if len(gtt.Projects) == 0 && a.confirm("¿Configurar un proyecto ahora?", true) {
		if err := a.addProject(gtt, ""); err != nil {
			return true, err
		}
	}
	a.printf("\n")
	a.ok(a.style(green+bold, "Listo.") + " Prueba: draco g -i <TAREA> -b <commit> --dry-run")
	return true, nil
}

// askToken keeps the current token on Enter; the token is never shown.
func (a *App) askToken(cfg *config.Config) error {
	prompt := "  API token (se crea en " + tokenURL + "): "
	if cfg.AtlassianToken != "" {
		prompt = "  API token (Enter conserva el actual): "
	}
	for {
		token, err := a.askSecret(prompt)
		if err != nil {
			return err
		}
		if token != "" {
			cfg.AtlassianToken = token
		}
		if cfg.AtlassianToken != "" {
			return nil
		}
		a.warn("este dato es obligatorio")
	}
}

// verifyAccount checks the credentials against the tracker and offers to fix
// an email that is not the account's: the token works, but searches by
// author would come back empty.
func (a *App) verifyAccount(cfg *config.Config) error {
	bundle, err := openProfile(cfg, profile.Options{AnyScope: true})
	if err != nil {
		return explainProfile(err)
	}
	ctx, cancel := a.context()
	defer cancel()
	u, err := bundle.Tracker.Me(ctx)
	if err != nil {
		return fmt.Errorf("no se pudo validar las credenciales: %w. Revisa email, token y URL, y vuelve a correr draco init", err)
	}
	if profile.SameAccount(cfg.AtlassianEmail, u) {
		a.ok(fmt.Sprintf("Autenticado como %s", strings.TrimSpace(u.Name+" "+parens(u.Email))))
		return nil
	}
	a.warn(fmt.Sprintf("el email ingresado (%s) no es el de la cuenta autenticada (%s)", cfg.AtlassianEmail, u.Email))
	if !a.confirm("  ¿Usar el email de la cuenta?", true) {
		return nil
	}
	cfg.AtlassianEmail = u.Email
	if err := saveConfig(cfg); err != nil {
		return err
	}
	a.ok("Email corregido a " + u.Email)
	return nil
}

func parens(s string) string {
	if s == "" {
		return ""
	}
	return "(" + s + ")"
}

func initHelp(a *App) {
	a.title("draco init", "Configura credenciales, el space y un proyecto")

	a.section("Uso:")
	a.printf("  draco init\n\n")

	a.section("Si vienes de gtt:")
	a.printf("  Sin configuración de draco, si existe ~/.config/gtt/config.yaml ofrece\n")
	a.printf("  importarla: credenciales, space y proyectos pasan tal cual, y solo se\n")
	a.printf("  verifica la cuenta. La configuración de gtt no se modifica.\n\n")

	a.section("Qué pregunta:")
	a.printf("  1. Email, API token y URL de tu sitio de Atlassian. El token no se muestra al\n")
	a.printf("     escribirlo; se crea en %s\n", tokenURL)
	a.printf("  2. Si no hay perfiles: el space de Confluence de los documentos\n")
	a.printf("     (confluence_space_key; un proyecto puede usar otro)\n")
	a.printf("  3. Valida las credenciales y ofrece corregir el email si no es el de la cuenta\n")
	a.printf("  4. Un proyecto (como draco project add): generar un documento lo necesita\n\n")
	a.printf("  Los perfiles son opcionales: otro destino (una carpeta Markdown) u otra\n")
	a.printf("  plantilla se agregan con draco profile add.\n\n")

	a.printf("  Se guarda en %s con permisos solo para tu usuario. Si ya\n", configPathHint())
	a.printf("  existe, se conservan proyectos, perfiles y ajustes; guardar reescribe el\n")
	a.printf("  archivo y se pierden los comentarios escritos a mano. Las variables\n")
	a.printf("  ATLASSIAN_EMAIL, ATLASSIAN_TOKEN y ATLASSIAN_BASE_URL siguen ganando sobre\n")
	a.printf("  el archivo y nunca se copian a él.\n")
}
