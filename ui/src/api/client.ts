import createClient from 'openapi-fetch';
import type { paths } from '@/types/api';

let accessToken: string | null = null;
let onAuthFailure: (() => void) | null = null;

export function setToken(token: string | null) {
  accessToken = token;
}

export function getToken(): string | null {
  return accessToken;
}

export function setOnAuthFailure(handler: () => void) {
  onAuthFailure = handler;
}

export const api = createClient<paths>({
  baseUrl: '',
});

// Add auth middleware
api.use({
  async onRequest({ request }) {
    if (accessToken) {
      request.headers.set('Authorization', `Bearer ${accessToken}`);
    }
    request.headers.set('Accept', 'application/vnd.schemaregistry.v1+json, application/json');
    return request;
  },
  async onResponse({ response }) {
    if (response.status === 401) {
      setToken(null);
      onAuthFailure?.();
    }
    return response;
  },
});

export class ApiClientError extends Error {
  readonly status: number;
  readonly errorCode?: number;

  constructor(status: number, message: string, errorCode?: number) {
    super(message);
    this.name = 'ApiClientError';
    this.status = status;
    this.errorCode = errorCode;
  }
}

// Helper for mutations and direct API calls that need error handling
export async function apiFetch<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const headers: Record<string, string> = {
    'Accept': 'application/vnd.schemaregistry.v1+json, application/json',
    ...(options.headers as Record<string, string> || {}),
  };

  if (accessToken) {
    headers['Authorization'] = `Bearer ${accessToken}`;
  }

  if (options.body && !headers['Content-Type']) {
    headers['Content-Type'] = 'application/vnd.schemaregistry.v1+json';
  }

  const response = await fetch(path, { ...options, headers });

  if (response.status === 401) {
    setToken(null);
    onAuthFailure?.();
    throw new ApiClientError(401, 'Session expired. Please sign in again.');
  }

  if (response.status === 204) {
    return undefined as T;
  }

  const body = await response.json();

  if (!response.ok) {
    throw new ApiClientError(
      response.status,
      body.message || `Request failed with status ${response.status}`,
      body.error_code
    );
  }

  return body as T;
}
