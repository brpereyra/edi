package main

import (
	"flag"
	"fmt"
	"os"

	"edi/internal/buffer"
	"edi/internal/config"
	"edi/internal/editor"

	"github.com/gdamore/tcell/v2"
)

const version = "0.1.0"

func printHelp() {
	fmt.Printf(`edi v%s - Editor de texto en consola rápido, personalizable y portable en Go

USO:
  edi [opciones] [archivo o carpeta]

EJEMPLOS:
  edi main.go           Abre o crea un archivo
  edi .                 Abre la carpeta actual en el Explorador de Archivos lateral
  edi /ruta/a/proyecto  Abre un proyecto específico en el Explorador
  edi --init            Inicializa la configuración en ~/.edi-config/

OPCIONES:
  -h, --help            Muestra esta ayuda detallada y sale
  -v, --version         Muestra la versión de edi
  --init                Crea la carpeta ~/.edi-config/ con temas y sintaxis base
  --config-dir          Muestra la ruta de la configuración activa

ATAJOS DE TECLADO CLAVE:
  Ctrl+P                Abrir la Paleta de Comandos interactiva
  Ctrl+O                Buscador rápido de archivos en el proyecto (Fuzzy Finder)
  Ctrl+B                Alternar panel del Explorador de Archivos lateral
  Ctrl+S / F2           Guardar archivo (pregunta nombre en terminal si es nuevo)
  Ctrl+N                Nueva pestaña en blanco (en el explorador: tecla 'a' o 'n')
  Ctrl+W                Cerrar pestaña activa
  Ctrl+Tab              Alternar entre pestañas abiertas
  Shift + Flechas       Seleccionar texto (multi-línea)
  Ctrl+C / Ctrl+X       Copiar / Cortar selección al portapapeles
  Ctrl+V                Pegar desde el portapapeles
  Ctrl+F                Buscar texto en el archivo
  Ctrl+G                Ir a una línea específica
  Alt+Up / Alt+Down     Mover línea o selección arriba / abajo
  Ctrl+Z / Ctrl+Y       Deshacer / Rehacer
  F1                    Abrir la documentación y guía completa dentro del editor
  Ctrl+Q                Salir del editor

PERSONALIZACIÓN (~/.edi-config/):
  - Temas:              ~/.edi-config/themes/*.json (Dracula, Monokai, Nord...)
  - Sintaxis:           ~/.edi-config/syntax/*.json (agrega lenguajes con solo crear un archivo)
  - Atajos:             ~/.edi-config/keybindings.json
  - Preferencias:       ~/.edi-config/config.json
  - Plugins:            ~/.edi-config/plugins/*.json (gofmt, prettier, etc.)

DOCUMENTACIÓN INTERNA:
  Dentro de edi, presiona F1 o abre la paleta con Ctrl+P y selecciona 'Documentación y Ayuda'.
`, version)
}

func main() {
	flag.Usage = printHelp

	initFlag := flag.Bool("init", false, "Inicializa la carpeta ~/.edi-config con plantillas predeterminadas")
	var versionFlag bool
	flag.BoolVar(&versionFlag, "version", false, "Muestra la versión de edi")
	flag.BoolVar(&versionFlag, "v", false, "Muestra la versión de edi")

	configFlag := flag.Bool("config-dir", false, "Muestra la ruta de configuración activa")

	var helpFlag bool
	flag.BoolVar(&helpFlag, "help", false, "Muestra la ayuda detallada")
	flag.BoolVar(&helpFlag, "h", false, "Muestra la ayuda detallada")

	flag.Parse()

	if helpFlag {
		printHelp()
		os.Exit(0)
	}

	if versionFlag {
		fmt.Printf("edi v%s (Editor de texto rápido y portable en Go)\n", version)
		os.Exit(0)
	}

	if *initFlag {
		dir, err := config.InitUserConfigDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error inicializando configuración: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Configuración inicializada exitosamente en:\n  %s\n\nPuedes agregar temas en %s/themes/ y sintaxis en %s/syntax/\n", dir, dir, dir)
		os.Exit(0)
	}

	if *configFlag {
		dir := config.FindConfigDir()
		if dir == "" {
			fmt.Println("Usando recursos embebidos por defecto (no se encontró carpeta ~/.edi-config ni local).")
			userDir, _ := config.GetUserConfigDir()
			fmt.Printf("Para crear tu configuración personalizada ejecuta: edi --init\n(Ruta: %s)\n", userDir)
		} else {
			fmt.Printf("Carpeta de configuración activa: %s\n", dir)
		}
		os.Exit(0)
	}

	// Archivo o carpeta a abrir
	args := flag.Args()
	var buf *buffer.Buffer
	var err error
	openFolder := false
	folderPath := ""

	if len(args) > 0 {
		stat, statErr := os.Stat(args[0])
		if statErr == nil && stat.IsDir() {
			openFolder = true
			folderPath = args[0]
			buf = buffer.NewEmptyBuffer()
		} else {
			buf, err = buffer.NewBufferFromFile(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error cargando archivo %s: %v\n", args[0], err)
				os.Exit(1)
			}
		}
	} else {
		buf = buffer.NewEmptyBuffer()
	}

	// Cargar configuración
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Advertencia: error cargando configuración: %v\n", err)
	}

	// Inicializar pantalla de terminal tcell
	screen, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error inicializando terminal: %v\n", err)
		os.Exit(1)
	}

	if err := screen.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Error preparando pantalla: %v\n", err)
		os.Exit(1)
	}

	screen.EnableMouse()
	screen.EnablePaste()

	ed, err := editor.NewEditor(screen, buf, cfg)
	if err != nil {
		screen.Fini()
		fmt.Fprintf(os.Stderr, "Error configurando editor: %v\n", err)
		os.Exit(1)
	}

	if openFolder {
		ed.OpenFolder(folderPath)
	}

	if err := ed.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error durante ejecución: %v\n", err)
		os.Exit(1)
	}
}
