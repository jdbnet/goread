<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { ArrowLeft } from "@lucide/vue";
import { api } from "../api";
import type { Series } from "../types";
import BookCard from "../components/BookCard.vue";

const props = defineProps<{ id: string }>();
const router = useRouter();
const series = ref<Series | null>(null);

onMounted(async () => {
  series.value = await api.getSeries(Number(props.id));
});
</script>

<template>
  <div v-if="series">
    <button type="button" class="mb-3 inline-flex items-center gap-1 text-sm text-stone-500" @click="router.push('/series')">
      <ArrowLeft :size="16" /> Series
    </button>
    <h1 class="mb-1 text-2xl font-bold tracking-tight">{{ series.name }}</h1>
    <p class="mb-6 text-sm text-stone-500">{{ series.book_count }} books</p>
    <div class="grid grid-cols-3 gap-4 sm:grid-cols-4 md:grid-cols-6">
      <div v-for="b in series.books" :key="b.id">
        <p v-if="b.sequence_number != null" class="mb-1 text-[10px] font-semibold uppercase tracking-wide text-stone-500">
          Book {{ b.sequence_number }}
        </p>
        <BookCard :book="b" />
      </div>
    </div>
  </div>
</template>
