package defaults

import "embed"

//go:embed themes syntax plugins config.json keybindings.json help.md
var FS embed.FS
