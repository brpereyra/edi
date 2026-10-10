package defaults

import "embed"

//go:embed themes syntax plugins config.json keybindings.json
var FS embed.FS
