#!/bin/bash
set -e
echo "========================================"
echo "🚀 Memulai Update Dekorin Backend & Web"
echo "========================================"

TARGET_DIR="/home/sonijulianto/dekorin"
if [ ! -d "$TARGET_DIR" ]; then
    TARGET_DIR="$HOME/dekorin"
fi

cd "$TARGET_DIR"
echo "📁 Direktori: $(pwd)"

TIMESTAMP=$(date +%s)
echo "⬇️  Mengunduh binary Linux terbaru..."
curl -f -L -o dekorin-api.new "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/dekorin-api-linux?v=$TIMESTAMP"
chmod +x dekorin-api.new
mv -f dekorin-api.new dekorin-api

echo "⬇️  Mengunduh web template & assets terbaru..."
mkdir -p web/templates
mkdir -p web/assets/icons
curl -f -L -o web/templates/superadmin.html "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/templates/superadmin.html?v=$TIMESTAMP"
curl -f -L -o web/templates/client_form.html "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/templates/client_form.html?v=$TIMESTAMP"
curl -f -L -o web/assets/icons/google-maps-icon.webp "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/icons/google-maps-icon.webp?v=$TIMESTAMP"

echo "🔄 Me-restart service dekorin..."
sudo systemctl restart dekorin

echo "========================================"
echo "✅ UPDATE BERHASIL & SUKSES!"
echo "🌐 Silakan buka: https://103.150.191.59.sslip.io/superadmin"
echo "========================================"
