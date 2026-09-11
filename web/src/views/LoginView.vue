<script setup lang="ts">
import { ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { api } from "../api";
import { safeRedirect, setAuthCache } from "../auth";
import { loadAccent } from "../accent";

const route = useRoute();
const router = useRouter();
const username = ref("");
const password = ref("");
const error = ref("");
const submitting = ref(false);

async function submit() {
  submitting.value = true;
  error.value = "";
  try {
    const status = await api.login(username.value, password.value);
    setAuthCache(status);
    await loadAccent();
    await router.replace(safeRedirect(route.query.redirect));
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Login failed";
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <div class="flex min-h-dvh flex-col items-center justify-center px-4 py-10">
    <form
      class="w-full max-w-sm rounded-2xl border border-stone-200 bg-white p-6 shadow-sm dark:border-stone-800 dark:bg-stone-900"
      @submit.prevent="submit"
    >
      <img src="/logo.png" alt="" class="mx-auto h-16 w-16" width="64" height="64" />
      <h1 class="mt-4 text-center text-xl font-bold tracking-tight">eBook Reader</h1>
      <p class="mt-1 text-center text-sm text-stone-500">Sign in to continue</p>

      <label class="mt-6 block text-xs font-medium text-stone-500" for="login-username">Username</label>
      <input
        id="login-username"
        v-model="username"
        class="mt-1 w-full rounded-xl border border-stone-300 bg-white px-3 py-2 text-sm dark:border-stone-700 dark:bg-stone-950"
        name="username"
        autocomplete="username"
        autocapitalize="none"
        spellcheck="false"
        required
      />

      <label class="mt-3 block text-xs font-medium text-stone-500" for="login-password">Password</label>
      <input
        id="login-password"
        v-model="password"
        class="mt-1 w-full rounded-xl border border-stone-300 bg-white px-3 py-2 text-sm dark:border-stone-700 dark:bg-stone-950"
        type="password"
        name="password"
        autocomplete="current-password"
        required
      />

      <p v-if="error" class="mt-3 text-sm text-red-600">{{ error }}</p>

      <button
        type="submit"
        class="mt-5 w-full rounded-xl bg-accent px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-50"
        :disabled="submitting"
      >
        {{ submitting ? "Signing in..." : "Sign in" }}
      </button>
    </form>
  </div>
</template>
