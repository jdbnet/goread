<script setup lang="ts">
import { onMounted, ref } from "vue";
import { RouterLink } from "vue-router";
import { Layers } from "@lucide/vue";
import { api } from "../api";
import type { Series } from "../types";

const series = ref<Series[]>([]);

onMounted(async () => {
  series.value = (await api.listSeries()).series;
});
</script>

<template>
  <div>
    <h1 class="mb-4 text-2xl font-bold tracking-tight">Series</h1>
    <ul v-if="series.length" class="divide-y divide-stone-200 dark:divide-stone-800">
      <li v-for="s in series" :key="s.id">
        <RouterLink :to="`/series/${s.id}`" class="flex items-center gap-3 py-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-300">
            <Layers :size="20" />
          </div>
          <div class="min-w-0 flex-1">
            <p class="truncate font-semibold">{{ s.name }}</p>
            <p class="text-xs text-stone-500">{{ s.book_count }} books</p>
          </div>
        </RouterLink>
      </li>
    </ul>
    <p v-else class="text-sm text-stone-500">No series yet. Series tags in EPUBs, or assign one from a book page.</p>
  </div>
</template>
