const SHELL_CACHE = "goread-shell-v1";
const API_CACHE = "goread-api-v2";
const BOOKS_CACHE = "goread-books";
const API_CACHE_MAX_ENTRIES = 48;

const PRECACHE_URLS = [
  "/",
  "/index.html",
  "/manifest.json",
  "/favicon.png",
  "/logo.png",
  "/apple-touch-icon.png",
  "/icons/icon-192.png",
  "/icons/icon-512.png",
  "/icons/icon-maskable-192.png",
  "/icons/icon-maskable-512.png",
];

self.addEventListener("install", (event) => {
  event.waitUntil(
    (async () => {
      const cache = await caches.open(SHELL_CACHE);
      await Promise.all(
        PRECACHE_URLS.map(async (url) => {
          try {
            const res = await fetch(url, { credentials: "same-origin" });
            if (res.ok) {
              await cache.put(url, res);
            }
          } catch {
            /* first visit may race */
          }
        }),
      );
      await self.skipWaiting();
    })(),
  );
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    (async () => {
      const keys = await caches.keys();
      await Promise.all(
        keys.map((key) => {
          const keep =
            key === SHELL_CACHE || key === API_CACHE || key === BOOKS_CACHE;
          if (keep) return Promise.resolve();
          if (key.startsWith("goread-shell-") || key.startsWith("goread-api-")) {
            return caches.delete(key);
          }
          if (key === "goread-api-v1") {
            return caches.delete(key);
          }
          if (key === BOOKS_CACHE || key.startsWith("goread-books")) {
            return Promise.resolve();
          }
          return Promise.resolve();
        }),
      );
      await self.clients.claim();
    })(),
  );
});

self.addEventListener("fetch", (event) => {
  const request = event.request;
  if (request.method !== "GET") {
    return;
  }
  const url = new URL(request.url);
  if (url.origin !== self.location.origin) {
    return;
  }

  if (request.mode === "navigate") {
    event.respondWith(navigationResponse(request));
    return;
  }

  if (isBookFile(url)) {
    event.respondWith(bookFileResponse(request, url));
    return;
  }

  if (isCacheableApi(url)) {
    event.respondWith(networkFirst(request, API_CACHE));
    return;
  }

  event.respondWith(networkFirst(request, SHELL_CACHE));
});

function isBookFile(url) {
  return /\/api\/v1\/books\/\d+\/file$/.test(url.pathname);
}

function isCacheableApi(url) {
  const path = url.pathname;
  if (!path.startsWith("/api/v1/")) return false;
  if (isBookFile(url)) return false;
  if (path === "/api/v1/scan" || path.startsWith("/api/v1/scan/")) return false;
  if (path.startsWith("/api/v1/metadata/")) return false;
  if (
    path === "/api/v1/auth/login" ||
    path === "/api/v1/auth/logout" ||
    path === "/api/v1/auth/credentials" ||
    path === "/api/v1/auth/disable"
  ) {
    return false;
  }
  return true;
}

function bookCacheKey(url) {
  return url.pathname;
}

async function navigationResponse(request) {
  try {
    const res = await fetch(request);
    if (res.ok) {
      const cache = await caches.open(SHELL_CACHE);
      await cache.put("/index.html", res.clone());
      await cache.put("/", res.clone());
    }
    return res;
  } catch {
    return (
      (await caches.match("/index.html")) ||
      (await caches.match("/")) ||
      new Response("GoRead is offline", {
        status: 503,
        headers: { "Content-Type": "text/plain; charset=utf-8" },
      })
    );
  }
}

async function bookFileResponse(request, url) {
  const key = bookCacheKey(url);
  const books = await caches.open(BOOKS_CACHE);
  const cached = await books.match(key);
  if (cached) {
    return cached;
  }
  const headers = new Headers(request.headers);
  headers.delete("Range");
  const res = await fetch(url.pathname, {
    method: "GET",
    headers,
    credentials: request.credentials,
    cache: "no-store",
  });
  return res;
}

async function trimCache(cache, maxEntries) {
  const keys = await cache.keys();
  if (keys.length <= maxEntries) return;
  const excess = keys.length - maxEntries;
  await Promise.all(keys.slice(0, excess).map((key) => cache.delete(key)));
}

async function networkFirst(request, cacheName) {
  try {
    const res = await fetch(request);
    if (res.ok) {
      const cache = await caches.open(cacheName);
      await cache.put(request, res.clone());
      if (cacheName === API_CACHE) {
        await trimCache(cache, API_CACHE_MAX_ENTRIES);
      }
    }
    return res;
  } catch (err) {
    const cached = await caches.match(request);
    if (cached) return cached;
    throw err;
  }
}
