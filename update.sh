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

echo "⬇️  Mengunduh web templates & assets..."
mkdir -p web/templates
mkdir -p web/assets/icons

# Templates HTML
for t in superadmin client_form wedding_invitation template_floral template_luxury_grey template_khitan_green template_creamy_latte template_green_leaves; do
    echo "  -> web/templates/${t}.html"
    curl -s -f -L -o "web/templates/${t}.html" "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/templates/${t}.html?v=$TIMESTAMP"
done

# Icons
curl -s -f -L -o web/assets/icons/google-maps-icon.webp "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/icons/google-maps-icon.webp?v=$TIMESTAMP"

# Template Assets Bundle
echo "⬇️  Mengunduh & mengekstrak aset template undangan..."
curl -s -f -L -o web/templates_bundle.tar.gz "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/templates_bundle.tar.gz?v=$TIMESTAMP"
tar -xzf web/templates_bundle.tar.gz -C web/
rm -f web/templates_bundle.tar.gz

echo "🔄 Me-restart service dekorin..."
sudo systemctl restart dekorin

echo "========================================"
echo "✅ UPDATE BERHASIL & SUKSES!"
echo "🌐 Silakan buka: https://103.150.191.59.sslip.io/superadmin"
echo "========================================"
