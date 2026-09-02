<script setup lang="ts">
import { computed, onMounted } from "vue";
import { RouterView, useRoute } from "vue-router";
import AppShell from "./components/AppShell.vue";
import { loadAccent } from "./accent";

const route = useRoute();
const hideChrome = computed(() => Boolean(route.meta.hideChrome));

onMounted(() => {
  void loadAccent();
});
</script>

<template>
  <RouterView v-if="hideChrome" />
  <AppShell v-else />
</template>
