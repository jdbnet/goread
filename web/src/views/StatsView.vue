<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { BookCheck, Flame, Clock3 } from "@lucide/vue";
import { api, formatDuration } from "../api";
import type { Stats } from "../types";

const stats = ref<Stats | null>(null);

onMounted(async () => {
  stats.value = await api.stats();
});

const maxDay = computed(() => {
  if (!stats.value?.daily.length) return 1;
  return Math.max(1, ...stats.value.daily.map((d) => d.time_read_seconds));
});

const last14 = computed(() => {
  if (!stats.value) return [];
  const by = new Map(stats.value.daily.map((d) => [d.day, d.time_read_seconds]));
  const days: { day: string; seconds: number }[] = [];
  const now = new Date();
  for (let i = 13; i >= 0; i--) {
    const d = new Date(now);
    d.setDate(now.getDate() - i);
    const key = d.toISOString().slice(0, 10);
    days.push({ day: key, seconds: by.get(key) ?? 0 });
  }
  return days;
});
</script>

<template>
  <div v-if="stats">
    <h1 class="mb-5 text-2xl font-bold tracking-tight">Analytics</h1>
    <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
      <div class="rounded-2xl border border-stone-200 bg-white p-4 dark:border-stone-800 dark:bg-stone-900">
        <Clock3 class="mb-2 text-amber-700 dark:text-amber-400" :size="22" />
        <p class="text-xs font-medium uppercase tracking-wide text-stone-500">Time read</p>
        <p class="mt-1 text-2xl font-bold">{{ formatDuration(stats.total_time_read_seconds) }}</p>
        <p class="text-xs text-stone-500">{{ formatDuration(stats.this_week_seconds) }} this week</p>
      </div>
      <div class="rounded-2xl border border-stone-200 bg-white p-4 dark:border-stone-800 dark:bg-stone-900">
        <BookCheck class="mb-2 text-amber-700 dark:text-amber-400" :size="22" />
        <p class="text-xs font-medium uppercase tracking-wide text-stone-500">Completed</p>
        <p class="mt-1 text-2xl font-bold">{{ stats.completed_books }}</p>
      </div>
      <div class="rounded-2xl border border-stone-200 bg-white p-4 dark:border-stone-800 dark:bg-stone-900">
        <Flame class="mb-2 text-amber-700 dark:text-amber-400" :size="22" />
        <p class="text-xs font-medium uppercase tracking-wide text-stone-500">Streak</p>
        <p class="mt-1 text-2xl font-bold">{{ stats.streak_days }} days</p>
      </div>
    </div>

    <h2 class="mb-3 mt-8 text-lg font-semibold">Last 14 days</h2>
    <div class="flex h-32 items-end gap-1.5">
      <div v-for="d in last14" :key="d.day" class="flex flex-1 flex-col items-center gap-1">
        <div
          class="w-full rounded-t bg-amber-600 dark:bg-amber-500"
          :style="{ height: `${Math.max(d.seconds ? 8 : 2, (d.seconds / maxDay) * 100)}%` }"
          :title="`${d.day}: ${formatDuration(d.seconds)}`"
        />
      </div>
    </div>
  </div>
</template>
