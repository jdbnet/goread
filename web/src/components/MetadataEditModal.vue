<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from "vue";
import { X, ImagePlus } from "@lucide/vue";
import { api, stripHtml } from "../api";
import type { Book } from "../types";

const props = defineProps<{ book: Book }>();
const emit = defineEmits<{
  close: [];
  saved: [book: Book];
}>();

const title = ref("");
const author = ref("");
const description = ref("");
const publisher = ref("");
const pubDate = ref("");
const isbn = ref("");
const language = ref("");
const seriesName = ref("");
const sequence = ref("");
const file = ref<File | null>(null);
const preview = ref("");
const saving = ref(false);
const error = ref("");

function fill(b: Book) {
  title.value = b.title;
  author.value = b.author;
  description.value = stripHtml(b.description);
  publisher.value = b.publisher;
  pubDate.value = b.pub_date;
  isbn.value = b.isbn;
  language.value = b.language;
  seriesName.value = b.series_name;
  sequence.value = b.sequence_number != null ? String(b.sequence_number) : "";
  file.value = null;
  preview.value = b.cover_url || "";
}

watch(() => props.book, fill, { immediate: true });

function onFile(ev: Event) {
  const input = ev.target as HTMLInputElement;
  const next = input.files?.[0] ?? null;
  file.value = next;
  if (preview.value.startsWith("blob:")) {
    URL.revokeObjectURL(preview.value);
  }
  preview.value = next ? URL.createObjectURL(next) : props.book.cover_url || "";
}

function onKey(ev: KeyboardEvent) {
  if (ev.key === "Escape") {
    emit("close");
  }
}

onMounted(() => {
  document.addEventListener("keydown", onKey);
});

onUnmounted(() => {
  document.removeEventListener("keydown", onKey);
  if (preview.value.startsWith("blob:")) {
    URL.revokeObjectURL(preview.value);
  }
});

async function save() {
  saving.value = true;
  error.value = "";
  try {
    let updated = await api.applyMetadata(props.book.id, {
      title: title.value,
      author: author.value,
      description: description.value,
      publisher: publisher.value,
      pub_date: pubDate.value,
      isbn: isbn.value,
      language: language.value,
    });
    const seq = sequence.value === "" ? null : Number(sequence.value);
    updated = await api.setSeries(updated.id, seriesName.value, Number.isFinite(seq) ? seq : null);
    if (file.value) {
      updated = await api.uploadCover(updated.id, file.value);
    }
    emit("saved", updated);
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Save failed";
  } finally {
    saving.value = false;
  }
}

const fieldClass =
  "w-full rounded-xl border border-stone-300 bg-white px-3 py-2 text-sm dark:border-stone-700 dark:bg-stone-900";
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-end justify-center sm:items-center">
    <button type="button" class="absolute inset-0 bg-black/40" aria-label="Close" @click="emit('close')" />
    <div
      class="relative z-10 flex max-h-[92dvh] w-full flex-col overflow-hidden rounded-t-2xl bg-stone-50 dark:bg-stone-950 sm:max-w-lg sm:rounded-2xl"
      role="dialog"
      aria-labelledby="edit-meta-title"
    >
      <header class="flex shrink-0 items-center justify-between border-b border-stone-200 px-4 py-3 dark:border-stone-800">
        <h2 id="edit-meta-title" class="text-lg font-semibold">Edit book</h2>
        <button type="button" class="rounded-full p-1" @click="emit('close')">
          <X :size="20" />
        </button>
      </header>
      <form class="flex min-h-0 flex-1 flex-col" @submit.prevent="save">
        <div class="flex-1 space-y-3 overflow-y-auto px-4 py-4">
          <div class="space-y-1">
            <span class="text-xs font-medium text-stone-500">Cover</span>
            <label class="flex cursor-pointer items-center gap-3">
              <span class="h-28 w-20 shrink-0 overflow-hidden rounded-lg bg-stone-200 dark:bg-stone-800">
                <img v-if="preview" :key="preview" :src="preview" alt="" class="h-full w-full object-cover" />
                <span v-else class="flex h-full items-center justify-center text-stone-400">
                  <ImagePlus :size="22" />
                </span>
              </span>
              <span class="min-w-0">
                <span
                  class="inline-flex rounded-xl bg-accent px-4 py-2 text-sm font-semibold text-white"
                >
                  {{ file ? "Change file" : "Choose file" }}
                </span>
                <span class="mt-1.5 block text-xs text-stone-500">JPEG, PNG, WebP, or GIF</span>
              </span>
              <input
                type="file"
                accept="image/jpeg,image/png,image/webp,image/gif"
                class="sr-only"
                @change="onFile"
              />
            </label>
          </div>
          <label class="block space-y-1">
            <span class="text-xs font-medium text-stone-500">Title</span>
            <input v-model="title" :class="fieldClass" />
          </label>
          <label class="block space-y-1">
            <span class="text-xs font-medium text-stone-500">Author</span>
            <input v-model="author" :class="fieldClass" />
          </label>
          <label class="block space-y-1">
            <span class="text-xs font-medium text-stone-500">Description</span>
            <textarea v-model="description" rows="4" :class="fieldClass" />
          </label>
          <div class="grid grid-cols-2 gap-3">
            <label class="block space-y-1">
              <span class="text-xs font-medium text-stone-500">Publisher</span>
              <input v-model="publisher" :class="fieldClass" />
            </label>
            <label class="block space-y-1">
              <span class="text-xs font-medium text-stone-500">Published</span>
              <input v-model="pubDate" :class="fieldClass" />
            </label>
            <label class="block space-y-1">
              <span class="text-xs font-medium text-stone-500">ISBN</span>
              <input v-model="isbn" :class="fieldClass" />
            </label>
            <label class="block space-y-1">
              <span class="text-xs font-medium text-stone-500">Language</span>
              <input v-model="language" :class="fieldClass" />
            </label>
            <label class="block space-y-1">
              <span class="text-xs font-medium text-stone-500">Series</span>
              <input v-model="seriesName" :class="fieldClass" />
            </label>
            <label class="block space-y-1">
              <span class="text-xs font-medium text-stone-500">Sequence</span>
              <input v-model="sequence" inputmode="decimal" :class="fieldClass" />
            </label>
          </div>
          <p v-if="error" class="text-sm text-red-600">{{ error }}</p>
        </div>
        <div class="flex shrink-0 gap-2 border-t border-stone-200 px-4 py-3 dark:border-stone-800">
          <button type="button" class="flex-1 rounded-xl border px-4 py-2 text-sm dark:border-stone-700" @click="emit('close')">
            Cancel
          </button>
          <button
            type="submit"
            class="flex-1 rounded-xl bg-accent px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
            :disabled="saving"
          >
            {{ saving ? "Saving…" : "Save" }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
