<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { ArrowLeft, BookOpen, Search, AlertTriangle, Check, Pencil, LoaderCircle } from "@lucide/vue";
import { api, stripHtml } from "../api";
import type { Book, MetadataHit } from "../types";
import CoverImage from "../components/CoverImage.vue";
import ProgressBar from "../components/ProgressBar.vue";
import MetadataEditModal from "../components/MetadataEditModal.vue";

const props = defineProps<{ id: string }>();
const router = useRouter();
const book = ref<Book | null>(null);
const query = ref("");
const hits = ref<MetadataHit[]>([]);
const searching = ref(false);
const applying = ref(false);
const message = ref("");
const editing = ref(false);

const bookId = computed(() => Number(props.id));

async function load() {
  book.value = await api.getBook(bookId.value);
  query.value = [book.value.title, book.value.author, book.value.isbn].filter(Boolean).join(" ");
}

async function search() {
  searching.value = true;
  message.value = "";
  try {
    hits.value = (await api.searchMetadata(query.value)).results;
  } catch (e) {
    message.value = e instanceof Error ? e.message : "Search failed";
  } finally {
    searching.value = false;
  }
}

async function apply(hit: MetadataHit) {
  applying.value = true;
  message.value = "";
  try {
    book.value = await api.applyMetadata(bookId.value, hit);
    message.value = "Metadata applied";
    hits.value = [];
  } catch (e) {
    message.value = e instanceof Error ? e.message : "Apply failed";
  } finally {
    applying.value = false;
  }
}

async function toggleComplete() {
  if (!book.value) return;
  await api.postProgress(book.value.id, { completed: !book.value.completed_at });
  await load();
}

function onEdited(updated: Book) {
  book.value = updated;
  editing.value = false;
  message.value = "Saved";
}

onMounted(load);
</script>

<template>
  <div v-if="book">
    <button type="button" class="mb-3 inline-flex items-center gap-1 text-sm text-stone-500" @click="router.replace('/library')">
      <ArrowLeft :size="16" /> Back
    </button>

    <div class="flex gap-4">
      <div class="w-28 shrink-0 sm:w-36">
        <CoverImage :book="book" />
      </div>
      <div class="min-w-0 flex-1">
        <h1 class="text-xl font-bold leading-tight">{{ book.title }}</h1>
        <p class="mt-1 text-stone-500">{{ book.author }}</p>
        <p v-if="book.series_name" class="mt-1 text-sm text-accent-strong dark:text-accent-soft">
          {{ book.series_name }}
          <span v-if="book.sequence_number != null">· {{ book.sequence_number }}</span>
        </p>
        <div v-if="book.percent_completed > 0" class="mt-3 max-w-xs">
          <ProgressBar :value="book.percent_completed" />
          <p class="mt-1 text-xs text-stone-500">{{ Math.round(book.percent_completed) }}%</p>
        </div>
        <div class="mt-4 flex flex-wrap gap-2">
          <button
            v-if="!book.file_missing"
            type="button"
            class="inline-flex items-center gap-2 rounded-full bg-accent px-4 py-2 text-sm font-semibold text-white"
            @click="router.replace(`/read/${book.id}`)"
          >
            <BookOpen :size="16" />
            {{ book.percent_completed > 0 ? "Continue" : "Read" }}
          </button>
          <button
            type="button"
            class="inline-flex items-center gap-2 rounded-full border border-stone-300 px-4 py-2 text-sm dark:border-stone-700"
            @click="editing = true"
          >
            <Pencil :size="16" />
            Edit
          </button>
          <button
            type="button"
            class="rounded-full border border-stone-300 px-4 py-2 text-sm dark:border-stone-700"
            @click="toggleComplete"
          >
            <span class="inline-flex items-center gap-1">
              <Check :size="16" />
              {{ book.completed_at ? "Mark unread" : "Mark complete" }}
            </span>
          </button>
        </div>
      </div>
    </div>

    <p v-if="book.file_missing" class="mt-4 flex items-center gap-2 rounded-xl bg-red-50 px-3 py-2 text-sm text-red-800 dark:bg-red-950 dark:text-red-200">
      <AlertTriangle :size="16" /> File is missing from the library. Progress is kept.
    </p>

    <p v-if="book.description" class="mt-5 whitespace-pre-wrap text-sm leading-relaxed text-stone-600 dark:text-stone-300">
      {{ stripHtml(book.description) }}
    </p>
    <dl class="mt-4 grid grid-cols-2 gap-2 text-sm sm:grid-cols-3">
      <div v-if="book.publisher"><dt class="text-xs text-stone-500">Publisher</dt><dd>{{ book.publisher }}</dd></div>
      <div v-if="book.pub_date"><dt class="text-xs text-stone-500">Published</dt><dd>{{ book.pub_date }}</dd></div>
      <div v-if="book.isbn"><dt class="text-xs text-stone-500">ISBN</dt><dd>{{ book.isbn }}</dd></div>
      <div v-if="book.language"><dt class="text-xs text-stone-500">Language</dt><dd>{{ book.language }}</dd></div>
    </dl>

    <section class="mt-8">
      <h2 class="mb-3 text-lg font-semibold">Search & match</h2>
      <div class="flex gap-2">
        <input v-model="query" class="flex-1 rounded-xl border border-stone-300 bg-white px-3 py-2 text-sm dark:border-stone-700 dark:bg-stone-900" @keydown.enter="search" />
        <button
          type="button"
          class="inline-flex items-center gap-2 rounded-xl border px-3 py-2 text-sm disabled:opacity-50"
          :disabled="searching"
          @click="search"
        >
          <LoaderCircle v-if="searching" :size="16" class="animate-spin" />
          <Search v-else :size="16" />
          {{ searching ? "Matching…" : "Match" }}
        </button>
      </div>
      <p v-if="message" class="mt-2 text-sm text-stone-500">{{ message }}</p>
      <ul class="mt-4 space-y-3">
        <li
          v-for="(hit, i) in hits"
          :key="i"
          class="flex gap-3 rounded-2xl border border-stone-200 p-3 dark:border-stone-800"
        >
          <img v-if="hit.cover_url" :src="hit.cover_url" alt="" class="h-20 w-14 rounded object-cover" />
          <div class="min-w-0 flex-1">
            <p class="font-semibold leading-snug">{{ hit.title }}</p>
            <p class="text-xs text-stone-500">{{ hit.author }}</p>
            <button
              type="button"
              class="mt-2 rounded-full bg-accent px-3 py-1 text-xs font-semibold text-white disabled:opacity-50"
              :disabled="applying"
              @click="apply(hit)"
            >
              Apply
            </button>
          </div>
        </li>
      </ul>
    </section>

    <MetadataEditModal
      v-if="editing"
      :book="book"
      @close="editing = false"
      @saved="onEdited"
    />
  </div>
</template>
