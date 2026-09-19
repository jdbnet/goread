import type {
  AuthStatus,
  BackupResponse,
  BackupSettings,
  Book,
  BookListResponse,
  MetadataHit,
  Progress,
  Series,
  Settings,
  Stats,
} from "./types";
import { ApiError, req } from "./http";
import { overlayBook, overlayBooks, overlayContinue, overlayStats, recordProgressPost, rememberBook, rememberProgress, type ProgressWrite } from "./offline/progress";

export { ApiError };

function overlayBookList(res: BookListResponse): BookListResponse {
  for (const book of res.books) {
    rememberBook(book);
  }
  return { ...res, books: overlayBooks(res.books) };
}

export const api = {
  authStatus(): Promise<AuthStatus> {
    return req("/api/v1/auth/status");
  },
  login(username: string, password: string): Promise<AuthStatus> {
    return req("/api/v1/auth/login", { method: "POST", body: JSON.stringify({ username, password }) });
  },
  logout(): Promise<AuthStatus> {
    return req("/api/v1/auth/logout", { method: "POST" });
  },
  saveCredentials(body: {
    username: string;
    password: string;
    current_password?: string;
  }): Promise<AuthStatus> {
    return req("/api/v1/auth/credentials", { method: "PUT", body: JSON.stringify(body) });
  },
  disableAuth(password: string): Promise<AuthStatus> {
    return req("/api/v1/auth/disable", { method: "POST", body: JSON.stringify({ password }) });
  },
  async listBooks(params: Record<string, string | number | undefined> = {}): Promise<BookListResponse> {
    const q = new URLSearchParams();
    for (const [k, v] of Object.entries(params)) {
      if (v !== undefined && v !== "") q.set(k, String(v));
    }
    const qs = q.toString();
    const res = await req<BookListResponse>(`/api/v1/library/books${qs ? `?${qs}` : ""}`);
    return overlayBookList(res);
  },
  async continueReading(): Promise<BookListResponse> {
    const res = await req<BookListResponse>("/api/v1/library/books?continue=1&limit=20");
    for (const book of res.books) {
      rememberBook(book);
    }
    return { ...res, books: overlayContinue(res.books) };
  },
  authors(): Promise<{ authors: string[] }> {
    return req("/api/v1/library/authors");
  },
  listSeries(): Promise<{ series: Series[] }> {
    return req("/api/v1/library/series");
  },
  async getSeries(id: number): Promise<Series> {
    const series = await req<Series>(`/api/v1/series/${id}`);
    if (series.books) {
      for (const book of series.books) {
        rememberBook(book);
      }
      series.books = overlayBooks(series.books);
    }
    return series;
  },
  async getBook(id: number): Promise<Book> {
    const book = await req<Book>(`/api/v1/books/${id}`);
    rememberBook(book);
    return overlayBook(book);
  },
  async applyMetadata(id: number, body: Partial<MetadataHit>): Promise<Book> {
    const book = await req<Book>(`/api/v1/books/${id}/metadata`, { method: "POST", body: JSON.stringify(body) });
    rememberBook(book);
    return overlayBook(book);
  },
  async uploadCover(id: number, file: File): Promise<Book> {
    const res = await fetch(`/api/v1/books/${id}/cover`, { method: "POST", body: formData(file), credentials: "same-origin" });
    if (res.status === 401) {
      if (window.location.pathname !== "/login") {
        const next = window.location.pathname + window.location.search;
        window.location.assign(`/login?redirect=${encodeURIComponent(next)}`);
      }
    }
    if (!res.ok) {
      throw new ApiError(res.status, await readCoverError(res));
    }
    const book = (await res.json()) as Book;
    rememberBook(book);
    return overlayBook(book);
  },
  async setSeries(id: number, name: string, sequence_number: number | null): Promise<Book> {
    const book = await req<Book>(`/api/v1/books/${id}/series`, {
      method: "PUT",
      body: JSON.stringify({ name, sequence_number }),
    });
    rememberBook(book);
    return overlayBook(book);
  },
  assignSeries(book_ids: number[], name: string): Promise<{ count: number; name: string }> {
    return req("/api/v1/library/series/assign", {
      method: "POST",
      body: JSON.stringify({ book_ids, name }),
    });
  },
  scan(): Promise<{ started: boolean; running: boolean }> {
    return req("/api/v1/scan", { method: "POST" });
  },
  scanStatus(): Promise<{ running: boolean }> {
    return req("/api/v1/scan");
  },
  searchMetadata(q: string): Promise<{ results: MetadataHit[] }> {
    return req(`/api/v1/metadata/search?q=${encodeURIComponent(q)}`);
  },
  async getProgress(id: number): Promise<Progress> {
    const p = await req<Progress>(`/api/v1/progress/${id}`);
    rememberProgress(p);
    return p;
  },
  postProgress(id: number, body: ProgressWrite): Promise<Progress> {
    return recordProgressPost(id, body);
  },
  async stats(): Promise<Stats> {
    const s = await req<Stats>("/api/v1/stats");
    return overlayStats(s);
  },
  settings(): Promise<Settings> {
    return req("/api/v1/settings");
  },
  saveSettings(s: Settings): Promise<Settings> {
    return req("/api/v1/settings", { method: "PUT", body: JSON.stringify(s) });
  },
  getBackup(): Promise<BackupResponse> {
    return req("/api/v1/backup");
  },
  saveBackupSettings(settings: BackupSettings): Promise<BackupResponse> {
    return req("/api/v1/backup", { method: "PUT", body: JSON.stringify(settings) });
  },
  createBackup(): Promise<BackupResponse> {
    return req("/api/v1/backup", { method: "POST" });
  },
  async downloadBackup(filename: string): Promise<void> {
    const res = await fetch(`/api/v1/backup/${encodeURIComponent(filename)}`, { credentials: "same-origin" });
    if (res.status === 401) {
      if (window.location.pathname !== "/login") {
        const next = window.location.pathname + window.location.search;
        window.location.assign(`/login?redirect=${encodeURIComponent(next)}`);
      }
      return;
    }
    if (!res.ok) {
      throw new ApiError(res.status, await readBackupError(res));
    }
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  },
  deleteBackup(filename: string): Promise<BackupResponse> {
    return req(`/api/v1/backup/${encodeURIComponent(filename)}`, { method: "DELETE" });
  },
  async restoreBackup(file: File): Promise<void> {
    const form = new FormData();
    form.append("backup", file);
    const res = await fetch("/api/v1/restore", { method: "POST", body: form, credentials: "same-origin" });
    if (res.status === 401) {
      if (window.location.pathname !== "/login") {
        const next = window.location.pathname + window.location.search;
        window.location.assign(`/login?redirect=${encodeURIComponent(next)}`);
      }
      return;
    }
    if (!res.ok) {
      throw new ApiError(res.status, await readBackupError(res));
    }
  },
};

async function readBackupError(res: Response): Promise<string> {
  let message = res.statusText;
  try {
    const body = (await res.json()) as { error?: string };
    if (body.error) message = body.error;
  } catch {
    /* ignore */
  }
  return message;
}

async function readCoverError(res: Response): Promise<string> {
  let message = res.statusText;
  try {
    const body = (await res.json()) as { error?: string };
    if (body.error) message = body.error;
  } catch {
    /* ignore */
  }
  return message;
}

function formData(file: File): FormData {
  const form = new FormData();
  form.append("cover", file);
  return form;
}

export function formatDuration(totalSeconds: number): string {
  const s = Math.max(0, Math.floor(totalSeconds));
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  if (h > 0) return `${h}h ${m}m`;
  if (m > 0) return `${m}m`;
  return `${s}s`;
}

export function stripHtml(html: string): string {
  const tmp = document.createElement("div");
  tmp.innerHTML = html;
  return (tmp.textContent || tmp.innerText || "").trim();
}

export function formatPubDate(value: string): string {
  const trimmed = value.trim();
  if (!trimmed) return "";
  if (/^\d{4}$/.test(trimmed)) return trimmed;
  const date = new Date(trimmed);
  if (Number.isNaN(date.getTime())) return trimmed;
  return date.toLocaleDateString(undefined, { day: "numeric", month: "long", year: "numeric" });
}
