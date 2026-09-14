export const SHELL_CACHE = "goread-shell-v1";
export const API_CACHE = "goread-api-v1";
export const BOOKS_CACHE = "goread-books";

export function bookFileUrl(bookId: number): string {
  return `/api/v1/books/${bookId}/file`;
}

export function bookJsonUrl(bookId: number): string {
  return `/api/v1/books/${bookId}`;
}
