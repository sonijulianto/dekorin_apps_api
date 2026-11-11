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
for t in landing superadmin client_form invitation_portal template_pn4 template_ta3 template_ta17 template_jawa3 template_jawa2; do
    echo "  -> web/templates/${t}.html"
    curl -s -f -L -o "web/templates/${t}.html" "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/templates/${t}.html?v=$TIMESTAMP"
done

# Bersihkan file template lama yang sudah dinonaktifkan
rm -f web/templates/wedding_invitation.html \
      web/templates/template_floral.html \
      web/templates/template_luxury_grey.html \
      web/templates/template_khitan_green.html \
      web/templates/template_creamy_latte.html \
      web/templates/template_green_leaves.html
rm -rf web/assets/templates/creamy-latte \
       web/assets/templates/floral \
       web/assets/templates/green-leaves \
       web/assets/templates/khitan-green \
       web/assets/templates/luxury-grey

# 4b. Unduh Aset Template Monochrome PN4 (Fonts, Ornamen, Tekstur, Foto Default, Audio)
echo "⬇️  Mengunduh aset template Monochrome PN4..."
mkdir -p web/assets/templates/pn4/fonts web/assets/templates/pn4/photos
for f in ARROW-UP-PN.json akad-pn1-2.png bg-teksture-pn4.png default-song.mp3 flower-pn1.png logo_bca.png logo_bri.png logo_mandiri.png resepsi-pn1.png video-pn1.png; do
    curl -s -f -L -o "web/assets/templates/pn4/${f}" "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/templates/pn4/${f}?v=$TIMESTAMP" || true
done
for fn in 2989Trajan.woff2 ArslanWessam.ttf BrittanySignature.ttf Gistesy.ttf NewYork.ttf Oranienbaum.ttf Saudagar.ttf; do
    curl -s -f -L -o "web/assets/templates/pn4/fonts/${fn}" "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/templates/pn4/fonts/${fn}?v=$TIMESTAMP" || true
done
for n in 1 2 3 4 5 6 7 8 10 12 13 14 15 16 17 18; do
    curl -s -f -L -o "web/assets/templates/pn4/photos/LeniArdi${n}.jpeg" "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/templates/pn4/photos/LeniArdi${n}.jpeg?v=$TIMESTAMP" || true
done

# 4c. Unduh Aset Template Sakura Blossom TA3 (Backgrounds, Pasangan Default, Audio, Foto Galeri Default)
echo "⬇️  Mengunduh aset template Sakura Blossom TA3..."
mkdir -p web/assets/templates/ta3/photos
for f in bg-cover.jpg bg-floral-arch.jpg bg-pink-floral.jpg bride-default.png couple-hero.png couple-welcome.png default-song.mp3 event-icon.png groom-default.png logo_bca.png logo_bni.png logo_bri.png logo_mandiri.png; do
    curl -s -f -L -o "web/assets/templates/ta3/${f}" "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/templates/ta3/${f}?v=$TIMESTAMP" || true
done
for n in 1 2 3 4 5 6; do
    curl -s -f -L -o "web/assets/templates/ta3/photos/gallery-${n}.jpg" "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/templates/ta3/photos/gallery-${n}.jpg?v=$TIMESTAMP" || true
done

# 4d. Unduh Aset Template Ocean Blossom TA17 (Backgrounds, Pasangan Default, Audio, Foto Galeri Default)
echo "⬇️  Mengunduh aset template Ocean Blossom TA17..."
mkdir -p web/assets/templates/ta17/photos
for f in bg-cover.jpg bg-arch.jpg bg-blue-floral.jpg bg-floral-texture.jpg bride-default.png couple-hero.png couple-welcome.png default-song.mp3 event-icon.png groom-default.png logo_bca.png logo_bni.png logo_bri.png logo_mandiri.png; do
    curl -s -f -L -o "web/assets/templates/ta17/${f}" "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/templates/ta17/${f}?v=$TIMESTAMP" || true
done
for n in 1 2 3 4 5 6; do
    curl -s -f -L -o "web/assets/templates/ta17/photos/gallery${n}.jpg" "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/templates/ta17/photos/gallery${n}.jpg?v=$TIMESTAMP" || true
done

# 4e. Unduh Aset Template Art Jawa Coklat 3 (Backgrounds, Video, Gunungan, Wayang GIF, Audio, Foto Default)
echo "⬇️  Mengunduh aset template Art Jawa Coklat 3..."
mkdir -p web/assets/templates/jawa3/photos
for f in bg-closing.jpg bg-couple.jpg bg-cover.jpg bg-event-pill.jpg bg-footer.jpg bg-hero.jpg bg-savedate.jpg bg-texture.jpg default-song.mp3 floral-corner.png gunungan.png logo_bca.png logo_bni.png logo_bri.png logo_mandiri.png water-jawa-coklat-3.mp4 wayang-anim.gif; do
    curl -s -f -L -o "web/assets/templates/jawa3/${f}" "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/templates/jawa3/${f}?v=$TIMESTAMP" || true
done
for pf in bride-default.jpg groom-default.jpg gallery1.jpg gallery2.jpg gallery3.jpg gallery4.jpg gallery5.jpg gallery6.jpg gallery7.jpg gallery8.jpg; do
    curl -s -f -L -o "web/assets/templates/jawa3/photos/${pf}" "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/templates/jawa3/photos/${pf}?v=$TIMESTAMP" || true
done

# 4f. Unduh Aset Template Art Jawa Biru 2 (Backgrounds, Video, Gunungan, Wayang GIF, Font Charlotte, Audio, Foto Default)
echo "⬇️  Mengunduh aset template Art Jawa Biru 2..."
mkdir -p web/assets/templates/jawa2/photos web/assets/templates/jawa2/fonts
for f in bg-closing.jpg bg-floral-blue.jpg bg-hero.jpg bg-savedate.jpg bg-texture.jpg default-song.mp3 floral-corner.png floral-left.png gunungan-biru.png icon-akad.png icon-resepsi.png icon-ucapan.png logo_bca.png logo_bni.png logo_bri.png logo_mandiri.png water-jawa-biru-2.mp4 wayang-anim.gif; do
    curl -s -f -L -o "web/assets/templates/jawa2/${f}" "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/templates/jawa2/${f}?v=$TIMESTAMP" || true
done
curl -s -f -L -o "web/assets/templates/jawa2/fonts/charlotte.woff2" "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/templates/jawa2/fonts/charlotte.woff2?v=$TIMESTAMP" || true
for pf in cover-default.jpg bride-default.jpg groom-default.jpg gallery1.jpg gallery2.jpg gallery3.jpg gallery4.jpg gallery5.jpg gallery6.jpg gallery7.jpg gallery8.jpg; do
    curl -s -f -L -o "web/assets/templates/jawa2/photos/${pf}" "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/templates/jawa2/photos/${pf}?v=$TIMESTAMP" || true
done

# 5. Icons & Branding
curl -s -f -L -o web/assets/icons/google-maps-icon.webp "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/icons/google-maps-icon.webp?v=$TIMESTAMP"
curl -s -f -L -o web/assets/icons/logo_bri.png "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/icons/logo_bri.png?v=$TIMESTAMP" || true
curl -s -f -L -o web/assets/favicon.ico "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/favicon.ico?v=$TIMESTAMP" || true
curl -s -f -L -o web/assets/favicon.png "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/favicon.png?v=$TIMESTAMP" || true
curl -s -f -L -o web/assets/apple-touch-icon.png "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/apple-touch-icon.png?v=$TIMESTAMP" || true
curl -s -f -L -o web/assets/logo_icon.png "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/logo_icon.png?v=$TIMESTAMP" || true
curl -s -f -L -o web/assets/dekorin_logo_square.png "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/dekorin_logo_square.png?v=$TIMESTAMP" || true
curl -s -f -L -o web/assets/dekorin-favicon.png "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/dekorin-favicon.png?v=$TIMESTAMP" || true
curl -s -f -L -o web/assets/dekorin-favicon-32x32.png "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/dekorin-favicon-32x32.png?v=$TIMESTAMP" || true
curl -s -f -L -o web/assets/dekorin-favicon.ico "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/dekorin-favicon.ico?v=$TIMESTAMP" || true
curl -s -f -L -o web/assets/dekorin-apple-touch-icon.png "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/dekorin-apple-touch-icon.png?v=$TIMESTAMP" || true
curl -s -f -L -o web/assets/icons/pwa-maskable-192.png "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/icons/pwa-maskable-192.png?v=$TIMESTAMP" || true
curl -s -f -L -o web/assets/icons/pwa-maskable-512.png "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/icons/pwa-maskable-512.png?v=$TIMESTAMP" || true
curl -s -f -L -o web/assets/superadmin-manifest.json "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/superadmin-manifest.json?v=$TIMESTAMP" || true
curl -s -f -L -o web/assets/superadmin-sw.js "https://raw.githubusercontent.com/sonijulianto/dekorin_apps_api/main/web/assets/superadmin-sw.js?v=$TIMESTAMP" || true

# 6. Konfigurasi Nginx: Limit upload 150M & HTTP Security Headers
if command -v nginx >/dev/null 2>&1 && [ -d /etc/nginx ]; then
    echo "⚙️  Mengonfigurasi client_max_body_size 150M & Security Headers di Nginx..."
    if ! grep -q "client_max_body_size" /etc/nginx/nginx.conf; then
        sudo sed -i '/http {/a \    client_max_body_size 150M;' /etc/nginx/nginx.conf 2>/dev/null || true
    else
        sudo sed -i 's/client_max_body_size .*/client_max_body_size 150M;/g' /etc/nginx/nginx.conf 2>/dev/null || true
    fi
    for f in /etc/nginx/sites-available/* /etc/nginx/sites-enabled/* /etc/nginx/conf.d/*; do
        if [ -f "$f" ]; then
            sudo sed -i 's/client_max_body_size [0-9]\+[kKmMgG]\+;/client_max_body_size 150M;/g' "$f" 2>/dev/null || true
            if ! grep -q "X-Content-Type-Options" "$f"; then
                sudo sed -i '/server_name/a \    add_header X-Content-Type-Options "nosniff" always;\n    add_header X-Frame-Options "SAMEORIGIN" always;\n    add_header X-XSS-Protection "1; mode=block" always;\n    add_header Referrer-Policy "strict-origin-when-cross-origin" always;' "$f" 2>/dev/null || true
            fi
        fi
    done
    # 6b. Pastikan virtual host subdomain undangan.dekorin.my.id (atau wildcard *.dekorin.my.id) terdaftar di Nginx
    if ! grep -rqE "(undangan\.dekorin\.my\.id|\*\.dekorin\.my\.id)" /etc/nginx/sites-enabled/ /etc/nginx/conf.d/ 2>/dev/null; then
        echo "🌐 Menambahkan konfigurasi Nginx untuk subdomain undangan.dekorin.my.id..."
        UNDANGAN_CONF="/etc/nginx/sites-available/undangan-dekorin"
        if [ ! -d "/etc/nginx/sites-available" ]; then
            UNDANGAN_CONF="/etc/nginx/conf.d/undangan-dekorin.conf"
        fi
        SSL_BLOCK=""
        if [ -f "/etc/letsencrypt/live/undangan.dekorin.my.id/fullchain.pem" ]; then
            SSL_BLOCK="    listen 443 ssl;\n    ssl_certificate /etc/letsencrypt/live/undangan.dekorin.my.id/fullchain.pem;\n    ssl_certificate_key /etc/letsencrypt/live/undangan.dekorin.my.id/privkey.pem;"
        elif [ -f "/etc/letsencrypt/live/dekorin.my.id/fullchain.pem" ]; then
            SSL_BLOCK="    listen 443 ssl;\n    ssl_certificate /etc/letsencrypt/live/dekorin.my.id/fullchain.pem;\n    ssl_certificate_key /etc/letsencrypt/live/dekorin.my.id/privkey.pem;"
        fi
        sudo bash -c "cat > $UNDANGAN_CONF" <<EOF
server {
    listen 80;
$(printf "%b" "$SSL_BLOCK")
    server_name undangan.dekorin.my.id;
    client_max_body_size 150M;

    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_cache_bypass \$http_upgrade;
    }
}
EOF
        if [ -d "/etc/nginx/sites-enabled" ] && [ "$UNDANGAN_CONF" = "/etc/nginx/sites-available/undangan-dekorin" ]; then
            sudo ln -sf "$UNDANGAN_CONF" /etc/nginx/sites-enabled/undangan-dekorin
        fi
        if ! sudo nginx -t >/dev/null 2>&1; then
            echo "⚠️  Validasi Nginx gagal untuk conf baru, membatalkan conf tambahan agar server tetap aman..."
            sudo rm -f /etc/nginx/sites-enabled/undangan-dekorin "$UNDANGAN_CONF"
        fi
    fi

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
echo "🌐 Silakan buka: https://dekorin.my.id/superadmin"
echo "========================================"
