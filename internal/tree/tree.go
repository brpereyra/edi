package tree

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TreeNode representa un nodo del árbol de archivos o directorios.
type TreeNode struct {
	Path     string
	Name     string
	IsDir    bool
	Expanded bool
	Depth    int
	Children []*TreeNode
}

// FileTree gestiona el estado y navegación del explorador de archivos.
type FileTree struct {
	RootPath    string
	RootNode    *TreeNode
	VisibleRows []*TreeNode
	SelectedIdx int
	ScrollY     int
}

// NewFileTree inicializa un nuevo árbol a partir del directorio raíz dado.
func NewFileTree(rootPath string) (*FileTree, error) {
	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		absPath = rootPath
	}

	ft := &FileTree{
		RootPath: absPath,
		RootNode: &TreeNode{
			Path:     absPath,
			Name:     filepath.Base(absPath),
			IsDir:    true,
			Expanded: true,
			Depth:    0,
		},
	}

	ft.loadChildren(ft.RootNode)
	ft.RebuildVisibleRows()
	return ft, nil
}

func (ft *FileTree) loadChildren(node *TreeNode) {
	if !node.IsDir {
		return
	}

	entries, err := os.ReadDir(node.Path)
	if err != nil {
		return
	}

	var dirs []*TreeNode
	var files []*TreeNode

	ignoreDirs := map[string]bool{
		".git":         true,
		"node_modules": true,
		"vendor":       true,
		"dist":         true,
		"build":        true,
	}

	for _, entry := range entries {
		name := entry.Name()
		if ignoreDirs[name] {
			continue
		}
		// Ignorar temporales y archivos ocultos irrelevantes
		if strings.HasPrefix(name, ".") && name != ".edi-config" && name != ".gitignore" {
			continue
		}

		childPath := filepath.Join(node.Path, name)
		isDir := entry.IsDir()

		childNode := &TreeNode{
			Path:     childPath,
			Name:     name,
			IsDir:    isDir,
			Expanded: false,
			Depth:    node.Depth + 1,
		}

		if isDir {
			dirs = append(dirs, childNode)
		} else {
			files = append(files, childNode)
		}
	}

	// Ordenar alfabéticamente (directorios primero)
	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name)
	})
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	node.Children = append(dirs, files...)
}

// RebuildVisibleRows aplana el árbol jerárquico según qué carpetas están expandidas.
func (ft *FileTree) RebuildVisibleRows() {
	var rows []*TreeNode

	var traverse func(node *TreeNode)
	traverse = func(node *TreeNode) {
		for _, child := range node.Children {
			rows = append(rows, child)
			if child.IsDir && child.Expanded {
				traverse(child)
			}
		}
	}

	traverse(ft.RootNode)
	ft.VisibleRows = rows

	if ft.SelectedIdx >= len(ft.VisibleRows) {
		ft.SelectedIdx = len(ft.VisibleRows) - 1
	}
	if ft.SelectedIdx < 0 && len(ft.VisibleRows) > 0 {
		ft.SelectedIdx = 0
	}
}

// MoveDown mueve la selección hacia abajo.
func (ft *FileTree) MoveDown() {
	if ft.SelectedIdx < len(ft.VisibleRows)-1 {
		ft.SelectedIdx++
	}
}

// MoveUp mueve la selección hacia arriba.
func (ft *FileTree) MoveUp() {
	if ft.SelectedIdx > 0 {
		ft.SelectedIdx--
	}
}

// GetSelected retorna el nodo actualmente seleccionado o nil.
func (ft *FileTree) GetSelected() *TreeNode {
	if ft.SelectedIdx >= 0 && ft.SelectedIdx < len(ft.VisibleRows) {
		return ft.VisibleRows[ft.SelectedIdx]
	}
	return nil
}

// ToggleCurrent expande o contrae la carpeta seleccionada.
func (ft *FileTree) ToggleCurrent() {
	selected := ft.GetSelected()
	if selected == nil || !selected.IsDir {
		return
	}

	selected.Expanded = !selected.Expanded
	if selected.Expanded && len(selected.Children) == 0 {
		ft.loadChildren(selected)
	}

	ft.RebuildVisibleRows()
}

// ExpandCurrent expande la carpeta seleccionada si no lo está.
func (ft *FileTree) ExpandCurrent() {
	selected := ft.GetSelected()
	if selected == nil || !selected.IsDir {
		return
	}

	if !selected.Expanded {
		selected.Expanded = true
		if len(selected.Children) == 0 {
			ft.loadChildren(selected)
		}
		ft.RebuildVisibleRows()
	}
}

// CollapseCurrent contrae la carpeta seleccionada.
func (ft *FileTree) CollapseCurrent() {
	selected := ft.GetSelected()
	if selected == nil {
		return
	}

	if selected.IsDir && selected.Expanded {
		selected.Expanded = false
		ft.RebuildVisibleRows()
	}
}

// Refresh recarga los contenidos del árbol preservando los directorios que estaban expandidos.
func (ft *FileTree) Refresh() {
	if ft == nil || ft.RootNode == nil {
		return
	}

	expandedMap := make(map[string]bool)
	var collectExpanded func(node *TreeNode)
	collectExpanded = func(node *TreeNode) {
		if node.IsDir && node.Expanded {
			expandedMap[node.Path] = true
			for _, child := range node.Children {
				collectExpanded(child)
			}
		}
	}
	collectExpanded(ft.RootNode)

	var reloadNode func(node *TreeNode)
	reloadNode = func(node *TreeNode) {
		if !node.IsDir {
			return
		}
		node.Children = nil
		ft.loadChildren(node)
		for _, child := range node.Children {
			if child.IsDir && expandedMap[child.Path] {
				child.Expanded = true
				reloadNode(child)
			}
		}
	}

	reloadNode(ft.RootNode)
	ft.RebuildVisibleRows()
}

