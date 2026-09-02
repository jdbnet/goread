<script setup lang="ts">
import { onMounted, ref } from "vue";
import { BookOpen, RefreshCw } from "@lucide/vue";
import { api } from "../api";
import type { Book } from "../types";
import ContinueCard from "../components/ContinueCard.vue";
import BookCard from "../components/BookCard.vue";

const continueBooks = ref<Book[]>([]);
const recent = ref<Book[]>([]);
const scanning = ref(false);
const error = ref("");

async function load() {
  error.value = "";
  try {
    const [c, r] = await Promise.all([
      api.continueReading(),
      api.listBooks({ sort: "added", limit: 12 }),
    ]);
    continueBooks.value = c.books;
    recent.value = r.books;
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Failed to load";
  }
}

async function scan() {
  scanning.value = true;
  try {
    await api.scan();
    const start = Date.now();
    while (Date.now() - start < 30000) {
      const st = await api.scanStatus();
      if (!st.running) break;
      await new Promise((r) => setTimeout(r, 400));
    }
    await load();
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Scan failed";
  } finally {
    scanning.value = false;
  }
}

onMounted(load);
</script>

<template>
  <div>
    <header class="mb-6 flex items-center justify-between gap-3">
      <div>
        <p class="text-xs font-semibold uppercase tracking-wider text-accent dark:text-accent-soft">Library</p>
        <h1 class="text-2xl font-bold tracking-tight">Continue reading</h1>
      </div>
      <button
        type="button"
        class="inline-flex items-center gap-2 rounded-full border border-stone-300 bg-white px-3 py-2 text-sm font-medium dark:border-stone-700 dark:bg-stone-900"
        :disabled="scanning"
        @click="scan"
      >
        <RefreshCw :size="16" :class="{ 'animate-spin': scanning }" />
        Scan
      </button>
    </header>

    <p v-if="error" class="mb-4 text-sm text-red-600">{{ error }}</p>

    <section v-if="continueBooks.length" class="mb-8">
      <div class="-mx-4 flex gap-4 overflow-x-auto px-4 pb-2">
        <ContinueCard v-for="b in continueBooks" :key="b.id" :book="b" />
      </div>
    </section>
    <section v-else class="mb-8 rounded-2xl border border-dashed border-stone-300 p-8 text-center dark:border-stone-700">
      <BookOpen class="mx-auto mb-3 text-stone-400" :size="36" />
      <p class="font-medium">Nothing in progress</p>
      <p class="mt-1 text-sm text-stone-500">Open a book from the library to start a session.</p>
    </section>

    <section>
      <h2 class="mb-3 text-lg font-semibold">Recently added</h2>
      <div v-if="recent.length" class="grid grid-cols-3 gap-4 sm:grid-cols-4 md:grid-cols-6">
        <BookCard v-for="b in recent" :key="b.id" :book="b" />
      </div>
      <p v-else class="text-sm text-stone-500">No books yet. Point LIBRARY_PATH at a folder of EPUBs and scan.</p>
    </section>
  </div>
</template>
