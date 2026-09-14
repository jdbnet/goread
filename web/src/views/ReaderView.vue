<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import EpubImport from "epubjs";
import type { Book as EpubBook, Rendition } from "epubjs";

const ePub =
  typeof EpubImport === "function"
    ? EpubImport
    : (EpubImport as { default: typeof EpubImport }).default;
import { ChevronLeft, Settings2, Sun, Moon, Lamp } from "@lucide/vue";
import { api } from "../api";
import { applyAccent } from "../accent";
import type { Book, Settings } from "../types";
import { useScreenWakeLock } from "../wakeLock";
import { hasBookFile } from "../offline/downloads";
import { getPending } from "../offline/progress";
import { online } from "../offline/status";

const props = defineProps<{ id: string }>();
const router = useRouter();
useScreenWakeLock();
const host = ref<HTMLElement | null>(null);
const book = ref<Book | null>(null);
const settings = ref<Settings>({ font_size: 18, line_height: 1.6, theme: "light", accent: "emerald" });
const showSettings = ref(false);
const error = ref("");
const loading = ref(true);

let epub: EpubBook | null = null;
let rendition: Rendition | null = null;
let heartbeat: ReturnType<typeof setInterval> | undefined;
let resizeObserver: ResizeObserver | undefined;
let lastCfi = "";
let lastPercent = 0;
let percentKnown = false;
let locationsReady = false;
let closed = false;

function measureHost(el: HTMLElement): { width: number; height: number } {
  const r = el.getBoundingClientRect();
  return { width: Math.floor(r.width), height: Math.floor(r.height) };
}

async function nextFrame(): Promise<void> {
  await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()));
}

async function settleLayout(): Promise<void> {
  await nextFrame();
  await nextFrame();
}

async function waitForHost(el: HTMLElement): Promise<{ width: number; height: number }> {
  for (let i = 0; i < 120; i++) {
    const size = measureHost(el);
    if (size.width >= 100 && size.height >= 100) {
      return size;
    }
    await nextFrame();
  }
  throw new Error("Reader layout did not settle.");
}

function fitRendition(el: HTMLElement) {
  if (!rendition) return;
  const { width, height } = measureHost(el);
  if (width < 50 || height < 50) return;
  rendition.resize(width, height);
}

function withTimeout<T>(promise: Promise<T>, ms: number, message: string): Promise<T> {
  return new Promise((resolve, reject) => {
    const t = window.setTimeout(() => reject(new Error(message)), ms);
    promise.then(
      (v) => {
        window.clearTimeout(t);
        resolve(v);
      },
      (err: unknown) => {
        window.clearTimeout(t);
        reject(err);
      },
    );
  });
}

const themeStyles: Record<Settings["theme"], Record<string, Record<string, string>>> = {
  light: { body: { background: "#fafaf9", color: "#1c1917" } },
  dark: { body: { background: "#0c0a09", color: "#e7e5e4" } },
  sepia: { body: { background: "#f4ecd8", color: "#5b4636" } },
};

function applyTheme() {
  if (!rendition) return;
  const t = settings.value.theme;
  rendition.themes.default({
    body: {
      ...themeStyles[t].body,
      "line-height": String(settings.value.line_height),
      "font-family": "Georgia, 'Times New Roman', serif",
    },
    p: { "line-height": String(settings.value.line_height) },
  });
  rendition.themes.fontSize(`${settings.value.font_size}px`);
}

async function persistSettings() {
  try {
    settings.value = await api.saveSettings(settings.value);
    applyTheme();
  } catch {
    applyTheme();
  }
}

function chromeBg(): string {
  if (settings.value.theme === "dark") return "#0c0a09";
  if (settings.value.theme === "sepia") return "#f4ecd8";
  return "#fafaf9";
}

function chromeFg(): string {
  if (settings.value.theme === "dark") return "#e7e5e4";
  if (settings.value.theme === "sepia") return "#5b4636";
  return "#1c1917";
}

async function sendProgress(delta: number) {
  if (!book.value) return;
  const body: {
    current_cfi?: string;
    percent_completed?: number;
    seconds_delta?: number;
  } = {
    seconds_delta: delta,
  };
  if (lastCfi) {
    body.current_cfi = lastCfi;
  }
  if (percentKnown) {
    body.percent_completed = lastPercent;
  }
  if (!body.current_cfi && body.percent_completed == null && delta <= 0) {
    return;
  }
  await api.postProgress(book.value.id, body);
}

function applyPercentFromCfi(cfi: string) {
  if (!locationsReady || !cfi || !epub) return;
  try {
    const p = epub.locations.percentageFromCfi(cfi);
    if (typeof p === "number" && Number.isFinite(p)) {
      lastPercent = Math.min(100, Math.max(0, p <= 1 ? p * 100 : p));
      percentKnown = true;
    }
  } catch {
    /* keep last */
  }
}

onMounted(async () => {
  try {
    const id = Number(props.id);
    const [b, s, pending] = await Promise.all([api.getBook(id), api.settings(), getPending(id)]);
    book.value = b;
    settings.value = { ...s, accent: s.accent || "emerald" };
    applyAccent(settings.value.accent);
    if (b.file_missing) {
      error.value = "This file is missing from the library.";
      loading.value = false;
      return;
    }
    const cachedFile = await hasBookFile(id);
    if (!online.value && !cachedFile) {
      error.value = "Download this book first to read offline.";
      loading.value = false;
      return;
    }
    lastCfi = pending?.current_cfi || b.current_cfi;
    lastPercent = pending?.percent_completed ?? b.percent_completed;
    percentKnown = lastPercent > 0;
    locationsReady = false;
    loading.value = true;
    await nextTick();
    await settleLayout();
    const el = host.value;
    if (!el) {
      error.value = "Reader failed to mount.";
      loading.value = false;
      return;
    }
    const { width, height } = await waitForHost(el);
    epub = ePub(`/api/v1/books/${id}/file`, {
      openAs: "epub",
      replacements: "blobUrl",
      requestCredentials: true,
    });
    rendition = epub.renderTo(el, {
      width,
      height,
      flow: "paginated",
      spread: "none",
      allowScriptedContent: true,
    });
    await withTimeout(epub.ready, 20000, "Timed out opening this EPUB.");
    if (closed) return;
    const initialTarget = lastCfi || undefined;
    await withTimeout(rendition.display(initialTarget), 12000, "Timed out rendering the first page.");
    if (closed) return;
    await settleLayout();
    fitRendition(el);
    applyTheme();
    // Chrome can miss the first paint at the pre-layout size; resize then redraw.
    await withTimeout(rendition.display(initialTarget), 12000, "Timed out rendering the first page.");
    applyTheme();
    loading.value = false;
    rendition.on("relocated", (loc) => {
      const cfi = loc.start?.cfi ?? "";
      lastCfi = cfi;
      applyPercentFromCfi(cfi);
    });
    void epub.locations.generate(1600)
      .then(() => {
        if (closed) return;
        locationsReady = true;
        applyPercentFromCfi(lastCfi);
        void sendProgress(0);
      })
      .catch(() => undefined);
    resizeObserver = new ResizeObserver(() => {
      if (!rendition || !host.value) return;
      fitRendition(host.value);
    });
    resizeObserver.observe(el);
    heartbeat = setInterval(() => {
      if (document.visibilityState !== "visible") return;
      void sendProgress(15);
    }, 15000);
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Failed to open book";
    loading.value = false;
  }
});

onBeforeUnmount(() => {
  closed = true;
  if (heartbeat) clearInterval(heartbeat);
  resizeObserver?.disconnect();
  void sendProgress(0);
  rendition?.destroy();
  epub?.destroy();
});

async function go(dir: "prev" | "next") {
  if (!rendition) return;
  if (dir === "next") await rendition.next();
  else await rendition.prev();
}

function leaveReader() {
  const dest = book.value ? `/books/${book.value.id}` : "/library";
  router.replace(dest);
}
</script>

<template>
  <div class="fixed inset-0 flex flex-col" :style="{ background: chromeBg(), color: chromeFg() }">
    <header class="safe-top flex shrink-0 items-center justify-between px-3 py-2">
      <button type="button" class="rounded-full p-2" @click="leaveReader">
        <ChevronLeft :size="24" />
      </button>
      <p class="max-w-[60%] truncate text-sm font-medium">{{ book?.title }}</p>
      <button type="button" class="rounded-full p-2" @click="showSettings = !showSettings">
        <Settings2 :size="20" />
      </button>
    </header>

    <p v-if="error" class="px-4 py-6 text-sm">{{ error }}</p>

    <div class="relative min-h-0 flex-1">
      <div ref="host" class="reader-host absolute inset-0 overflow-hidden" />
      <p
        v-if="loading && !error"
        class="pointer-events-none absolute inset-0 z-10 flex items-center justify-center text-sm opacity-70"
      >
        Opening book…
      </p>
      <button
        type="button"
        class="absolute inset-y-0 left-0 z-10 w-[28%] cursor-pointer appearance-none border-0 bg-transparent p-0"
        aria-label="Previous page"
        @click="go('prev')"
      />
      <button
        type="button"
        class="absolute inset-y-0 right-0 z-10 w-[28%] cursor-pointer appearance-none border-0 bg-transparent p-0"
        aria-label="Next page"
        @click="go('next')"
      />
    </div>

    <aside
      v-if="showSettings"
      class="safe-bottom absolute inset-x-0 bottom-0 z-20 rounded-t-2xl border-t p-4 shadow-xl"
      :style="{ background: chromeBg(), color: chromeFg(), borderColor: 'currentColor' }"
    >
      <div class="mb-3 flex items-center justify-between">
        <p class="text-sm font-semibold">Reader</p>
        <div class="flex gap-2">
          <button type="button" class="rounded-full p-2" :class="settings.theme === 'light' ? 'ring-2 ring-accent-bar' : ''" @click="settings.theme = 'light'; persistSettings()">
            <Sun :size="18" />
          </button>
          <button type="button" class="rounded-full p-2" :class="settings.theme === 'dark' ? 'ring-2 ring-accent-bar' : ''" @click="settings.theme = 'dark'; persistSettings()">
            <Moon :size="18" />
          </button>
          <button type="button" class="rounded-full p-2" :class="settings.theme === 'sepia' ? 'ring-2 ring-accent-bar' : ''" @click="settings.theme = 'sepia'; persistSettings()">
            <Lamp :size="18" />
          </button>
        </div>
      </div>
      <label class="block text-xs">Font size {{ settings.font_size }}px</label>
      <input v-model.number="settings.font_size" type="range" min="14" max="32" class="w-full" @change="persistSettings" />
      <label class="mt-2 block text-xs">Line height {{ settings.line_height }}</label>
      <input v-model.number="settings.line_height" type="range" min="1.2" max="2.2" step="0.1" class="w-full" @change="persistSettings" />
    </aside>
  </div>
</template>
