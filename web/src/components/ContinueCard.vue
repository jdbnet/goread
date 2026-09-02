<script setup lang="ts">
import { RouterLink } from "vue-router";
import type { Book } from "../types";
import CoverImage from "./CoverImage.vue";
import ProgressBar from "./ProgressBar.vue";

defineProps<{ book: Book }>();
</script>

<template>
  <RouterLink
    :to="book.file_missing ? `/books/${book.id}` : `/read/${book.id}`"
    class="flex min-w-[9.5rem] max-w-[9.5rem] shrink-0 flex-col"
  >
    <CoverImage :book="book" />
    <h3 class="mt-2 line-clamp-2 text-sm font-semibold leading-snug">{{ book.title }}</h3>
    <div class="mt-1.5">
      <ProgressBar :value="book.percent_completed" />
      <p class="mt-0.5 text-[10px] text-stone-500">{{ Math.round(book.percent_completed) }}%</p>
    </div>
  </RouterLink>
</template>
