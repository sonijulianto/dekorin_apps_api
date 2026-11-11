const CACHE_VERSION = 'dekorin-admin-v1';

const PRECACHE_ASSETS = [
  '/assets/logo.png',
  '/assets/dekorin-favicon.png?v=20261111',
  '/assets/dekorin-apple-touch-icon.png?v=20261111',
  '/assets/logo_icon.png?v=20261111',
  '/assets/icons/pwa-maskable-192.png?v=20261111',
  '/assets/icons/pwa-maskable-512.png?v=20261111'
];

function buildOfflineResponse() {
  const html = `<!DOCTYPE html>
<html lang="id">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0, viewport-fit=cover">
  <meta name="theme-color" content="#D4A574">
  <title>Dekorin Admin - Sedang Offline</title>
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      background: linear-gradient(135deg, #F5F0EB 0%, #F8F5F1 50%, #EAE4DC 100%);
      color: #2C2C2C;
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: calc(16px + env(safe-area-inset-top, 0px)) 16px calc(16px + env(safe-area-inset-bottom, 0px));
    }
    .card {
      background: #FFFFFF;
      border: 1px solid rgba(120, 113, 108, 0.18);
      border-radius: 24px;
      box-shadow: 0 20px 45px rgba(44, 44, 44, 0.08);
      max-width: 400px;
      width: 100%;
      padding: 32px 24px;
      text-align: center;
    }
    .logo {
      height: 52px;
      width: auto;
      object-fit: contain;
      margin: 0 auto 18px;
      display: block;
    }
    .badge {
      display: inline-block;
      background: #F9F3EC;
      color: #C59563;
      border: 1px solid rgba(212, 165, 116, 0.45);
      border-radius: 999px;
      padding: 5px 12px;
      font-size: 11px;
      font-weight: 700;
      letter-spacing: 0.04em;
      text-transform: uppercase;
      margin-bottom: 14px;
    }
    h1 {
      font-size: 20px;
      font-weight: 700;
      color: #2C2C2C;
      margin-bottom: 8px;
    }
    p {
      font-size: 13px;
      line-height: 1.6;
      color: #6B6661;
      margin-bottom: 24px;
    }
    button {
      width: 100%;
      min-height: 46px;
      border: none;
      border-radius: 14px;
      background: #D4A574;
      color: #FFFFFF;
      font-size: 14px;
      font-weight: 700;
      cursor: pointer;
      box-shadow: 0 8px 20px rgba(212, 165, 116, 0.3);
    }
    button:active {
      background: #C59563;
      transform: scale(0.99);
    }
  </style>
</head>
<body>
  <div class="card">
    <img src="/assets/logo.png" alt="Dekorin" class="logo" onerror="this.style.display='none'">
    <div class="badge">Mode Offline</div>
    <h1>Koneksi Internet Terputus</h1>
    <p>Web Admin Dekorin membutuhkan koneksi internet aktif untuk menyinkronkan data agenda, katalog, dan laporan secara real-time. Sesi login Anda tetap aman tersimpan.</p>
    <button type="button" onclick="window.location.reload()">Coba Muat Ulang</button>
  </div>
</body>
</html>`;

  return new Response(html, {
    status: 200,
    headers: {
      'Content-Type': 'text/html; charset=utf-8',
      'Cache-Control': 'no-store'
    }
  });
}

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_VERSION).then((cache) =>
      Promise.allSettled(
        PRECACHE_ASSETS.map((assetUrl) =>
          cache.add(new Request(assetUrl, { cache: 'reload' }))
        )
      )
    )
  );
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((keys) =>
      Promise.all(
        keys
          .filter((key) => key.startsWith('dekorin-admin-') && key !== CACHE_VERSION)
          .map((key) => caches.delete(key))
      )
    ).then(() => self.clients.claim())
  );
});

self.addEventListener('fetch', (event) => {
  const request = event.request;
  if (request.method !== 'GET') return;

  const url = new URL(request.url);
  if (url.origin !== self.location.origin) return;

  // Jangan pernah meng-cache API, download APK, upload user, atau service worker itu sendiri
  if (
    url.pathname.startsWith('/api/') ||
    url.pathname.startsWith('/download/') ||
    url.pathname.startsWith('/uploads/') ||
    url.pathname === '/superadmin/sw.js'
  ) {
    return;
  }

  // Navigasi halaman /superadmin: Network-First dengan fallback halaman offline mandiri
  if (request.mode === 'navigate' && url.pathname.startsWith('/superadmin')) {
    event.respondWith(
      fetch(request).catch(() => buildOfflineResponse())
    );
    return;
  }

  // Cache ringan aset inti admin (ikon & logo) dengan strategi Stale-While-Revalidate
  const isAdminAsset =
    url.pathname.startsWith('/assets/') &&
    !url.pathname.startsWith('/assets/templates/') &&
    !url.pathname.startsWith('/assets/owlink/');

  if (isAdminAsset) {
    event.respondWith(
      caches.open(CACHE_VERSION).then((cache) =>
        cache.match(request).then((cachedResponse) => {
          const networkFetch = fetch(request)
            .then((networkResponse) => {
              if (networkResponse && networkResponse.ok) {
                cache.put(request, networkResponse.clone());
              }
              return networkResponse;
            })
            .catch(() => cachedResponse);

          return cachedResponse || networkFetch;
        })
      )
    );
  }
});
