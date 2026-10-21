# Instala draco en Windows: descarga el binario del último release, verifica su
# SHA-256 contra checksums.txt y lo copia a %LOCALAPPDATA%\Programs\draco.
#   iwr -useb https://raw.githubusercontent.com/kaelence-01/draco-hq/main/scripts/install.ps1 | iex
# $env:DRACO_VERSION = "vX.Y.Z" instala esa versión.

& {
	$ErrorActionPreference = "Stop"
	$repo = "kaelence-01/draco-hq"
	$asset = "draco-windows-amd64.exe"
	$dir = Join-Path $env:LOCALAPPDATA "Programs\draco"
	$exe = Join-Path $dir "draco.exe"

	if (-not [Environment]::Is64BitOperatingSystem) { throw "draco se publica solo para Windows de 64 bits" }
	if ($env:DRACO_VERSION) {
		$base = "https://github.com/$repo/releases/download/$($env:DRACO_VERSION)"
	} else {
		$base = "https://github.com/$repo/releases/latest/download"
	}

	$tmp = Join-Path ([IO.Path]::GetTempPath()) ("draco-" + [Guid]::NewGuid())
	New-Item -ItemType Directory -Force $tmp | Out-Null
	try {
		[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
		Write-Host "Descargando $asset..."
		Invoke-WebRequest -UseBasicParsing "$base/$asset" -OutFile (Join-Path $tmp $asset)
		Invoke-WebRequest -UseBasicParsing "$base/checksums.txt" -OutFile (Join-Path $tmp "checksums.txt")

		$want = $null
		foreach ($line in Get-Content (Join-Path $tmp "checksums.txt")) {
			$parts = $line -split '\s+'
			if ($parts.Count -ge 2 -and $parts[1] -eq $asset) { $want = $parts[0].ToLower() }
		}
		if (-not $want) { throw "checksums.txt no trae $asset" }
		$got = (Get-FileHash (Join-Path $tmp $asset) -Algorithm SHA256).Hash.ToLower()
		if ($got -ne $want) { throw "el SHA-256 no coincide con checksums.txt: no se instaló nada" }
		Write-Host "OK: SHA-256 verificado"

		New-Item -ItemType Directory -Force $dir | Out-Null
		# A running draco.exe cannot be overwritten: it is set aside as .old,
		# which draco removes on its next run (as draco update does).
		if (Test-Path $exe) {
			Remove-Item "$exe.old" -Force -ErrorAction SilentlyContinue
			Move-Item $exe "$exe.old" -Force
		}
		Move-Item (Join-Path $tmp $asset) $exe -Force
		Write-Host "OK: Instalado en $exe"
	} finally {
		Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue
	}

	# The user PATH keeps its registry type: rewriting a REG_EXPAND_SZ value as
	# REG_SZ would freeze the %VAR% entries of other tools.
	$key = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey("Environment", $true)
	try {
		$current = [string]$key.GetValue("Path", "", [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
		$kind = [Microsoft.Win32.RegistryValueKind]::ExpandString
		if ($null -ne $key.GetValue("Path")) { $kind = $key.GetValueKind("Path") }
		$entries = $current -split ';' | ForEach-Object { $_.Trim().Trim('"').TrimEnd('\').ToLower() }
		if ($entries -notcontains $dir.TrimEnd('\').ToLower()) {
			$value = $current.TrimEnd('; ')
			if ($value) { $value += ";$dir" } else { $value = $dir }
			$key.SetValue("Path", $value, $kind)
			Write-Host "OK: Carpeta agregada al PATH del usuario (las terminales nuevas la verán)"
		}
	} finally {
		$key.Close()
	}
	if (($env:Path -split ';') -notcontains $dir) { $env:Path += ";$dir" }

	& $exe version
	Write-Host ""
	Write-Host "Siguiente paso: draco init (o draco, y Configuración)"
}
