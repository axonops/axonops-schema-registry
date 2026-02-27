import { createContext, useContext, useState, useCallback, useEffect, useMemo, type ReactNode } from 'react';
import { setContextPrefix } from '@/api/client';

const STORAGE_KEY = 'schema-registry-context';

interface RegistryContextType {
  selectedContext: string;
  setSelectedContext: (context: string) => void;
  contextPrefix: string;
}

const RegistryContext = createContext<RegistryContextType | null>(null);

export function RegistryContextProvider({ children }: { children: ReactNode }) {
  const [selectedContext, setSelectedContextState] = useState<string>(() => {
    if (typeof window === 'undefined') return '';
    return sessionStorage.getItem(STORAGE_KEY) || '';
  });

  const contextPrefix = useMemo(
    () => (selectedContext ? `/contexts/${selectedContext}` : ''),
    [selectedContext],
  );

  // Sync context prefix to the API client module whenever it changes
  useEffect(() => {
    setContextPrefix(contextPrefix);
  }, [contextPrefix]);

  // Persist to sessionStorage whenever selectedContext changes
  useEffect(() => {
    if (selectedContext) {
      sessionStorage.setItem(STORAGE_KEY, selectedContext);
    } else {
      sessionStorage.removeItem(STORAGE_KEY);
    }
  }, [selectedContext]);

  const setSelectedContext = useCallback((context: string) => {
    setSelectedContextState(context);
  }, []);

  return (
    <RegistryContext.Provider value={{ selectedContext, setSelectedContext, contextPrefix }}>
      {children}
    </RegistryContext.Provider>
  );
}

export function useRegistryContext() {
  const ctx = useContext(RegistryContext);
  if (!ctx) throw new Error('useRegistryContext must be used within RegistryContextProvider');
  return ctx;
}
