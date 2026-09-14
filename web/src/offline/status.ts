import { ref } from "vue";
import { hydrateDownloads } from "./downloads";
import { flushPending, hydratePending } from "./progress";

export const online = ref(typeof navigator === "undefined" ? true : navigator.onLine);

const PREFETCH_PATHS = [
  "/api/v1/library/books?continue=1&limit=20",
  "/api/v1/library/books?sort=added&limit=12",
  "/api/v1/library/books?limit=24",
  "/api/v1/library/series",
  "/api/v1/library/authors",
  "/api/v1/stats",
  "/api/v1/settings",
  "/api/v1/auth/status",
];

export async function prefetchCore(): Promise<void> {
  if (!navigator.onLine) return;
  await Promise.allSettled(
    PREFETCH_PATHS.map((path) => fetch(path, { credentials: "same-origin", headers: { Accept: "application/json" } })),
  );
}

export async function startOffline(): Promise<void> {
  await Promise.all([hydrateDownloads(), hydratePending()]);
  online.value = navigator.onLine;
  window.addEventListener("online", () => {
    online.value = true;
    void flushPending();
    void prefetchCore();
  });
  window.addEventListener("offline", () => {
    online.value = false;
  });
  document.addEventListener("visibilitychange", () => {
    if (document.visibilityState === "visible" && navigator.onLine) {
      void flushPending();
    }
  });
  if (navigator.onLine) {
    void flushPending();
  }
}

export async function registerServiceWorker(): Promise<void> {
  if (!import.meta.env.PROD || !("serviceWorker" in navigator)) return;
  try {
    await navigator.serviceWorker.register("/serviceworker.js");
    await navigator.serviceWorker.ready;
  } catch {
    /* SW is best effort */
  }
}
