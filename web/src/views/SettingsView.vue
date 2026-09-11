<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { Check } from "@lucide/vue";
import { api } from "../api";
import { ACCENTS, applyAccent } from "../accent";
import { getAuthStatus, setAuthCache } from "../auth";
import type { AccentId, AuthStatus, Settings } from "../types";

const router = useRouter();
const settings = ref<Settings | null>(null);
const auth = ref<AuthStatus | null>(null);
const saving = ref(false);
const error = ref("");

const username = ref("");
const password = ref("");
const confirm = ref("");
const currentPassword = ref("");
const authSaving = ref(false);
const authError = ref("");
const authMessage = ref("");
const loggingOut = ref(false);

const fieldClass =
  "mt-1 w-full rounded-xl border border-stone-300 bg-white px-3 py-2 text-sm dark:border-stone-700 dark:bg-stone-900";

onMounted(async () => {
  try {
    const [s, a] = await Promise.all([api.settings(), getAuthStatus(true)]);
    settings.value = s;
    auth.value = a;
    username.value = a.username;
    applyAccent(s.accent);
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
    if (auth.value) {
      auth.value = { ...auth.value, accent: id };
      setAuthCache(auth.value);
    }
  } catch (e) {
    applyAccent(settings.value.accent);
    error.value = e instanceof Error ? e.message : "Failed to save";
  } finally {
    saving.value = false;
  }
}

async function saveLogin() {
  if (authSaving.value) return;
  authError.value = "";
  authMessage.value = "";
  const name = username.value.trim();
  if (!name) {
    authError.value = "Username is required";
    return;
  }
  if (password.value && password.value.length < 8) {
    authError.value = "Password must be at least 8 characters";
    return;
  }
  if (password.value !== confirm.value) {
    authError.value = "Passwords do not match";
    return;
  }
  if (!auth.value?.enabled && !password.value) {
    authError.value = "Password is required";
    return;
  }
  if (auth.value?.enabled && !currentPassword.value) {
    authError.value = "Current password is required";
    return;
  }
  authSaving.value = true;
  try {
    const status = await api.saveCredentials({
      username: name,
      password: password.value,
      current_password: currentPassword.value || undefined,
    });
    setAuthCache(status);
    auth.value = status;
    username.value = status.username;
    password.value = "";
    confirm.value = "";
    currentPassword.value = "";
    authMessage.value = status.enabled ? "Login saved" : "";
  } catch (e) {
    authError.value = e instanceof Error ? e.message : "Failed to save login";
  } finally {
    authSaving.value = false;
  }
}

async function disableLogin() {
  if (authSaving.value) return;
  authError.value = "";
  authMessage.value = "";
  if (!currentPassword.value) {
    authError.value = "Current password is required to turn login off";
    return;
  }
  authSaving.value = true;
  try {
    const status = await api.disableAuth(currentPassword.value);
    setAuthCache(status);
    auth.value = status;
    password.value = "";
    confirm.value = "";
    currentPassword.value = "";
    authMessage.value = "Login is off";
  } catch (e) {
    authError.value = e instanceof Error ? e.message : "Failed to turn login off";
  } finally {
    authSaving.value = false;
  }
}

async function logout() {
  if (loggingOut.value) return;
  loggingOut.value = true;
  authError.value = "";
  try {
    const status = await api.logout();
    setAuthCache(status);
    await router.push({ name: "login" });
  } catch (e) {
    authError.value = e instanceof Error ? e.message : "Failed to log out";
    loggingOut.value = false;
  }
}
</script>

<template>
  <div>
    <h1 class="mb-5 text-2xl font-bold tracking-tight">Settings</h1>
    <section>
      <h2 class="text-lg font-semibold">Accent colour</h2>
      <p class="mt-1 text-sm text-stone-500">Used for buttons, progress, and highlights. Emerald is the default.</p>
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

    <section class="mt-10">
      <div class="flex items-start justify-between gap-3">
        <div>
          <h2 class="text-lg font-semibold">Login</h2>
          <p class="mt-1 text-sm text-stone-500">
            Optional. Once a username and password are saved, this app requires them to open.
          </p>
        </div>
        <button
          v-if="auth?.enabled && auth.authenticated"
          type="button"
          class="shrink-0 rounded-full border border-stone-300 bg-white px-3 py-1.5 text-sm font-medium dark:border-stone-700 dark:bg-stone-900"
          :disabled="loggingOut"
          @click="logout"
        >
          {{ loggingOut ? "Signing out..." : "Sign out" }}
        </button>
      </div>

      <form class="mt-4 max-w-md space-y-3" @submit.prevent="saveLogin">
        <div>
          <label class="text-xs font-medium text-stone-500" for="settings-username">Username</label>
          <input
            id="settings-username"
            v-model="username"
            :class="fieldClass"
            name="username"
            autocomplete="username"
            autocapitalize="none"
            spellcheck="false"
            required
          />
        </div>
        <div>
          <label class="text-xs font-medium text-stone-500" for="settings-password">
            {{ auth?.enabled ? "New password" : "Password" }}
          </label>
          <input
            id="settings-password"
            v-model="password"
            :class="fieldClass"
            type="password"
            name="password"
            autocomplete="new-password"
            :placeholder="auth?.enabled ? 'Leave blank to keep the current password' : ''"
            :required="!auth?.enabled"
            :minlength="password ? 8 : undefined"
          />
        </div>
        <div>
          <label class="text-xs font-medium text-stone-500" for="settings-confirm">Confirm password</label>
          <input
            id="settings-confirm"
            v-model="confirm"
            :class="fieldClass"
            type="password"
            name="confirm"
            autocomplete="new-password"
            :required="!auth?.enabled || password.length > 0"
          />
        </div>
        <div v-if="auth?.enabled">
          <label class="text-xs font-medium text-stone-500" for="settings-current">Current password</label>
          <input
            id="settings-current"
            v-model="currentPassword"
            :class="fieldClass"
            type="password"
            name="current_password"
            autocomplete="current-password"
            required
          />
        </div>
        <p v-if="authError" class="text-sm text-red-600">{{ authError }}</p>
        <p v-else-if="authMessage" class="text-sm text-emerald-700 dark:text-emerald-400">{{ authMessage }}</p>
        <div class="flex flex-wrap gap-2 pt-1">
          <button
            type="submit"
            class="rounded-xl bg-accent px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
            :disabled="authSaving"
          >
            {{ auth?.enabled ? "Update login" : "Turn on login" }}
          </button>
          <button
            v-if="auth?.enabled"
            type="button"
            class="rounded-xl border border-stone-300 px-4 py-2 text-sm font-medium dark:border-stone-700"
            :disabled="authSaving"
            @click="disableLogin"
          >
            Turn off login
          </button>
        </div>
      </form>
    </section>
  </div>
</template>
