<script setup lang="ts">
defineProps<{ value: number }>();
</script>

<template>
  <div class="h-1.5 w-full overflow-hidden rounded-full bg-stone-200 dark:bg-stone-800">
    <div
      class="h-full rounded-full bg-accent-bar transition-[width] dark:bg-accent-bar-dark"
      :style="{ width: `${Math.min(100, Math.max(0, value))}%` }"
    />
  </div>
</template>
