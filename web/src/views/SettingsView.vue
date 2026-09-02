<script setup lang="ts">
import { onMounted, ref } from "vue";
import { Check } from "@lucide/vue";
import { api } from "../api";
import { ACCENTS, applyAccent } from "../accent";
import type { AccentId, Settings } from "../types";

const settings = ref<Settings | null>(null);
const saving = ref(false);
const error = ref("");

onMounted(async () => {
  try {
    settings.value = await api.settings();
    applyAccent(settings.value.accent);
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Failed to load settings";
  }
});

async function choose(id: AccentId) {
  if (!settings.value || saving.value || settings.value.accent === id) return;
  saving.value = true;
  error.value = "";
  applyAccent(id);
  try {
    settings.value = await api.saveSettings({ ...settings.value, accent: id });
  } catch (e) {
    applyAccent(settings.value.accent);
    error.value = e instanceof Error ? e.message : "Failed to save";
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <div>
    <h1 class="mb-5 text-2xl font-bold tracking-tight">Settings</h1>
    <section>
      <h2 class="text-lg font-semibold">Accent colour</h2>
      <p class="mt-1 text-sm text-stone-500">Used for buttons, progress, and highlights. Amber is the default.</p>
      <ul class="mt-4 grid grid-cols-5 gap-3 sm:grid-cols-10">
        <li v-for="c in ACCENTS" :key="c.id">
          <button
            type="button"
            class="flex w-full flex-col items-center gap-1.5"
            :aria-pressed="settings?.accent === c.id"
            :aria-label="c.label"
            @click="choose(c.id)"
          >
            <span
              class="relative flex h-10 w-10 items-center justify-center rounded-full ring-2 ring-offset-2 ring-offset-stone-50 dark:ring-offset-stone-950"
              :class="settings?.accent === c.id ? 'ring-stone-900 dark:ring-stone-100' : 'ring-transparent'"
              :style="{ background: c.accent }"
            >
              <Check v-if="settings?.accent === c.id" :size="18" class="text-white" />
            </span>
            <span class="text-[11px] font-medium text-stone-600 dark:text-stone-400">{{ c.label }}</span>
          </button>
        </li>
      </ul>
      <p v-if="error" class="mt-3 text-sm text-red-600">{{ error }}</p>
    </section>
  </div>
</template>
