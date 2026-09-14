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

export async function req<T>(path: string, init?: RequestInit): Promise<T> {
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

export function isNetworkError(err: unknown): boolean {
  if (err instanceof ApiError) {
    return err.status >= 500;
  }
  return err instanceof TypeError;
}
