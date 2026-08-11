const defaultHeaders = {
  'Content-Type': 'application/json',
}

interface ApiErrorBody {
  error?: string
}

export class ApiError extends Error {
  constructor(
    message: string,
    public readonly status: number,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

export async function request<T>(
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: {
      ...defaultHeaders,
      ...init.headers,
    },
  })

  if (!response.ok) {
    let body: ApiErrorBody = {}
    try {
      body = (await response.json()) as ApiErrorBody
    } catch {
      // The status text remains the fallback for non-JSON failures.
    }

    throw new ApiError(
      body.error ?? response.statusText ?? 'Request failed',
      response.status,
    )
  }

  return response.json() as Promise<T>
}
