import { apiFetch } from './client';

export interface AuthConfig {
  methods: string[];
  ldap_enabled: boolean;
}

export interface AuthUser {
  username: string;
  email?: string;
  role: 'super_admin' | 'admin' | 'developer' | 'readonly';
  auth_method: string;
}

export interface AuthResponse {
  token: string;
  expires_at: string;
  user: AuthUser;
}

export async function fetchAuthConfig(): Promise<AuthConfig> {
  return apiFetch<AuthConfig>('/ui/auth/config');
}

export async function loginWithCredentials(
  username: string,
  password: string
): Promise<AuthResponse> {
  return apiFetch<AuthResponse>('/ui/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  });
}

export async function loginWithApiKey(key: string): Promise<AuthResponse> {
  return apiFetch<AuthResponse>('/ui/auth/apikey', {
    method: 'POST',
    body: JSON.stringify({ key }),
  });
}

export async function fetchSession(): Promise<AuthResponse> {
  return apiFetch<AuthResponse>('/ui/auth/session');
}

export async function logout(): Promise<void> {
  return apiFetch<void>('/ui/auth/logout', { method: 'POST' });
}
