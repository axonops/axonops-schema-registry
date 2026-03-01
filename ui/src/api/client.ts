import createClient from 'openapi-fetch';
import type { paths } from '@/types/api';

let onAuthFailure: (() => void) | null = null;
let _contextPrefix = '';

export function setContextPrefix(prefix: string) {
  _contextPrefix = prefix;
}

export function getContextPrefix(): string {
  return _contextPrefix;
}

// Paths that should be prepended with the registry context prefix.
// These are the context-aware API routes (under /api/v1).
const CONTEXT_AWARE_PREFIXES = [
  '/api/v1/subjects',
  '/api/v1/schemas',
  '/api/v1/config',
  '/api/v1/mode',
  '/api/v1/compatibility',
  '/api/v1/exporters',
  '/api/v1/dek-registry',
];

function withContextPrefix(path: string): string {
  if (!_contextPrefix) return path;
  const isContextAware = CONTEXT_AWARE_PREFIXES.some(
    (prefix) => path === prefix || path.startsWith(prefix + '/') || path.startsWith(prefix + '?'),
  );
  if (!isContextAware) return path;
  // Insert context prefix after /api/v1: /api/v1/subjects → /api/v1/contexts/ctx/subjects
  return path.replace('/api/v1/', `/api/v1/contexts/${_contextPrefix}/`);
}

export function setOnAuthFailure(handler: () => void) {
  onAuthFailure = handler;
}

export const api = createClient<paths>({
  baseUrl: '',
  credentials: 'same-origin',
});

// Add middleware for accept header and 401 handling
api.use({
  async onRequest({ request }) {
    request.headers.set('Accept', 'application/vnd.schemaregistry.v1+json, application/json');
    return request;
  },
  async onResponse({ response }) {
    if (response.status === 401) {
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

  if (options.body && !headers['Content-Type']) {
    headers['Content-Type'] = 'application/vnd.schemaregistry.v1+json';
  }

  const response = await fetch(withContextPrefix(path), {
    ...options,
    headers,
    credentials: 'same-origin',
  });

  if (response.status === 401) {
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
      body.message || body.error || `Request failed with status ${response.status}`,
      body.error_code
    );
  }

  return body as T;
}
