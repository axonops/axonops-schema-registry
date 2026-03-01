import { apiFetch } from './client';

export interface AuthConfig {
  auth_enabled: boolean;
}

export interface AuthUser {
  username: string;
}

export interface AuthResponse {
  username: string;
}

export async function fetchAuthConfig(): Promise<AuthConfig> {
  return apiFetch<AuthConfig>('/api/auth/config');
}

export async function loginWithCredentials(
  username: string,
  password: string
): Promise<AuthResponse> {
  return apiFetch<AuthResponse>('/api/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  });
}

export async function fetchSession(): Promise<AuthResponse> {
  return apiFetch<AuthResponse>('/api/auth/session');
}

export async function logout(): Promise<void> {
  return apiFetch<void>('/api/auth/logout', { method: 'POST' });
}
