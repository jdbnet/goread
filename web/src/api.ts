import type { AuthStatus, Book, BookListResponse, MetadataHit, Progress, Series, Settings, Stats } from "./types";

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

function redirectToLogin(): void {
  if (window.location.pathname === "/login") return;
  const next = window.location.pathname + window.location.search;
  window.location.assign(`/login?redirect=${encodeURIComponent(next)}`);
}

async function readError(res: Response): Promise<string> {
  let message = res.statusText;
  try {
    const body = (await res.json()) as { error?: string };
    if (body.error) message = body.error;
  } catch {
    /* ignore */
  }
  return message;
}

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    credentials: "same-origin",
    ...init,
    headers: {
      Accept: "application/json",
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
      ...init?.headers,
    },
  });
  if (res.status === 401 && path !== "/api/v1/auth/login" && path !== "/api/v1/auth/status") {
    redirectToLogin();
  }
  if (!res.ok) {
    throw new ApiError(res.status, await readError(res));
  }
  return (await res.json()) as T;
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
  listBooks(params: Record<string, string | number | undefined> = {}): Promise<BookListResponse> {
    const q = new URLSearchParams();
    for (const [k, v] of Object.entries(params)) {
      if (v !== undefined && v !== "") q.set(k, String(v));
    }
    const qs = q.toString();
    return req(`/api/v1/library/books${qs ? `?${qs}` : ""}`);
  },
  continueReading(): Promise<BookListResponse> {
    return req("/api/v1/library/books?continue=1&limit=20");
  },
  authors(): Promise<{ authors: string[] }> {
    return req("/api/v1/library/authors");
  },
  listSeries(): Promise<{ series: Series[] }> {
    return req("/api/v1/library/series");
  },
  getSeries(id: number): Promise<Series> {
    return req(`/api/v1/series/${id}`);
  },
  getBook(id: number): Promise<Book> {
    return req(`/api/v1/books/${id}`);
  },
  applyMetadata(id: number, body: Partial<MetadataHit>): Promise<Book> {
    return req(`/api/v1/books/${id}/metadata`, { method: "POST", body: JSON.stringify(body) });
  },
  async uploadCover(id: number, file: File): Promise<Book> {
    const res = await fetch(`/api/v1/books/${id}/cover`, { method: "POST", body: formData(file), credentials: "same-origin" });
    if (res.status === 401) {
      redirectToLogin();
    }
    if (!res.ok) {
      throw new ApiError(res.status, await readError(res));
    }
    return (await res.json()) as Book;
  },
  setSeries(id: number, name: string, sequence_number: number | null): Promise<Book> {
    return req(`/api/v1/books/${id}/series`, {
      method: "PUT",
      body: JSON.stringify({ name, sequence_number }),
    });
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
  getProgress(id: number): Promise<Progress> {
    return req(`/api/v1/progress/${id}`);
  },
  postProgress(
    id: number,
    body: {
      current_cfi?: string;
      percent_completed?: number;
      seconds_delta?: number;
      completed?: boolean;
    },
  ): Promise<Progress> {
    return req(`/api/v1/progress/${id}`, { method: "POST", body: JSON.stringify(body) });
  },
  stats(): Promise<Stats> {
    return req("/api/v1/stats");
  },
  settings(): Promise<Settings> {
    return req("/api/v1/settings");
  },
  saveSettings(s: Settings): Promise<Settings> {
    return req("/api/v1/settings", { method: "PUT", body: JSON.stringify(s) });
  },
};

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
