<script setup lang="ts">
defineProps<{ value: number }>();
</script>

<template>
  <div class="h-1.5 w-full overflow-hidden rounded-full bg-stone-200 dark:bg-stone-800">
    <div
      class="h-full rounded-full bg-amber-600 transition-[width] dark:bg-amber-500"
      :style="{ width: `${Math.min(100, Math.max(0, value))}%` }"
    />
  </div>
</template>
