import type { Book } from "../types";

const DB_NAME = "goread-offline";
const DB_VERSION = 1;

export const DOWNLOADS_STORE = "downloads";
export const PENDING_STORE = "pendingProgress";
export const BASELINES_STORE = "baselines";

export type DownloadRecord = {
  bookId: number;
  title: string;
  downloadedAt: string;
  snapshot: Book;
};

export type BaselineRecord = {
  bookId: number;
  lastReadAt: string | null;
};

export type PendingProgress = {
  bookId: number;
  current_cfi?: string;
  percent_completed?: number;
  completed?: boolean;
  seconds_delta: number;
  baseline_last_read_at: string | null;
  updatedAt: string;
  snapshot?: Book;
};

function openDb(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, DB_VERSION);
    req.onupgradeneeded = () => {
      const db = req.result;
      if (!db.objectStoreNames.contains(DOWNLOADS_STORE)) {
        db.createObjectStore(DOWNLOADS_STORE, { keyPath: "bookId" });
      }
      if (!db.objectStoreNames.contains(PENDING_STORE)) {
        db.createObjectStore(PENDING_STORE, { keyPath: "bookId" });
      }
      if (!db.objectStoreNames.contains(BASELINES_STORE)) {
        db.createObjectStore(BASELINES_STORE, { keyPath: "bookId" });
      }
    };
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error ?? new Error("IndexedDB open failed"));
  });
}

function reqToPromise<T>(request: IDBRequest<T>): Promise<T> {
  return new Promise((resolve, reject) => {
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error ?? new Error("IndexedDB request failed"));
  });
}

function txDone(tx: IDBTransaction): Promise<void> {
  return new Promise((resolve, reject) => {
    tx.oncomplete = () => resolve();
    tx.onerror = () => reject(tx.error ?? new Error("IndexedDB transaction failed"));
    tx.onabort = () => reject(tx.error ?? new Error("IndexedDB transaction aborted"));
  });
}

export async function idbPut<T>(store: string, value: T): Promise<void> {
  const db = await openDb();
  try {
    const tx = db.transaction(store, "readwrite");
    tx.objectStore(store).put(value);
    await txDone(tx);
  } finally {
    db.close();
  }
}

export async function idbGet<T>(store: string, key: number): Promise<T | undefined> {
  const db = await openDb();
  try {
    const tx = db.transaction(store, "readonly");
    const value = await reqToPromise(tx.objectStore(store).get(key));
    await txDone(tx);
    return value as T | undefined;
  } finally {
    db.close();
  }
}

export async function idbDelete(store: string, key: number): Promise<void> {
  const db = await openDb();
  try {
    const tx = db.transaction(store, "readwrite");
    tx.objectStore(store).delete(key);
    await txDone(tx);
  } finally {
    db.close();
  }
}

export async function idbGetAll<T>(store: string): Promise<T[]> {
  const db = await openDb();
  try {
    const tx = db.transaction(store, "readonly");
    const value = await reqToPromise(tx.objectStore(store).getAll());
    await txDone(tx);
    return value as T[];
  } finally {
    db.close();
  }
}
