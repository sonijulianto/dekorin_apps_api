# Dekorin Project Guidelines & Rules

## 1. Asset & Icon Priority (Flutter Mobile App)
- **Utamakan Aset Lokal**: Setiap kali membuat screen, dialog, bottom sheet, atau widget baru di `dekorin_apps`, **selalu periksa dan utamakan mengambil icon dan gambar dari folder `assets/`** (`assets/icons/`, `assets/images/`, dll.) dibandingkan menggunakan icon Material bawaan (`Icons.*`).
- **Aset Khusus yang Wajib Digunakan**:
  - **Google Maps**: Selalu gunakan `Image.asset('assets/icons/google-maps-icon.webp', width: ..., height: ...)` untuk segala hal terkait peta, lokasi, dan Maps URL (jangan gunakan `Icons.map_outlined` atau `Icons.location_on`).
  - **WhatsApp**: Gunakan `assets/icons/wa.svg` jika tersedia untuk tombol kontak chat WhatsApp.
- **Pengecekan Folder Assets**: Sebelum memilih icon untuk fitur baru (logo, pembayaran, media sosial, navigasi, dll.), selalu cek folder `dekorin_apps/assets/` terlebih dahulu untuk melihat apakah aset kustom sudah disediakan oleh user.
- **Pendaftaran di Pubspec**: Pastikan setiap path/sub-folder aset baru di `assets/` sudah terdaftar pada bagian `flutter: assets:` di `dekorin_apps/pubspec.yaml`.

## 2. Design System & Palette Konsisten
- Gunakan identitas visual Dekorin:
  - **Primary Gold**: `AppTheme.primaryGold` / `AppColors.primaryGold` (`#D4A574`)
  - **Secondary Gold**: `AppColors.secondaryGold` (`#C59563`)
  - **Gold Light**: `AppColors.goldLight` (`#F9F3EC`)
  - **Text Dark**: `AppColors.textDark` (`#2C2C2C`)
  - **Typography**: Gunakan `AppTypography` yang berbasis font Poppins / Plus Jakarta Sans.


## 3. Typography Wajib `AppTypography`
- **Semua teks wajib memakai `AppTypography`** (`dekorin_apps/lib/config/theme/app_typography.dart`). **Jangan pernah** menulis `TextStyle(...)` langsung atau `GoogleFonts.*` di widget/screen.
- Pilih style dasar yang paling sesuai lalu sesuaikan dengan `.copyWith(...)`, contoh:
  - `style: AppTypography.bodyMedium`
  - `style: AppTypography.titleSmall.copyWith(color: AppColors.primaryGold)`
  - `style: AppTypography.caption.copyWith(fontSize: 10, fontWeight: FontWeight.bold)`
- Berlaku juga untuk `labelStyle`, `hintStyle`, `prefixStyle`, `titleTextStyle`, `TabBar.labelStyle`, `TextSpan`, dll.
- Referensi skala: `h1` 28 · `h2` 24 · `h3` 20 · `titleLarge` 18 · `titleMedium` 16 (w600) · `titleSmall` 14 (w600) · `bodyLarge` 16 · `bodyMedium` 14 · `bodySmall` 12 (textLight) · `labelLarge` 14 bold · `labelMedium` 12 w500 · `caption` 11 (textLight) · `button` 16 w600 putih.
- **Perhatian warna**: semua style `AppTypography` sudah memiliki `color` eksplisit, sehingga teks di dalam tombol (`ElevatedButton`, `FloatingActionButton`, `OutlinedButton`, `TextButton`, `Chip`) **tidak lagi mewarisi `foregroundColor`**. Selalu set `color` yang sesuai via `.copyWith(color: ...)` (mis. `Colors.white` untuk tombol gold) atau gunakan `AppTypography.button`.
- Pengecualian: `pw.TextStyle` pada pembuatan PDF (package `pdf`) tetap memakai API package tersebut.

## 4. Keamanan Database & Perlindungan Data User (CRITICAL / ZERO DATA LOSS)
- **DILARANG KERAS Menghapus atau Mengubah Isi Database**: Dalam kondisi apa pun, **JANGAN PERNAH** menjalankan perintah yang menghapus, mengosongkan, atau merusak data yang sudah ada di database (baik lokal maupun VPS/production).
- **Larangan Perintah SQL Destruktif**:
  - Dilarang menjalankan `DROP DATABASE`, `DROP TABLE`, `DROP COLUMN`, atau `TRUNCATE`.
  - Dilarang menjalankan `DELETE FROM` atau `UPDATE` langsung ke database tanpa instruksi dan konfirmasi eksplisit dari user.
- **Aturan Migrasi & Perubahan Model (Go / GORM)**:
  - Saat menambah atau memodifikasi struct model di `dekorin_apps_api`, pastikan perubahan bersifat **non-destruktif (additive only)**.
  - Kolom baru pada tabel yang sudah berjalan wajib aman bagi data lama (gunakan `default:...` atau *nullable* / pointer) agar `AutoMigrate` tidak gagal atau merusak record user.
  - Jangan pernah menghapus atau mengganti nama field model yang sudah dipakai di production tanpa membuat skema migrasi yang aman.
- **Larangan Reset Docker Volume & File Uploads**:
  - **DILARANG** menjalankan `docker compose down -v`, `docker volume rm`, atau `docker system prune --volumes` yang dapat menghapus volume database `pgdata` (`dekorin_postgres`).
  - **DILARANG** menghapus atau mengosongkan folder `uploads/` yang berisi foto/aset milik user.
- **Wajib Konfirmasi**: Jika suatu task mengharuskan perubahan skema database atau menyentuh server production/VPS, **wajib meminta konfirmasi user terlebih dahulu** dan jelaskan dampaknya terhadap data yang ada.

## 5. Standar Keamanan Release & Deployment (Mobile App & Backend VPS)
- **Kompatibilitas Mundur API (*Backward Compatibility*)**:
  - Jangan pernah menghapus atau mengganti nama key JSON pada response/request di `dekorin_apps_api` yang masih dipakai oleh APK versi lama di HP user.
  - Field baru pada request API wajib bersifat opsional / memiliki nilai *default* di backend agar user yang belum update APK tidak mengalami error (`400 Bad Request` / `500`).
- **Defensive Parsing (`fromJson`) di Flutter**:
  - Semua model di `dekorin_apps/lib/data/models/` wajib menggunakan *null-safe fallback* (mis. `json['field'] ?? ''`, `(json['price'] as num?)?.toDouble() ?? 0.0`) dan dilarang menggunakan *hard cast* tanpa nullable agar data lama di database tidak pernah menyebabkan *crash* / layar merah di aplikasi user.
- **Checklist Wajib Sebelum Build APK Release (`dekorin_apps`)**:
  - Pastikan `ApiEndpoint.baseUrl` di `dekorin_apps/lib/data/datasources/remote/api_endpoint.dart` mengarah ke URL production (`https://103.150.191.59.sslip.io/api`), bukan `localhost` atau IP lokal.
  - Naikkan `version` dan `build number` di `dekorin_apps/pubspec.yaml` setiap kali membuat rilis APK baru agar fitur In-App / OTA Update berjalan semestinya.
  - Wajib menjalankan `flutter analyze` dan memastikan **0 error** sebelum menjalankan `flutter build apk --release`.
  - Jika rilis membutuhkan perubahan endpoint/database baru, **backend wajib di-deploy ke VPS terlebih dahulu** sebelum APK baru didistribusikan ke user.
- **Checklist Wajib Saat Build & Push Backend (`dekorin_apps_api`)**:
  - Setiap ada perubahan kode Go di `dekorin_apps_api`, wajib melakukan cross-compile binary Linux (`GOOS=linux GOARCH=amd64 go build -o dekorin-api-linux ./cmd/api/main.go`) sebelum commit & push ke GitHub agar `update.sh` di VPS selalu mengunduh binary terbaru yang kompatibel dengan Linux.
  - Jika menambahkan file template HTML baru di `web/templates/`, wajib mendaftarkannya ke dalam loop unduhan di `dekorin_apps_api/update.sh`.
  - Jangan pernah meng-commit file `.env` atau kredensial rahasia ke repository Git.


