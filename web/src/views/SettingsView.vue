<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { Check } from "@lucide/vue";
import { api } from "../api";
import { ACCENTS, applyAccent } from "../accent";
import { getAuthStatus, setAuthCache } from "../auth";
import type { AccentId, AuthStatus, BackupFile, BackupSettings, Settings } from "../types";
import { online } from "../offline/status";

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

const backupSettings = ref<BackupSettings | null>(null);
const backups = ref<BackupFile[]>([]);
const backupSaving = ref(false);
const backupRunning = ref(false);
const backupError = ref("");
const backupMessage = ref("");
const restoreInput = ref<HTMLInputElement | null>(null);
const restoring = ref(false);
const restoreMessage = ref("");

const fieldClass =
  "mt-1 w-full rounded-xl border border-stone-300 bg-white px-3 py-2 text-sm dark:border-stone-700 dark:bg-stone-900";

onMounted(async () => {
  try {
    const [s, a, b] = await Promise.all([api.settings(), getAuthStatus(true), api.getBackup()]);
    settings.value = s;
    auth.value = a;
    username.value = a.username;
    applyAccent(s.accent);
    backupSettings.value = b.settings;
    backups.value = b.backups;
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Failed to load settings";
  }
});

function formatBytes(size: number): string {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

function formatBackupTime(value: string | null): string {
  if (!value) return "Never";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

function hourLabel(hour: number): string {
  const date = new Date();
  date.setHours(hour, 0, 0, 0);
  return date.toLocaleTimeString(undefined, { hour: "numeric", minute: "2-digit" });
}

async function saveBackupSettings() {
  if (!backupSettings.value || backupSaving.value) return;
  backupSaving.value = true;
  backupError.value = "";
  backupMessage.value = "";
  try {
    const res = await api.saveBackupSettings(backupSettings.value);
    backupSettings.value = res.settings;
    backups.value = res.backups;
    backupMessage.value = "Backup settings saved";
  } catch (e) {
    backupError.value = e instanceof Error ? e.message : "Failed to save backup settings";
  } finally {
    backupSaving.value = false;
  }
}

async function runBackupNow() {
  if (backupRunning.value) return;
  backupRunning.value = true;
  backupError.value = "";
  backupMessage.value = "";
  try {
    const res = await api.createBackup();
    backupSettings.value = res.settings;
    backups.value = res.backups;
    backupMessage.value = "Backup created";
  } catch (e) {
    backupError.value = e instanceof Error ? e.message : "Failed to create backup";
  } finally {
    backupRunning.value = false;
  }
}

async function downloadBackupFile(filename: string) {
  backupError.value = "";
  try {
    await api.downloadBackup(filename);
  } catch (e) {
    backupError.value = e instanceof Error ? e.message : "Failed to download backup";
  }
}

async function removeBackupFile(filename: string) {
  if (!window.confirm(`Delete backup ${filename}?`)) return;
  backupError.value = "";
  backupMessage.value = "";
  try {
    const res = await api.deleteBackup(filename);
    backupSettings.value = res.settings;
    backups.value = res.backups;
    backupMessage.value = "Backup deleted";
  } catch (e) {
    backupError.value = e instanceof Error ? e.message : "Failed to delete backup";
  }
}

function pickRestoreFile() {
  restoreInput.value?.click();
}

async function onRestoreSelected(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = "";
  if (!file) return;
  if (!file.name.toLowerCase().endsWith(".zip")) {
    backupError.value = "Please choose a .zip backup file";
    return;
  }
  if (
    !window.confirm(
      "Restore this backup? All current reading progress, settings, and covers will be replaced. The server will restart.",
    )
  ) {
    return;
  }
  restoring.value = true;
  backupError.value = "";
  restoreMessage.value = "";
  try {
    await api.restoreBackup(file);
    restoreMessage.value = "Restore complete. The server is restarting; this page will reload shortly.";
    await waitForServer();
    window.location.reload();
  } catch (e) {
    backupError.value = e instanceof Error ? e.message : "Failed to restore backup";
    restoring.value = false;
  }
}

async function waitForServer() {
  for (let i = 0; i < 60; i++) {
    await new Promise((resolve) => setTimeout(resolve, 1000));
    try {
      const res = await fetch("/healthz");
      if (res.ok) return;
    } catch {
      /* server still restarting */
    }
  }
}

async function choose(id: AccentId) {
  if (!settings.value || saving.value || settings.value.accent === id || !online.value) return;
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
          :disabled="loggingOut || !online"
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
            :disabled="authSaving || !online"
          >
            {{ auth?.enabled ? "Update login" : "Turn on login" }}
          </button>
          <button
            v-if="auth?.enabled"
            type="button"
            class="rounded-xl border border-stone-300 px-4 py-2 text-sm font-medium dark:border-stone-700"
            :disabled="authSaving || !online"
            @click="disableLogin"
          >
            Turn off login
          </button>
        </div>
      </form>
    </section>

    <section class="mt-10">
      <h2 class="text-lg font-semibold">Backup &amp; restore</h2>
      <p class="mt-1 text-sm text-stone-500">
        Back up reading progress, settings, and covers. EPUB files in your library are not included.
      </p>

      <form v-if="backupSettings" class="mt-4 max-w-2xl space-y-4" @submit.prevent="saveBackupSettings">
        <label class="flex items-center gap-2 text-sm">
          <input v-model="backupSettings.enabled" type="checkbox" class="rounded border-stone-300" :disabled="!online" />
          Enable automatic backups
        </label>

        <div v-if="backupSettings.enabled" class="space-y-3 rounded-xl border border-stone-200 p-4 dark:border-stone-800">
          <div>
            <p class="text-xs font-medium text-stone-500">Schedule</p>
            <div class="mt-2 flex flex-wrap gap-4 text-sm">
              <label class="flex items-center gap-2">
                <input
                  v-model="backupSettings.schedule_mode"
                  type="radio"
                  value="interval"
                  name="backup-schedule"
                  :disabled="!online"
                />
                Every N hours
              </label>
              <label class="flex items-center gap-2">
                <input
                  v-model="backupSettings.schedule_mode"
                  type="radio"
                  value="daily"
                  name="backup-schedule"
                  :disabled="!online"
                />
                Daily at a set hour
              </label>
            </div>
          </div>

          <div v-if="backupSettings.schedule_mode === 'interval'">
            <label class="text-xs font-medium text-stone-500" for="backup-interval">Interval (hours)</label>
            <input
              id="backup-interval"
              v-model.number="backupSettings.interval_hours"
              :class="fieldClass"
              type="number"
              min="1"
              max="720"
              :disabled="!online"
            />
          </div>

          <div v-else>
            <label class="text-xs font-medium text-stone-500" for="backup-hour">Time of day (server timezone)</label>
            <select id="backup-hour" v-model.number="backupSettings.daily_hour" :class="fieldClass" :disabled="!online">
              <option v-for="h in 24" :key="h - 1" :value="h - 1">{{ hourLabel(h - 1) }}</option>
            </select>
          </div>

          <div>
            <label class="text-xs font-medium text-stone-500" for="backup-retention">Keep last N backups</label>
            <input
              id="backup-retention"
              v-model.number="backupSettings.retention_count"
              :class="fieldClass"
              type="number"
              min="1"
              max="100"
              :disabled="!online"
            />
          </div>
        </div>

        <p class="text-sm text-stone-500">Last backup: {{ formatBackupTime(backupSettings.last_run_at) }}</p>

        <div class="flex flex-wrap gap-2">
          <button
            type="submit"
            class="rounded-xl bg-accent px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
            :disabled="backupSaving || !online"
          >
            {{ backupSaving ? "Saving..." : "Save backup settings" }}
          </button>
          <button
            type="button"
            class="rounded-xl border border-stone-300 px-4 py-2 text-sm font-medium dark:border-stone-700"
            :disabled="backupRunning || !online"
            @click="runBackupNow"
          >
            {{ backupRunning ? "Backing up..." : "Backup now" }}
          </button>
          <button
            type="button"
            class="rounded-xl border border-stone-300 px-4 py-2 text-sm font-medium dark:border-stone-700"
            :disabled="restoring || !online"
            @click="pickRestoreFile"
          >
            {{ restoring ? "Restoring..." : "Restore from file" }}
          </button>
          <input ref="restoreInput" type="file" accept=".zip,application/zip" class="hidden" @change="onRestoreSelected" />
        </div>

        <p v-if="backupError" class="text-sm text-red-600">{{ backupError }}</p>
        <p v-else-if="restoreMessage" class="text-sm text-emerald-700 dark:text-emerald-400">{{ restoreMessage }}</p>
        <p v-else-if="backupMessage" class="text-sm text-emerald-700 dark:text-emerald-400">{{ backupMessage }}</p>

        <div v-if="backups.length" class="overflow-hidden rounded-xl border border-stone-200 dark:border-stone-800">
          <table class="w-full text-left text-sm">
            <thead class="bg-stone-100 text-xs uppercase tracking-wide text-stone-500 dark:bg-stone-900">
              <tr>
                <th class="px-3 py-2 font-medium">File</th>
                <th class="px-3 py-2 font-medium">Size</th>
                <th class="px-3 py-2 font-medium">Created</th>
                <th class="px-3 py-2 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="file in backups"
                :key="file.filename"
                class="border-t border-stone-200 dark:border-stone-800"
              >
                <td class="px-3 py-2 font-mono text-xs">{{ file.filename }}</td>
                <td class="px-3 py-2">{{ formatBytes(file.size) }}</td>
                <td class="px-3 py-2">{{ formatBackupTime(file.created_at) }}</td>
                <td class="px-3 py-2">
                  <div class="flex flex-wrap gap-2">
                    <button
                      type="button"
                      class="text-accent hover:underline"
                      :disabled="!online"
                      @click="downloadBackupFile(file.filename)"
                    >
                      Download
                    </button>
                    <button
                      type="button"
                      class="text-red-600 hover:underline"
                      :disabled="!online"
                      @click="removeBackupFile(file.filename)"
                    >
                      Delete
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-else class="text-sm text-stone-500">No backups stored on the server yet.</p>
      </form>
    </section>
  </div>
</template>
