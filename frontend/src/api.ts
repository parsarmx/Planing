export class APIError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}
export async function api<T>(path: string, method = 'GET', body?: unknown): Promise<T> {
  const response = await fetch('/api' + path, {
    method,
    credentials: 'same-origin',
    headers: method === 'GET' ? {} : { 'Content-Type': 'application/json' },
    body: method === 'GET' ? undefined : JSON.stringify(body ?? {}),
  }).catch(() => {
    throw new APIError(0, 'ارتباط با سرور برقرار نشد. اتصال اینترنت را بررسی کنید.');
  });
  if (!response.ok) {
    const data = await response.json().catch(() => null);
    throw new APIError(
      response.status,
      data?.error?.message || 'ارتباط با سرور برقرار نشد. دوباره تلاش کنید.',
    );
  }
  return response.status === 204 ? (undefined as T) : response.json();
}
