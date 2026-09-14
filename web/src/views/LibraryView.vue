<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { Layers, Search, SlidersHorizontal, SquareCheck, X } from "@lucide/vue";
import { api } from "../api";
import type { Book } from "../types";
import BookCard from "../components/BookCard.vue";
import AddToSeriesModal from "../components/AddToSeriesModal.vue";
import { listDownloads } from "../offline/downloads";
import { online } from "../offline/status";
import { overlayBook, syncTick } from "../offline/progress";

const books = ref<Book[]>([]);
const total = ref(0);
const q = ref("");
const author = ref("");
const status = ref("");
const sort = ref("added");
const authors = ref<string[]>([]);
const page = ref(1);
const loading = ref(false);
const showFilters = ref(false);
const selecting = ref(false);
const selected = ref<number[]>([]);
const showSeriesModal = ref(false);
const notice = ref("");

const selectedCount = computed(() => selected.value.length);
const pageSize = 24;

function sortBooks(list: Book[]): Book[] {
  const copy = [...list];
  if (sort.value === "title") {
    copy.sort((a, b) => a.title.localeCompare(b.title));
  } else if (sort.value === "author") {
    copy.sort((a, b) => a.author.localeCompare(b.author) || a.title.localeCompare(b.title));
  } else if (sort.value === "recent") {
    copy.sort((a, b) => Date.parse(b.last_read_at || "0") - Date.parse(a.last_read_at || "0"));
  } else {
    copy.sort((a, b) => Date.parse(b.created_at) - Date.parse(a.created_at));
  }
  return copy;
}

async function loadDownloaded() {
  loading.value = true;
  try {
    const rows = await listDownloads();
    const needle = q.value.trim().toLowerCase();
    let list = rows.map((row) => overlayBook(row.snapshot));
    if (needle) {
      list = list.filter(
        (b) => b.title.toLowerCase().includes(needle) || b.author.toLowerCase().includes(needle),
      );
    }
    if (author.value) {
      list = list.filter((b) => b.author === author.value);
    }
    list = sortBooks(list);
    total.value = list.length;
    const start = (page.value - 1) * pageSize;
    books.value = list.slice(start, start + pageSize);
  } finally {
    loading.value = false;
  }
}

async function load() {
  if (status.value === "downloaded") {
    await loadDownloaded();
    return;
  }
  loading.value = true;
  try {
    const res = await api.listBooks({
      q: q.value,
      author: author.value,
      status: status.value,
      sort: sort.value,
      page: page.value,
      limit: pageSize,
    });
    books.value = res.books;
    total.value = res.total;
  } catch {
    if (!books.value.length) {
      notice.value = "Library is unavailable offline until it has been opened online.";
    }
  } finally {
    loading.value = false;
  }
}

onMounted(async () => {
  try {
    authors.value = (await api.authors()).authors;
  } catch {
    authors.value = [];
  }
  await load();
});

watch([sort, status, author], () => {
  page.value = 1;
  void load();
});
watch(syncTick, () => {
  void load();
});

let t: ReturnType<typeof setTimeout> | undefined;
watch(q, () => {
  clearTimeout(t);
  t = setTimeout(() => {
    page.value = 1;
    void load();
  }, 250);
});

function toggleSelectMode() {
  selecting.value = !selecting.value;
  selected.value = [];
  notice.value = "";
}

function toggleBook(id: number) {
  if (selected.value.includes(id)) {
    selected.value = selected.value.filter((x) => x !== id);
  } else {
    selected.value = [...selected.value, id];
  }
}

function isSelected(id: number): boolean {
  return selected.value.includes(id);
}

function onAssigned(name: string) {
  showSeriesModal.value = false;
  selecting.value = false;
  selected.value = [];
  notice.value = `Added to ${name}`;
  void load();
}
</script>

<template>
  <div>
    <header class="mb-4 flex items-start justify-between gap-3">
      <div>
        <h1 class="text-2xl font-bold tracking-tight">Library</h1>
        <p class="text-sm text-stone-500">
          <template v-if="selecting">{{ selectedCount }} selected</template>
          <template v-else>{{ total }} books</template>
        </p>
      </div>
      <button
        type="button"
        class="inline-flex items-center gap-2 rounded-full border border-stone-300 bg-white px-3 py-2 text-sm font-medium dark:border-stone-700 dark:bg-stone-900"
        @click="toggleSelectMode"
      >
        <X v-if="selecting" :size="16" />
        <SquareCheck v-else :size="16" />
        {{ selecting ? "Cancel" : "Select" }}
      </button>
    </header>

    <p v-if="notice" class="mb-3 text-sm text-stone-500">{{ notice }}</p>

    <div v-if="selecting && selectedCount > 0" class="mb-4 flex gap-2">
      <button
        type="button"
        class="inline-flex flex-1 items-center justify-center gap-2 rounded-xl bg-accent px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-50"
        :disabled="!online"
        @click="showSeriesModal = true"
      >
        <Layers :size="16" />
        Add to series
      </button>
    </div>

    <div class="mb-4 flex gap-2">
      <label class="relative flex-1">
        <Search class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-stone-400" :size="18" />
        <input
          v-model="q"
          type="search"
          placeholder="Title or author"
          class="w-full rounded-xl border border-stone-300 bg-white py-2.5 pl-10 pr-3 text-sm dark:border-stone-700 dark:bg-stone-900"
        />
      </label>
      <button
        type="button"
        class="rounded-xl border border-stone-300 bg-white px-3 dark:border-stone-700 dark:bg-stone-900"
        @click="showFilters = !showFilters"
      >
        <SlidersHorizontal :size="18" />
      </button>
    </div>

    <div v-if="showFilters" class="mb-4 grid grid-cols-1 gap-2 sm:grid-cols-3">
      <select v-model="sort" class="rounded-xl border border-stone-300 bg-white px-3 py-2 text-sm dark:border-stone-700 dark:bg-stone-900">
        <option value="added">Recently added</option>
        <option value="title">Title</option>
        <option value="author">Author</option>
        <option value="recent">Recently read</option>
      </select>
      <select v-model="status" class="rounded-xl border border-stone-300 bg-white px-3 py-2 text-sm dark:border-stone-700 dark:bg-stone-900">
        <option value="">All</option>
        <option value="unread">Unread</option>
        <option value="reading">Reading</option>
        <option value="completed">Completed</option>
        <option value="downloaded">Downloaded</option>
      </select>
      <select v-model="author" class="rounded-xl border border-stone-300 bg-white px-3 py-2 text-sm dark:border-stone-700 dark:bg-stone-900">
        <option value="">Any author</option>
        <option v-for="a in authors" :key="a" :value="a">{{ a }}</option>
      </select>
    </div>

    <div v-if="loading && !books.length" class="text-sm text-stone-500">Loading…</div>
    <div v-else class="grid grid-cols-3 gap-4 sm:grid-cols-4 md:grid-cols-6">
      <BookCard
        v-for="b in books"
        :key="b.id"
        :book="b"
        :selecting="selecting"
        :selected="isSelected(b.id)"
        @toggle="toggleBook(b.id)"
      />
    </div>

    <div v-if="total > pageSize" class="mt-6 flex justify-center gap-3">
      <button
        type="button"
        class="rounded-full border px-4 py-1.5 text-sm disabled:opacity-40"
        :disabled="page <= 1"
        @click="page--; load()"
      >
        Prev
      </button>
      <span class="py-1.5 text-sm text-stone-500">Page {{ page }}</span>
      <button
        type="button"
        class="rounded-full border px-4 py-1.5 text-sm disabled:opacity-40"
        :disabled="page * pageSize >= total"
        @click="page++; load()"
      >
        Next
      </button>
    </div>

    <AddToSeriesModal
      v-if="showSeriesModal"
      :book-ids="selected"
      @close="showSeriesModal = false"
      @assigned="onAssigned"
    />
  </div>
</template>
