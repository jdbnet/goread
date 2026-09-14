import { ref } from "vue";
import type { Book } from "../types";
import { ApiError } from "../http";
import { API_CACHE, BOOKS_CACHE, bookFileUrl, bookJsonUrl } from "./cacheNames";
import { DOWNLOADS_STORE, idbDelete, idbGet, idbGetAll, idbPut, type DownloadRecord } from "./idb";

export const downloadedIds = ref<Set<number>>(new Set());

export async function hydrateDownloads(): Promise<void> {
  const rows = await idbGetAll<DownloadRecord>(DOWNLOADS_STORE);
  downloadedIds.value = new Set(rows.map((row) => row.bookId));
}

export async function listDownloads(): Promise<DownloadRecord[]> {
  return idbGetAll<DownloadRecord>(DOWNLOADS_STORE);
}

export async function getDownload(bookId: number): Promise<DownloadRecord | undefined> {
  return idbGet<DownloadRecord>(DOWNLOADS_STORE, bookId);
}

export async function hasBookFile(bookId: number): Promise<boolean> {
  const cache = await caches.open(BOOKS_CACHE);
  const hit = await cache.match(bookFileUrl(bookId));
  return Boolean(hit);
}

async function persistStorage(): Promise<void> {
  if (!navigator.storage?.persist) return;
  try {
    await navigator.storage.persist();
  } catch {
    /* best effort */
  }
}

async function putOk(cacheName: string, request: string, res: Response): Promise<void> {
  if (!res.ok) return;
  const cache = await caches.open(cacheName);
  await cache.put(request, res);
}

export async function downloadBook(book: Book): Promise<void> {
  await persistStorage();
  const fileRes = await fetch(bookFileUrl(book.id), { credentials: "same-origin" });
  if (!fileRes.ok) {
    throw new ApiError(fileRes.status, "Could not download this book.");
  }
  await putOk(BOOKS_CACHE, bookFileUrl(book.id), fileRes);

  const jsonRes = await fetch(bookJsonUrl(book.id), {
    credentials: "same-origin",
    headers: { Accept: "application/json" },
  });
  if (jsonRes.ok) {
    await putOk(API_CACHE, bookJsonUrl(book.id), jsonRes.clone());
    try {
      const fresh = (await jsonRes.json()) as Book;
      book = fresh;
    } catch {
      /* keep caller snapshot */
    }
  }

  if (book.has_cover && book.cover_url) {
    try {
      const coverRes = await fetch(book.cover_url, { credentials: "same-origin" });
      await putOk(API_CACHE, book.cover_url, coverRes);
    } catch {
      /* cover is optional */
    }
  }

  const record: DownloadRecord = {
    bookId: book.id,
    title: book.title,
    downloadedAt: new Date().toISOString(),
    snapshot: book,
  };
  await idbPut(DOWNLOADS_STORE, record);
  const next = new Set(downloadedIds.value);
  next.add(book.id);
  downloadedIds.value = next;
}

export async function removeDownload(bookId: number): Promise<void> {
  const cache = await caches.open(BOOKS_CACHE);
  await cache.delete(bookFileUrl(bookId));
  await idbDelete(DOWNLOADS_STORE, bookId);
  const next = new Set(downloadedIds.value);
  next.delete(bookId);
  downloadedIds.value = next;
}
