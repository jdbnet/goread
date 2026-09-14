import { onBeforeUnmount, onMounted } from "vue";

function canRequestWakeLock(): boolean {
  return typeof navigator !== "undefined" && "wakeLock" in navigator;
}

export function useScreenWakeLock(): void {
  let sentinel: WakeLockSentinel | null = null;
  let enabled = false;
  let requestId = 0;

  async function acquire(): Promise<void> {
    if (!enabled || document.visibilityState !== "visible") return;
    if (!canRequestWakeLock()) return;
    if (sentinel && !sentinel.released) return;
    const id = ++requestId;
    try {
      const next = await navigator.wakeLock.request("screen");
      if (id !== requestId || !enabled) {
        await next.release();
        return;
      }
      sentinel = next;
      next.addEventListener("release", () => {
        if (sentinel === next) {
          sentinel = null;
        }
      });
    } catch {
      sentinel = null;
    }
  }

  function onVisibility(): void {
    if (document.visibilityState === "visible") {
      void acquire();
    }
  }

  function onPointerDown(): void {
    if (!sentinel || sentinel.released) {
      void acquire();
    }
  }

  onMounted(() => {
    enabled = true;
    document.addEventListener("visibilitychange", onVisibility);
    document.addEventListener("pointerdown", onPointerDown);
    void acquire();
  });

  onBeforeUnmount(() => {
    enabled = false;
    requestId += 1;
    document.removeEventListener("visibilitychange", onVisibility);
    document.removeEventListener("pointerdown", onPointerDown);
    const held = sentinel;
    sentinel = null;
    if (held && !held.released) {
      void held.release();
    }
  });
}
