<script setup lang="ts">
import { onMounted, ref } from "vue";
import { api } from "../api";
import type { Series } from "../types";
import SeriesCard from "../components/SeriesCard.vue";

const series = ref<Series[]>([]);

onMounted(async () => {
  series.value = (await api.listSeries()).series;
});
</script>

<template>
  <div>
    <header class="mb-4">
      <h1 class="text-2xl font-bold tracking-tight">Series</h1>
      <p class="text-sm text-stone-500">{{ series.length }} series</p>
    </header>
    <div v-if="series.length" class="grid grid-cols-3 gap-4 sm:grid-cols-4 md:grid-cols-6">
      <SeriesCard v-for="s in series" :key="s.id" :series="s" />
    </div>
    <p v-else class="text-sm text-stone-500">No series yet. Series tags in EPUBs, or select books on the Library page.</p>
  </div>
</template>
