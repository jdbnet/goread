import { computed, ref, shallowRef } from "vue";
import type { Book, Progress, Stats } from "../types";
import { ApiError, isNetworkError, req } from "../http";
import { PENDING_STORE, BASELINES_STORE, idbDelete, idbGet, idbGetAll, idbPut, type PendingProgress, type BaselineRecord } from "./idb";

export type ProgressWrite = {
  current_cfi?: string;
  percent_completed?: number;
  completed?: boolean;
  seconds_delta?: number;
};

const pendingMap = shallowRef(new Map<number, PendingProgress>());
const baselines = new Map<number, string | null>();
const snapshots = new Map<number, Book>();
const bookLocks = new Map<number, Promise<unknown>>();

export const syncTick = ref(0);
export const syncing = ref(false);
export const pendingCount = computed(() => pendingMap.value.size);

function bumpSync(): void {
  pendingMap.value = new Map(pendingMap.value);
  syncTick.value += 1;
}

function withBookLock<T>(bookId: number, fn: () => Promise<T>): Promise<T> {
  const prev = bookLocks.get(bookId) ?? Promise.resolve();
  const next = prev.catch(() => undefined).then(fn);
  bookLocks.set(bookId, next);
  return next;
}

export function rememberBook(book: Book): void {
  snapshots.set(book.id, book);
  const incoming = book.last_read_at;
  const current = baselines.get(book.id);
  if (!current || (incoming && (Number.isNaN(Date.parse(current)) || Date.parse(incoming) >= Date.parse(current)))) {
    void setBaseline(book.id, incoming);
  }
}

export function rememberProgress(p: Progress): void {
  void setBaseline(p.book_id, p.last_read_at);
}

function setBaseline(bookId: number, lastReadAt: string | null): void {
  const current = baselines.get(bookId);
  if (current && lastReadAt && Date.parse(lastReadAt) < Date.parse(current)) {
    return;
  }
  baselines.set(bookId, lastReadAt);
  void idbPut<BaselineRecord>(BASELINES_STORE, { bookId, lastReadAt });
}

export function overlayBook(book: Book): Book {
  const pending = pendingMap.value.get(book.id);
  if (!pending) return book;
  let completedAt = book.completed_at;
  if (pending.completed === true) {
    completedAt = completedAt || pending.updatedAt;
  } else if (pending.completed === false) {
    completedAt = null;
  }
  return {
    ...book,
    current_cfi: pending.current_cfi ?? book.current_cfi,
    percent_completed: pending.percent_completed ?? book.percent_completed,
    last_read_at: pending.updatedAt,
    completed_at: completedAt,
    total_time_read_seconds: book.total_time_read_seconds + pending.seconds_delta,
  };
}

export function overlayBooks(books: Book[]): Book[] {
  return books.map(overlayBook);
}

export function overlayContinue(books: Book[]): Book[] {
  const out = books.map(overlayBook);
  const seen = new Set(out.map((b) => b.id));
  for (const pending of pendingMap.value.values()) {
    if (seen.has(pending.bookId) || !pending.snapshot) continue;
    out.unshift(overlayBook(pending.snapshot));
  }
  return out;
}

export function overlayStats(stats: Stats): Stats {
  let extra = 0;
  for (const pending of pendingMap.value.values()) {
    extra += pending.seconds_delta;
  }
  if (extra <= 0) return stats;
  const now = new Date();
  const today = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}-${String(now.getDate()).padStart(2, "0")}`;
  const daily = stats.daily.map((d) =>
    d.day === today ? { ...d, time_read_seconds: d.time_read_seconds + extra } : d,
  );
  if (!daily.some((d) => d.day === today)) {
    daily.unshift({ day: today, time_read_seconds: extra });
  }
  return {
    ...stats,
    total_time_read_seconds: stats.total_time_read_seconds + extra,
    this_week_seconds: stats.this_week_seconds + extra,
    daily,
  };
}

export async function hydratePending(): Promise<void> {
  const [rows, storedBaselines] = await Promise.all([
    idbGetAll<PendingProgress>(PENDING_STORE),
    idbGetAll<BaselineRecord>(BASELINES_STORE),
  ]);
  const next = new Map<number, PendingProgress>();
  for (const row of rows) {
    next.set(row.bookId, row);
    if (row.snapshot) snapshots.set(row.bookId, row.snapshot);
  }
  pendingMap.value = next;
  for (const row of storedBaselines) {
    baselines.set(row.bookId, row.lastReadAt);
  }
}

export async function getPending(bookId: number): Promise<PendingProgress | undefined> {
  return pendingMap.value.get(bookId) ?? idbGet<PendingProgress>(PENDING_STORE, bookId);
}

function mergePending(bookId: number, body: ProgressWrite, existing?: PendingProgress): PendingProgress {
  return {
    bookId,
    current_cfi: body.current_cfi ?? existing?.current_cfi,
    percent_completed: body.percent_completed ?? existing?.percent_completed,
    completed: body.completed ?? existing?.completed,
    seconds_delta: (existing?.seconds_delta ?? 0) + Math.max(0, body.seconds_delta ?? 0),
    baseline_last_read_at: existing?.baseline_last_read_at ?? baselines.get(bookId) ?? null,
    updatedAt: new Date().toISOString(),
    snapshot: existing?.snapshot ?? snapshots.get(bookId),
  };
}

function toLocalProgress(bookId: number, pending: PendingProgress): Progress {
  const snap = pending.snapshot ?? snapshots.get(bookId);
  let completedAt: string | null = snap?.completed_at ?? null;
  if (pending.completed === true) {
    completedAt = completedAt || pending.updatedAt;
  } else if (pending.completed === false) {
    completedAt = null;
  }
  return {
    book_id: bookId,
    current_cfi: pending.current_cfi ?? snap?.current_cfi ?? "",
    percent_completed: pending.percent_completed ?? snap?.percent_completed ?? 0,
    total_time_read_seconds: (snap?.total_time_read_seconds ?? 0) + pending.seconds_delta,
    last_read_at: pending.updatedAt,
    completed_at: completedAt,
  };
}

async function savePending(pending: PendingProgress): Promise<void> {
  await idbPut(PENDING_STORE, pending);
  const next = new Map(pendingMap.value);
  next.set(pending.bookId, pending);
  pendingMap.value = next;
}

async function clearPending(bookId: number): Promise<void> {
  await idbDelete(PENDING_STORE, bookId);
  if (!pendingMap.value.has(bookId)) return;
  const next = new Map(pendingMap.value);
  next.delete(bookId);
  pendingMap.value = next;
}

function payloadFromPending(pending: PendingProgress, position: boolean): Record<string, unknown> {
  const body: Record<string, unknown> = {};
  if (pending.seconds_delta > 0) {
    body.seconds_delta = pending.seconds_delta;
  }
  if (position) {
    if (pending.current_cfi) body.current_cfi = pending.current_cfi;
    if (pending.percent_completed != null) body.percent_completed = pending.percent_completed;
    if (pending.completed != null) body.completed = pending.completed;
  }
  if (pending.baseline_last_read_at) {
    body.baseline_last_read_at = pending.baseline_last_read_at;
  }
  return body;
}

function serverIsNewer(server: string | null, baseline: string | null): boolean {
  if (!server || !baseline) return false;
  const a = Date.parse(server);
  const b = Date.parse(baseline);
  if (Number.isNaN(a) || Number.isNaN(b)) return false;
  return a > b;
}

async function postToServer(bookId: number, body: Record<string, unknown>): Promise<Progress> {
  return req<Progress>(`/api/v1/progress/${bookId}`, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

async function queueLocal(bookId: number, body: ProgressWrite): Promise<Progress> {
  const existing = pendingMap.value.get(bookId) ?? (await idbGet<PendingProgress>(PENDING_STORE, bookId));
  const merged = mergePending(bookId, body, existing);
  await savePending(merged);
  bumpSync();
  return toLocalProgress(bookId, merged);
}

export async function recordProgressPost(bookId: number, body: ProgressWrite): Promise<Progress> {
  return withBookLock(bookId, async () => {
    if (!navigator.onLine) {
      return queueLocal(bookId, body);
    }
    const existing = pendingMap.value.get(bookId) ?? (await idbGet<PendingProgress>(PENDING_STORE, bookId));
    const merged = mergePending(bookId, body, existing);
    const payload = existing
      ? payloadFromPending(merged, true)
      : (Object.fromEntries(
          Object.entries(body).filter(([, v]) => v !== undefined),
        ) as Record<string, unknown>);
    try {
      const result = await postToServer(bookId, payload);
      rememberProgress(result);
      if (existing) {
        await clearPending(bookId);
        bumpSync();
      }
      return result;
    } catch (err) {
      if (isNetworkError(err) || !navigator.onLine) {
        await savePending(merged);
        bumpSync();
        return toLocalProgress(bookId, merged);
      }
      throw err;
    }
  });
}

async function flushOne(pending: PendingProgress): Promise<void> {
  await withBookLock(pending.bookId, async () => {
    const current =
      pendingMap.value.get(pending.bookId) ?? (await idbGet<PendingProgress>(PENDING_STORE, pending.bookId));
    if (!current) return;
    let position = true;
    try {
      const server = await req<Progress>(`/api/v1/progress/${current.bookId}`);
      rememberProgress(server);
      if (serverIsNewer(server.last_read_at, current.baseline_last_read_at)) {
        position = false;
      }
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) {
        await clearPending(current.bookId);
        return;
      }
      throw err;
    }
    const result = await postToServer(current.bookId, payloadFromPending(current, position));
    await clearPending(current.bookId);
    rememberProgress(result);
  });
}

export async function flushPending(): Promise<void> {
  if (!navigator.onLine || pendingMap.value.size === 0) return;
  if (syncing.value) return;
  syncing.value = true;
  try {
    const rows = [...pendingMap.value.values()];
    for (const row of rows) {
      try {
        await flushOne(row);
      } catch (err) {
        if (!navigator.onLine || isNetworkError(err)) {
          break;
        }
      }
    }
    bumpSync();
  } finally {
    syncing.value = false;
  }
}
