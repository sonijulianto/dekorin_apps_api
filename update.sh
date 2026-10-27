#!/bin/bash
set -e
echo "========================================"
echo "🚀 Memulai Update Dekorin Backend & Web"
echo "========================================"

TARGET_DIR="/home/sonijulianto/dekorin"
if [ ! -d "$TARGET_DIR" ]; then
    TARGET_DIR="$HOME/dekorin"
fi

if [ ! -d "$TARGET_DIR" ]; then
    # Jika direktori di atas tidak ada, cari lokasi file dekorin-api yang sedang berjalan
    RUNNING_PATH=$(pgrep -a dekorin-api | head -n 1 | awk '{print $2}')
    if [ -n "$RUNNING_PATH" ] && [ -f "$RUNNING_PATH" ]; then
        TARGET_DIR=$(dirname "$RUNNING_PATH")
    fi
fi

mkdir -p "$TARGET_DIR"
cd "$TARGET_DIR"
echo "📁 Direktori Kerja: $(pwd)"

TIMESTAMP=$(date +%s)

# 1. Unduh Binary Linux Terbaru
echo "⬇️  Mengunduh binary Linux terbaru (dekorin-api-linux)..."
curl -f -L -o dekorin-api.new "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/dekorin-api-linux?v=$TIMESTAMP"
chmod +x dekorin-api.new
mv -f dekorin-api.new dekorin-api

# 2. Siapkan Folder Web & Uploads
mkdir -p web/templates
mkdir -p web/assets/icons
mkdir -p uploads/apks

# 3. Unduh & Ekstrak Bundle Aset Terlebih Dahulu
echo "⬇️  Mengunduh & mengekstrak bundle aset template..."
if curl -s -f -L -o web/templates_bundle.tar.gz "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/templates_bundle.tar.gz?v=$TIMESTAMP"; then
    tar -xzf web/templates_bundle.tar.gz -C web/ 2>/dev/null || true
    rm -f web/templates_bundle.tar.gz
fi

# 4. Unduh File Template HTML Terbaru (Dijalankan SETELAH tar agar tidak tertimpa)
echo "⬇️  Mengunduh template HTML terbaru..."
for t in superadmin client_form wedding_invitation template_floral template_luxury_grey template_khitan_green template_creamy_latte template_green_leaves; do
    echo "  -> web/templates/${t}.html"
    curl -s -f -L -o "web/templates/${t}.html" "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/templates/${t}.html?v=$TIMESTAMP"
done

# 5. Icons
curl -s -f -L -o web/assets/icons/google-maps-icon.webp "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/icons/google-maps-icon.webp?v=$TIMESTAMP"

# 6. Konfigurasi Nginx: Naikkan limit upload body agar file APK besar tidak terkena 413
if command -v nginx >/dev/null 2>&1 && [ -d /etc/nginx ]; then
    echo "⚙️  Mengonfigurasi client_max_body_size 150M di Nginx..."
    if ! grep -q "client_max_body_size" /etc/nginx/nginx.conf; then
        sudo sed -i '/http {/a \    client_max_body_size 150M;' /etc/nginx/nginx.conf 2>/dev/null || true
    else
        sudo sed -i 's/client_max_body_size .*/client_max_body_size 150M;/g' /etc/nginx/nginx.conf 2>/dev/null || true
    fi
    for f in /etc/nginx/sites-available/* /etc/nginx/sites-enabled/* /etc/nginx/conf.d/*; do
        if [ -f "$f" ]; then
            sudo sed -i 's/client_max_body_size [0-9]\+[kKmMgG]\+;/client_max_body_size 150M;/g' "$f" 2>/dev/null || true
        fi
    done
    sudo nginx -t >/dev/null 2>&1 && sudo systemctl reload nginx 2>/dev/null || true
fi

# 7. Restart Service Dekorin
echo "🔄 Me-restart service backend..."
if command -v systemctl >/dev/null 2>&1; then
    sudo systemctl restart dekorin 2>/dev/null || sudo systemctl restart dekorin-api 2>/dev/null || systemctl restart dekorin 2>/dev/null || true
fi

# Cek status proses
echo "📊 Memeriksa status proses dekorin-api..."
sleep 2
if pgrep -f dekorin-api >/dev/null; then
    echo "🟢 Service dekorin-api berjalan dengan PID: $(pgrep -f dekorin-api | head -n 1)"
else
    echo "⚠️  Service belum aktif melalui systemctl, mencoba menjalankan ulang..."
    sudo systemctl restart dekorin || true
fi

echo "========================================"
echo "✅ UPDATE BERHASIL & SELESAI!"
echo "🌐 Silakan buka: https://103.150.191.59.sslip.io/superadmin"
echo "========================================"
