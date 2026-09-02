<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue";
import { Check, Layers, Plus, X } from "@lucide/vue";
import { api } from "../api";
import type { Series } from "../types";

const props = defineProps<{ bookIds: number[] }>();
const emit = defineEmits<{
  close: [];
  assigned: [name: string];
}>();

const series = ref<Series[]>([]);
const newName = ref("");
const saving = ref(false);
const error = ref("");
const filter = ref("");

const filtered = computed(() => {
  const q = filter.value.trim().toLowerCase();
  if (!q) return series.value;
  return series.value.filter((s) => s.name.toLowerCase().includes(q));
});

onMounted(async () => {
  document.addEventListener("keydown", onKey);
  try {
    series.value = (await api.listSeries()).series;
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Failed to load series";
  }
});

onUnmounted(() => {
  document.removeEventListener("keydown", onKey);
});

function onKey(ev: KeyboardEvent) {
  if (ev.key === "Escape" && !saving.value) {
    emit("close");
  }
}

async function assign(name: string) {
  const trimmed = name.trim();
  if (!trimmed || saving.value) return;
  saving.value = true;
  error.value = "";
  try {
    await api.assignSeries(props.bookIds, trimmed);
    emit("assigned", trimmed);
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Failed to assign series";
  } finally {
    saving.value = false;
  }
}

function create() {
  void assign(newName.value);
}

const fieldClass =
  "w-full rounded-xl border border-stone-300 bg-white px-3 py-2 text-sm dark:border-stone-700 dark:bg-stone-900";
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-end justify-center sm:items-center">
    <button type="button" class="absolute inset-0 bg-black/40" aria-label="Close" :disabled="saving" @click="emit('close')" />
    <div
      class="relative z-10 flex max-h-[92dvh] w-full flex-col overflow-hidden rounded-t-2xl bg-stone-50 dark:bg-stone-950 sm:max-w-lg sm:rounded-2xl"
      role="dialog"
      aria-labelledby="add-series-title"
    >
      <header class="flex shrink-0 items-center justify-between border-b border-stone-200 px-4 py-3 dark:border-stone-800">
        <h2 id="add-series-title" class="text-lg font-semibold">Add to series</h2>
        <button type="button" class="rounded-full p-1" :disabled="saving" @click="emit('close')">
          <X :size="20" />
        </button>
      </header>
      <div class="flex min-h-0 flex-1 flex-col">
        <p class="px-4 pt-3 text-sm text-stone-500">
          {{ bookIds.length }} {{ bookIds.length === 1 ? "book" : "books" }} selected
        </p>
        <div v-if="series.length > 6" class="px-4 pt-3">
          <input v-model="filter" type="search" placeholder="Filter series" :class="fieldClass" />
        </div>
        <div class="min-h-0 flex-1 overflow-y-auto px-4 py-3">
          <p v-if="!series.length" class="text-sm text-stone-500">No series yet. Create one below.</p>
          <p v-else-if="!filtered.length" class="text-sm text-stone-500">No matching series.</p>
          <ul v-else class="divide-y divide-stone-200 dark:divide-stone-800">
            <li v-for="s in filtered" :key="s.id">
              <button
                type="button"
                class="flex w-full items-center gap-3 py-3 text-left disabled:opacity-50"
                :disabled="saving"
                @click="assign(s.name)"
              >
                <span
                  class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-accent-muted text-accent-muted-fg dark:bg-accent-muted-dark dark:text-accent-muted-fg-dark"
                >
                  <Layers :size="20" />
                </span>
                <span class="min-w-0 flex-1">
                  <span class="block truncate font-semibold">{{ s.name }}</span>
                  <span class="text-xs text-stone-500">{{ s.book_count }} books</span>
                </span>
                <Check :size="18" class="shrink-0 text-stone-300" />
              </button>
            </li>
          </ul>
        </div>
        <form class="shrink-0 border-t border-stone-200 px-4 py-3 dark:border-stone-800" @submit.prevent="create">
          <label class="block space-y-1">
            <span class="text-xs font-medium text-stone-500">New series</span>
            <span class="flex gap-2">
              <input v-model="newName" :class="fieldClass" placeholder="Series name" :disabled="saving" />
              <button
                type="submit"
                class="inline-flex shrink-0 items-center gap-1 rounded-xl bg-accent px-3 py-2 text-sm font-semibold text-white disabled:opacity-50"
                :disabled="saving || !newName.trim()"
              >
                <Plus :size="16" />
                Create
              </button>
            </span>
          </label>
          <p v-if="error" class="mt-2 text-sm text-red-600">{{ error }}</p>
          <p v-if="saving" class="mt-2 text-sm text-stone-500">Adding to series…</p>
        </form>
      </div>
    </div>
  </div>
</template>
