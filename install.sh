#!/usr/bin/env bash
# ==============================================================================
#  edi installer - Fast, portable TUI text editor in Go
#  Repository: https://github.com/brpereyra/edi
# ==============================================================================
set -euo pipefail

REPO="brpereyra/edi"
BINARY_NAME="edi"

# Colors for terminal output
RED=$(printf '\033[0;31m')
GREEN=$(printf '\033[0;32m')
CYAN=$(printf '\033[0;36m')
YELLOW=$(printf '\033[1;33m')
BOLD=$(printf '\033[1m')
NC=$(printf '\033[0m') # No Color

info() {
    printf "${CYAN}[info]${NC} %s\n" "$1"
}

success() {
    printf "${GREEN}[ok]${NC} %s\n" "$1"
}

warn() {
    printf "${YELLOW}[warn]${NC} %s\n" "$1"
}

error() {
    printf "${RED}[error]${NC} %s\n" "$1" >&2
    exit 1
}

# 1. Detect Operating System
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
    linux)  TARGET_OS="linux" ;;
    darwin) TARGET_OS="darwin" ;;
    *)
        error "Sistema operativo no soportado por este script: $OS. edi soporta Linux y macOS."
        ;;
esac

# 2. Detect Architecture
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64|amd64) TARGET_ARCH="amd64" ;;
    arm64|aarch64) TARGET_ARCH="arm64" ;;
    *)
        error "Arquitectura no soportada: $ARCH. Se requiere amd64 o arm64."
        ;;
esac

TARGET_BINARY="${BINARY_NAME}-${TARGET_OS}-${TARGET_ARCH}"
info "Detectado sistema: ${BOLD}${TARGET_OS} (${TARGET_ARCH})${NC}"

# 3. Determine Installation Directory
# Priority:
#  a) $EDI_INSTALL_DIR if specified
#  b) /usr/local/bin if writable without sudo
#  c) ~/.local/bin (standard user binary path)
if [ -n "${EDI_INSTALL_DIR:-}" ]; then
    INSTALL_DIR="$EDI_INSTALL_DIR"
elif [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
else
    INSTALL_DIR="${HOME}/.local/bin"
fi

mkdir -p "$INSTALL_DIR"

TEMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t 'edi-install')"
trap 'rm -rf "$TEMP_DIR"' EXIT

INSTALLED_PATH="${INSTALL_DIR}/${BINARY_NAME}"
DOWNLOAD_SUCCESS=false

# 4. Try downloading precompiled binary from GitHub Releases
LATEST_URL="https://github.com/${REPO}/releases/latest/download/${TARGET_BINARY}"
info "Buscando binario precompilado en GitHub Releases..."

if curl -f -L -s -S "$LATEST_URL" -o "${TEMP_DIR}/${BINARY_NAME}" 2>/dev/null; then
    chmod +x "${TEMP_DIR}/${BINARY_NAME}"
    mv "${TEMP_DIR}/${BINARY_NAME}" "$INSTALLED_PATH"
    DOWNLOAD_SUCCESS=true
    success "Binario descargado desde GitHub Releases e instalado en: ${BOLD}${INSTALLED_PATH}${NC}"
fi

# 5. Fallback: If no release binary yet, build from source if Go is installed
if [ "$DOWNLOAD_SUCCESS" = false ]; then
    warn "No se encontró release binario precompilado todavía en GitHub Releases."
    if command -v go >/dev/null 2>&1; then
        info "Detectado compilador Go local ($(go version | awk '{print $3}')). Compilando desde el código fuente..."
        if [ -d "cmd/edi" ]; then
            CGO_ENABLED=0 go build -ldflags="-s -w" -o "${TEMP_DIR}/${BINARY_NAME}" ./cmd/edi
        else
            git clone --depth 1 "https://github.com/${REPO}.git" "${TEMP_DIR}/source" >/dev/null 2>&1 || {
                error "No se pudo clonar el repositorio para compilar."
            }
            (
                cd "${TEMP_DIR}/source"
                CGO_ENABLED=0 go build -ldflags="-s -w" -o "${TEMP_DIR}/${BINARY_NAME}" ./cmd/edi
            )
        fi
        chmod +x "${TEMP_DIR}/${BINARY_NAME}"
        mv "${TEMP_DIR}/${BINARY_NAME}" "$INSTALLED_PATH"
        success "Compilado e instalado exitosamente en: ${BOLD}${INSTALLED_PATH}${NC}"
    else
        error "No hay binarios precompilados en releases y no se encontró 'go' instalado para compilar desde el código fuente.\nPuedes instalar Go o publicar un release en https://github.com/${REPO}/releases."
    fi
fi

# 6. Verify installation
if [ -x "$INSTALLED_PATH" ]; then
    VERSION="$("$INSTALLED_PATH" --version 2>/dev/null || echo "v0.1.0")"
    success "Instalación completada: ${BOLD}${VERSION}${NC}"
else
    error "Error verificando el binario instalado en $INSTALLED_PATH"
fi

# 7. Check if INSTALL_DIR is in PATH
case ":$PATH:" in
    *":${INSTALL_DIR}:"*)
        PATH_OK=true
        ;;
    *)
        PATH_OK=false
        ;;
esac

if [ "$PATH_OK" = false ]; then
    warn "El directorio '${INSTALL_DIR}' no está en tu variable \$PATH."

    CURRENT_SHELL="$(basename "${SHELL:-bash}")"
    RC_PATH=""
    RC_NAME=""
    EXPORT_CMD=""

    case "$CURRENT_SHELL" in
        zsh)
            RC_PATH="${HOME}/.zshrc"
            RC_NAME="~/.zshrc"
            EXPORT_CMD="export PATH=\"\$PATH:${INSTALL_DIR}\""
            ;;
        bash)
            if [ -f "${HOME}/.bashrc" ]; then
                RC_PATH="${HOME}/.bashrc"
                RC_NAME="~/.bashrc"
            elif [ -f "${HOME}/.bash_profile" ]; then
                RC_PATH="${HOME}/.bash_profile"
                RC_NAME="~/.bash_profile"
            else
                RC_PATH="${HOME}/.bashrc"
                RC_NAME="~/.bashrc"
            fi
            EXPORT_CMD="export PATH=\"\$PATH:${INSTALL_DIR}\""
            ;;
        fish)
            RC_PATH="${HOME}/.config/fish/config.fish"
            RC_NAME="~/.config/fish/config.fish"
            EXPORT_CMD="set -gx PATH \$PATH ${INSTALL_DIR}"
            ;;
        *)
            RC_NAME="tu archivo de configuración de shell"
            EXPORT_CMD="export PATH=\"\$PATH:${INSTALL_DIR}\""
            ;;
    esac

    ADDED_TO_RC=false
    if [ -n "$RC_PATH" ]; then
        mkdir -p "$(dirname "$RC_PATH")" 2>/dev/null || true
        touch "$RC_PATH" 2>/dev/null || true

        # Comprobar si ya existe la ruta activa en el archivo
        if ! grep -q "${INSTALL_DIR}" "$RC_PATH" 2>/dev/null || grep -q "#.*${INSTALL_DIR}" "$RC_PATH" 2>/dev/null; then
            printf "\n# edi text editor\n%s\n" "$EXPORT_CMD" >> "$RC_PATH"
            ADDED_TO_RC=true
            success "Se agregó automáticamente a ${BOLD}${RC_NAME}${NC}"
            info "Para activarlo en tu sesión actual de terminal ejecuta:"
            printf "  ${BOLD}source %s${NC}\n" "$RC_NAME"
            printf "  (o directamente: ${BOLD}%s${NC})\n\n" "$EXPORT_CMD"
        fi
    fi

    if [ "$ADDED_TO_RC" = false ]; then
        printf "\nComando para agregarlo o activarlo manualmente:\n"
        printf "  ${BOLD}echo '%s' >> %s && source %s${NC}\n\n" "$EXPORT_CMD" "$RC_NAME" "$RC_NAME"
    fi
fi

# 8. Setup default configs (~/.edi-config/)
if [ ! -d "${HOME}/.edi-config" ]; then
    info "Inicializando configuración base en ~/.edi-config/..."
    "$INSTALLED_PATH" --init >/dev/null 2>&1 || true
    success "Configuración creada en ~/.edi-config/ (temas, sintaxis y atajos incluidos)."
fi

printf "\n${GREEN}${BOLD}¡edi está listo para usar!${NC}\n"
printf "Ejecuta:  ${CYAN}edi .${NC}         para abrir la carpeta actual\n"
printf "          ${CYAN}edi main.go${NC}   para editar un archivo\n"
printf "          ${CYAN}edi --help${NC}    para ver todos los atajos y opciones\n\n"
