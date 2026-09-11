import { api } from "./api";
import type { AuthStatus } from "./types";

export function safeRedirect(raw: unknown): string {
  if (typeof raw !== "string") return "/";
  if (!raw.startsWith("/") || raw.startsWith("//") || raw.startsWith("/\\")) return "/";
  if (raw === "/login" || raw.startsWith("/login?")) return "/";
  return raw;
}

let cached: AuthStatus | null = null;

export async function getAuthStatus(force = false): Promise<AuthStatus> {
  if (!force && cached) return cached;
  cached = await api.authStatus();
  return cached;
}

export function setAuthCache(status: AuthStatus | null): void {
  cached = status;
}
