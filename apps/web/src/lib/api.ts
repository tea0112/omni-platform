const BASE_URL = process.env.IDENTITY_SERVICE_URL ?? "http://localhost:8080";

type ApiError = {
  code: string;
  message: string;
  details?: { fields?: Record<string, string> };
};

export class IdentityError extends Error {
  code: string;
  details?: { fields?: Record<string, string> };

  constructor(err: ApiError) {
    super(err.message);
    this.name = "IdentityError";
    this.code = err.code;
    this.details = err.details;
  }
}

type RequestOptions = {
  method?: string;
  body?: unknown;
  token?: string;
};

export async function api<T = unknown>(
  path: string,
  options: RequestOptions = {}
): Promise<T> {
  const { method = "GET", body, token } = options;

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${BASE_URL}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  const data = await res.json();

  if (!res.ok) {
    throw new IdentityError(data as ApiError);
  }

  return data as T;
}
