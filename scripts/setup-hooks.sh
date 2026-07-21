#!/bin/bash
# Setup script para instalar Git hooks

set -e

# Detectar directorio raíz del repo
REPO_ROOT=$(git rev-parse --show-toplevel)
HOOKS_DIR="$REPO_ROOT/.githooks"

echo "🔧 Configurando Git hooks..."
echo "   Hooks dir: $HOOKS_DIR"
echo ""

# Configurar Git para usar los hooks desde .githooks/
git config core.hooksPath .githooks

# Hacer ejecutables los hooks
chmod +x "$HOOKS_DIR"/*.* 2>/dev/null || true

echo "✅ Git hooks configurados exitosamente"
echo ""
echo "Hooks instalados:"
ls -1 "$HOOKS_DIR" | sed 's/^/   ✓ /'
echo ""
echo "💡 Tip: Los hooks se ejecutarán automáticamente en los eventos correspondientes"
