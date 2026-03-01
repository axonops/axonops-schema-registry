import { createContext, useContext, useState, useCallback, useEffect, type ReactNode } from 'react';
import type { AuthUser, AuthConfig } from '@/api/auth';
import { fetchAuthConfig, loginWithCredentials, fetchSession, logout as logoutApi } from '@/api/auth';
import { setOnAuthFailure } from '@/api/client';

interface AuthContextType {
  user: AuthUser | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  authConfig: AuthConfig | null;
  login: (username: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const AuthContext = createContext<AuthContextType | null>(null as any);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [authConfig, setAuthConfig] = useState<AuthConfig | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    async function init() {
      try {
        const config = await fetchAuthConfig();
        setAuthConfig(config);

        try {
          const session = await fetchSession();
          setUser({ username: session.username });
        } catch {
          // No valid session — user needs to login
        }
      } catch {
        // Auth config fetch failed
        setAuthConfig({ auth_enabled: true });
      } finally {
        setIsLoading(false);
      }
    }
    init();

    setOnAuthFailure(() => {
      setUser(null);
    });
  }, []);

  const login = useCallback(async (username: string, password: string) => {
    const res = await loginWithCredentials(username, password);
    setUser({ username: res.username });
  }, []);

  const logout = useCallback(async () => {
    try {
      await logoutApi();
    } catch {
      // Best-effort
    }
    setUser(null);
  }, []);

  return (
    <AuthContext.Provider value={{
      user,
      isAuthenticated: !!user,
      isLoading,
      authConfig,
      login,
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
