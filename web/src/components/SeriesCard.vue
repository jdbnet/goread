<script setup lang="ts">
import { computed } from "vue";
import { RouterLink } from "vue-router";
import { Layers } from "@lucide/vue";
import type { Series } from "../types";

const props = defineProps<{ series: Series }>();

const covers = computed(() => (props.series.cover_urls ?? []).slice(0, 3));

function coverStyle(i: number, n: number): Record<string, string> {
  if (n <= 1) {
    return { left: "0", width: "100%", height: "100%", zIndex: "1" };
  }
  const step = n === 2 ? 16 : 14;
  return {
    left: `${i * step}%`,
    width: "78%",
    height: "92%",
    top: `${i * 4}%`,
    zIndex: String(n - i),
    transform: `rotate(${i * 3.5}deg)`,
    transformOrigin: "top left",
  };
}
</script>

<template>
  <RouterLink :to="`/series/${series.id}`" class="group block">
    <div class="relative aspect-[2/3] overflow-hidden rounded-xl bg-stone-200 dark:bg-stone-800">
      <template v-if="covers.length">
        <img
          v-for="(url, i) in covers"
          :key="url"
          :src="url"
          alt=""
          class="absolute rounded-md object-cover shadow-md ring-1 ring-black/10"
          :style="coverStyle(i, covers.length)"
        />
      </template>
      <div v-else class="flex h-full w-full flex-col items-center justify-center gap-2 px-3 text-stone-400">
        <Layers :size="32" />
        <span class="line-clamp-3 text-center text-xs font-medium text-stone-500">{{ series.name }}</span>
      </div>
    </div>
    <div class="mt-2">
      <h3 class="line-clamp-2 text-sm font-semibold leading-snug group-hover:text-accent-strong dark:group-hover:text-accent-soft">
        {{ series.name }}
      </h3>
      <p class="mt-0.5 text-xs text-stone-500 dark:text-stone-400">
        {{ series.book_count }} {{ series.book_count === 1 ? "book" : "books" }}
      </p>
    </div>
  </RouterLink>
</template>
