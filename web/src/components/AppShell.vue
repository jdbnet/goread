<script setup lang="ts">
import { RouterLink, RouterView, useRoute } from "vue-router";
import { House, Library, Layers, ChartNoAxesColumn, Settings } from "@lucide/vue";

const route = useRoute();

const items = [
  { to: "/", name: "home", label: "Home", icon: House },
  { to: "/library", name: "library", label: "Library", icon: Library },
  { to: "/series", name: "series", label: "Series", icon: Layers },
  { to: "/stats", name: "stats", label: "Stats", icon: ChartNoAxesColumn },
  { to: "/settings", name: "settings", label: "Settings", icon: Settings },
] as const;

function active(name: string): boolean {
  if (name === "home") return route.name === "home";
  if (name === "library") return route.name === "library" || route.name === "book";
  if (name === "series") return route.name === "series" || route.name === "series-detail";
  return route.name === name;
}
</script>

<template>
  <div class="flex min-h-dvh flex-col">
    <main class="mx-auto w-full max-w-6xl flex-1 px-4 pb-24 pt-4 sm:px-6">
      <RouterView />
    </main>
    <nav
      class="safe-bottom fixed inset-x-0 bottom-0 z-30 border-t border-stone-200 bg-stone-50/95 backdrop-blur dark:border-stone-800 dark:bg-stone-950/95"
    >
      <ul class="mx-auto grid max-w-6xl grid-cols-5">
        <li v-for="item in items" :key="item.to">
          <RouterLink
            :to="item.to"
            class="flex flex-col items-center gap-1 py-2 text-[11px] font-medium"
            :class="active(item.name) ? 'text-accent dark:text-accent-soft' : 'text-stone-500 dark:text-stone-400'"
          >
            <component :is="item.icon" :size="22" :stroke-width="active(item.name) ? 2.4 : 1.8" />
            {{ item.label }}
          </RouterLink>
        </li>
      </ul>
    </nav>
  </div>
</template>
