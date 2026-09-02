import type { Book, BookListResponse, MetadataHit, Progress, Series, Settings, Stats } from "./types";

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: {
      Accept: "application/json",
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
      ...init?.headers,
    },
  });
  if (!res.ok) {
    let message = res.statusText;
    try {
      const body = (await res.json()) as { error?: string };
      if (body.error) message = body.error;
    } catch {
      /* ignore */
    }
    throw new Error(message);
  }
  return (await res.json()) as T;
}

export const api = {
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
  setSeries(id: number, name: string, sequence_number: number | null): Promise<Book> {
    return req(`/api/v1/books/${id}/series`, {
      method: "PUT",
      body: JSON.stringify({ name, sequence_number }),
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
