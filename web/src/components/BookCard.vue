<script setup lang="ts">
import { RouterLink } from "vue-router";
import type { Book } from "../types";
import CoverImage from "./CoverImage.vue";
import ProgressBar from "./ProgressBar.vue";

defineProps<{ book: Book }>();
</script>

<template>
  <RouterLink :to="`/books/${book.id}`" class="group block">
    <CoverImage :book="book" />
    <div class="mt-2">
      <h3 class="line-clamp-2 text-sm font-semibold leading-snug group-hover:text-accent-strong dark:group-hover:text-accent-soft">
        {{ book.title }}
      </h3>
      <p class="mt-0.5 truncate text-xs text-stone-500 dark:text-stone-400">{{ book.author }}</p>
      <div v-if="book.percent_completed > 0" class="mt-1.5">
        <ProgressBar :value="book.percent_completed" />
        <p class="mt-0.5 text-[10px] text-stone-500">{{ Math.round(book.percent_completed) }}%</p>
      </div>
    </div>
  </RouterLink>
</template>
