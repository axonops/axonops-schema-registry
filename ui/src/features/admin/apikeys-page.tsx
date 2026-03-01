// API Keys admin page is not available in the open-source UI.
// This file is retained as a stub to prevent broken imports.

import { useEffect } from 'react';
import { useNavigate } from '@tanstack/react-router';

export function ApiKeysPage() {
  const navigate = useNavigate();
  useEffect(() => {
    navigate({ to: '/ui/admin/users' });
  }, [navigate]);
  return null;
}
