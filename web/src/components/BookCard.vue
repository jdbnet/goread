<script setup lang="ts">
import { RouterLink } from "vue-router";
import { Check } from "@lucide/vue";
import type { Book } from "../types";
import CoverImage from "./CoverImage.vue";
import ProgressBar from "./ProgressBar.vue";

defineProps<{ book: Book; selecting?: boolean; selected?: boolean }>();
const emit = defineEmits<{ toggle: [] }>();
</script>

<template>
  <component
    :is="selecting ? 'button' : RouterLink"
    :type="selecting ? 'button' : undefined"
    :to="selecting ? undefined : `/books/${book.id}`"
    class="group relative block w-full text-left"
    :aria-pressed="selecting ? selected : undefined"
    @click="selecting ? emit('toggle') : undefined"
  >
    <CoverImage :book="book" />
    <span
      v-if="selecting"
      class="absolute left-1.5 top-1.5 z-10 flex h-6 w-6 items-center justify-center rounded-full border-2 shadow-sm"
      :class="
        selected
          ? 'border-accent bg-accent text-white'
          : 'border-white bg-black/35 text-transparent'
      "
    >
      <Check :size="14" :stroke-width="3" />
    </span>
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
  </component>
</template>
