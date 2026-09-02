<script setup lang="ts">
import { onMounted, ref, watch } from "vue";
import { Search, SlidersHorizontal } from "@lucide/vue";
import { api } from "../api";
import type { Book } from "../types";
import BookCard from "../components/BookCard.vue";

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

async function load() {
  loading.value = true;
  try {
    const res = await api.listBooks({
      q: q.value,
      author: author.value,
      status: status.value,
      sort: sort.value,
      page: page.value,
      limit: 24,
    });
    books.value = res.books;
    total.value = res.total;
  } finally {
    loading.value = false;
  }
}

onMounted(async () => {
  authors.value = (await api.authors()).authors;
  await load();
});

watch([sort, status, author], () => {
  page.value = 1;
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
</script>

<template>
  <div>
    <header class="mb-4">
      <h1 class="text-2xl font-bold tracking-tight">Library</h1>
      <p class="text-sm text-stone-500">{{ total }} books</p>
    </header>

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
      </select>
      <select v-model="author" class="rounded-xl border border-stone-300 bg-white px-3 py-2 text-sm dark:border-stone-700 dark:bg-stone-900">
        <option value="">Any author</option>
        <option v-for="a in authors" :key="a" :value="a">{{ a }}</option>
      </select>
    </div>

    <div v-if="loading && !books.length" class="text-sm text-stone-500">Loading…</div>
    <div v-else class="grid grid-cols-3 gap-4 sm:grid-cols-4 md:grid-cols-6">
      <BookCard v-for="b in books" :key="b.id" :book="b" />
    </div>

    <div v-if="total > 24" class="mt-6 flex justify-center gap-3">
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
        :disabled="page * 24 >= total"
        @click="page++; load()"
      >
        Next
      </button>
    </div>
  </div>
</template>
