<script setup lang="ts">
import { BookOpen, Check, Download } from "@lucide/vue";
import type { Book } from "../types";
import { downloadedIds } from "../offline/downloads";

defineProps<{ book: Book; compact?: boolean }>();
</script>

<template>
  <div
    class="relative overflow-hidden bg-stone-200 dark:bg-stone-800"
    :class="compact ? 'aspect-[2/3] rounded-lg' : 'aspect-[2/3] rounded-xl'"
  >
    <img
      v-if="book.has_cover && book.cover_url"
      :key="book.cover_url"
      :src="book.cover_url"
      :alt="book.title"
      class="h-full w-full object-cover"
      :class="book.completed_at ? 'opacity-70' : ''"
    />
    <div
      v-else
      class="flex h-full w-full flex-col items-center justify-center gap-2 px-3 text-stone-400"
      :class="book.completed_at ? 'opacity-70' : ''"
    >
      <BookOpen :size="compact ? 22 : 32" />
      <span class="line-clamp-3 text-center text-xs font-medium text-stone-500">{{ book.title }}</span>
    </div>
    <div
      v-if="downloadedIds.has(book.id)"
      class="absolute left-1.5 top-1.5 flex items-center justify-center rounded-full bg-black/60 text-white shadow-sm"
      :class="compact ? 'h-5 w-5' : 'h-7 w-7'"
      aria-label="Downloaded"
    >
      <Download :size="compact ? 12 : 16" :stroke-width="2.5" />
    </div>
    <div
      v-if="book.completed_at"
      class="absolute right-1.5 top-1.5 flex items-center justify-center rounded-full bg-accent text-white shadow-sm"
      :class="compact ? 'h-5 w-5' : 'h-7 w-7'"
      aria-label="Completed"
    >
      <Check :size="compact ? 12 : 16" :stroke-width="3" />
    </div>
    <div
      v-if="book.file_missing"
      class="absolute inset-x-0 bottom-0 bg-red-700/90 px-1 py-0.5 text-center text-[10px] font-semibold uppercase tracking-wide text-white"
    >
      Missing
    </div>
  </div>
</template>
