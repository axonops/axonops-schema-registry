// API Keys are not available in the open-source UI.
// This page is retained as a stub to prevent broken imports.

import { useEffect } from 'react';
import { useNavigate } from '@tanstack/react-router';

export function MyApiKeysPage() {
  const navigate = useNavigate();
  useEffect(() => {
    navigate({ to: '/ui/account' });
  }, [navigate]);
  return null;
}
