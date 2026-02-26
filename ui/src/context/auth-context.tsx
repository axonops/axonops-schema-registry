import { createContext, useContext, useState, useCallback, useEffect, useRef, type ReactNode } from 'react';
import type { AuthUser, AuthConfig } from '@/api/auth';
import { fetchAuthConfig, loginWithCredentials, loginWithApiKey, fetchSession, logout as logoutApi } from '@/api/auth';
import { setToken, setOnAuthFailure } from '@/api/client';

interface AuthContextType {
  user: AuthUser | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  authConfig: AuthConfig | null;
  login: (username: string, password: string) => Promise<void>;
  loginWithKey: (apiKey: string) => Promise<void>;
  logout: () => Promise<void>;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const AuthContext = createContext<AuthContextType | null>(null as any);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [authConfig, setAuthConfig] = useState<AuthConfig | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const refreshTimerRef = useRef<number | undefined>(undefined);

  const scheduleRefresh = useCallback((expiresAt: string) => {
    if (refreshTimerRef.current) clearTimeout(refreshTimerRef.current);

    const expiryMs = new Date(expiresAt).getTime();
    const refreshMs = expiryMs - Date.now() - 60_000; // 1 min before expiry

    if (refreshMs > 0) {
      refreshTimerRef.current = window.setTimeout(async () => {
        try {
          const res = await fetchSession();
          setToken(res.token);
          setUser(res.user);
          scheduleRefresh(res.expires_at);
        } catch {
          setToken(null);
          setUser(null);
        }
      }, refreshMs);
    }
  }, []);

  useEffect(() => {
    async function init() {
      // Dev bypass: auto-authenticate when no backend is running
      if (import.meta.env.DEV && import.meta.env.VITE_DEV_AUTH_BYPASS === 'true') {
        setUser({
          username: 'dev-admin',
          email: 'dev@localhost',
          role: 'super_admin',
          auth_method: 'dev-bypass',
        });
        setAuthConfig({ methods: ['basic', 'api_key'], ldap_enabled: false });
        setIsLoading(false);
        return;
      }

      try {
        const config = await fetchAuthConfig();
        setAuthConfig(config);

        try {
          const session = await fetchSession();
          setToken(session.token);
          setUser(session.user);
          scheduleRefresh(session.expires_at);
        } catch {
          // No valid session
        }
      } catch {
        // Auth config fetch failed — server might not have auth enabled
        // Set empty config so login page can show appropriate state
        setAuthConfig({ methods: [], ldap_enabled: false });
      } finally {
        setIsLoading(false);
      }
    }
    init();

    setOnAuthFailure(() => {
      setUser(null);
    });
  }, [scheduleRefresh]);

  const login = useCallback(async (username: string, password: string) => {
    const res = await loginWithCredentials(username, password);
    setToken(res.token);
    setUser(res.user);
    scheduleRefresh(res.expires_at);
  }, [scheduleRefresh]);

  const loginWithKey = useCallback(async (apiKey: string) => {
    const res = await loginWithApiKey(apiKey);
    setToken(res.token);
    setUser(res.user);
    scheduleRefresh(res.expires_at);
  }, [scheduleRefresh]);

  const logout = useCallback(async () => {
    try {
      await logoutApi();
    } catch {
      // Best-effort
    }
    if (refreshTimerRef.current) clearTimeout(refreshTimerRef.current);
    setToken(null);
    setUser(null);
  }, []);

  return (
    <AuthContext.Provider value={{
      user,
      isAuthenticated: !!user,
      isLoading,
      authConfig,
      login,
      loginWithKey,
      logout,
    }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
