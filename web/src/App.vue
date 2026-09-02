<script setup lang="ts">
import { computed } from "vue";
import { RouterView, useRoute } from "vue-router";
import AppShell from "./components/AppShell.vue";

const route = useRoute();
const hideChrome = computed(() => Boolean(route.meta.hideChrome));
</script>

<template>
  <RouterView v-if="hideChrome" />
  <AppShell v-else />
</template>
