import { useState, useMemo } from 'react';
import { useApiKeys, useCreateApiKey, useRevokeApiKey, useRotateApiKey, useDeleteApiKey } from '@/api/queries';
import type { ApiKey, CreateApiKeyRequest, CreateApiKeyResponse } from '@/api/queries';
import { useAuth } from '@/context/auth-context';
import { PageBreadcrumbs } from '@/components/shared/breadcrumbs';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from '@/components/ui/dialog';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Skeleton } from '@/components/ui/skeleton';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { toast } from 'sonner';
import { Plus, Copy, RotateCcw, Ban, Trash2, Search, Key, AlertTriangle, AlertCircle } from 'lucide-react';

// ── Helpers ──

function getKeyStatus(key: ApiKey): 'active' | 'revoked' | 'expired' {
  if (key.revoked_at) return 'revoked';
  if (key.expires_at && new Date(key.expires_at) < new Date()) return 'expired';
  return 'active';
}

const STATUS_BADGE_VARIANT: Record<ReturnType<typeof getKeyStatus>, 'default' | 'secondary' | 'destructive' | 'outline'> = {
  active: 'default',
  revoked: 'destructive',
  expired: 'secondary',
};

const STATUS_LABELS: Record<ReturnType<typeof getKeyStatus>, string> = {
  active: 'Active',
  revoked: 'Revoked',
  expired: 'Expired',
};

const EXPIRY_PRESETS = [
  { label: '7 days', value: '604800' },
  { label: '30 days', value: '2592000' },
  { label: '90 days', value: '7776000' },
  { label: '1 year', value: '31536000' },
  { label: 'Never', value: '0' },
] as const;

const ROLE_OPTIONS = ['admin', 'developer', 'readonly'] as const;

function formatDate(dateStr: string | null): string {
  if (!dateStr) return 'Never';
  return new Date(dateStr).toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}

// ── Component ──

export function ApiKeysPage() {
  useAuth();

  // Data fetching
  const { data: apiKeys, isLoading, error: fetchError } = useApiKeys();

  // Mutations
  const createMutation = useCreateApiKey();
  const revokeMutation = useRevokeApiKey();
  const rotateMutation = useRotateApiKey();
  const deleteMutation = useDeleteApiKey();

  // Search
  const [searchQuery, setSearchQuery] = useState('');

  // Create dialog
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [formName, setFormName] = useState('');
  const [formRole, setFormRole] = useState<string>('');
  const [formExpiry, setFormExpiry] = useState<string>('');

  // One-time key display (used for both create and rotate)
  const [createdKeyResponse, setCreatedKeyResponse] = useState<CreateApiKeyResponse | null>(null);
  const [keyDisplayOpen, setKeyDisplayOpen] = useState(false);
  const [keyCopied, setKeyCopied] = useState(false);

  // Confirm dialogs
  const [revokeTarget, setRevokeTarget] = useState<ApiKey | null>(null);
  const [rotateTarget, setRotateTarget] = useState<ApiKey | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<ApiKey | null>(null);

  // ── Filtered keys ──

  const filteredKeys = useMemo(() => {
    if (!apiKeys) return [];
    if (!searchQuery.trim()) return apiKeys;
    const q = searchQuery.toLowerCase();
    return apiKeys.filter((k) => k.name.toLowerCase().includes(q));
  }, [apiKeys, searchQuery]);

  // ── Create form handlers ──

  const resetCreateForm = () => {
    setFormName('');
    setFormRole('');
    setFormExpiry('');
  };

  const handleCreateSubmit = () => {
    if (!formName.trim() || !formRole || !formExpiry) return;

    const request: CreateApiKeyRequest = {
      name: formName.trim(),
      role: formRole as CreateApiKeyRequest['role'],
    };

    const expirySeconds = parseInt(formExpiry, 10);
    if (expirySeconds > 0) {
      request.expires_in = expirySeconds;
    }

    createMutation.mutate(request, {
      onSuccess: (response) => {
        setCreateDialogOpen(false);
        resetCreateForm();
        setCreatedKeyResponse(response);
        setKeyDisplayOpen(true);
        setKeyCopied(false);
        toast.success(`API key "${response.name}" created successfully`);
      },
      onError: (err: Error) => {
        toast.error(`Failed to create API key: ${err.message}`);
      },
    });
  };

  const handleCreateDialogClose = (open: boolean) => {
    if (!open) {
      resetCreateForm();
    }
    setCreateDialogOpen(open);
  };

  // ── Key display dialog ──

  const handleCopyKey = async () => {
    if (!createdKeyResponse) return;
    try {
      await navigator.clipboard.writeText(createdKeyResponse.key);
      setKeyCopied(true);
      toast.success('API key copied to clipboard');
    } catch {
      toast.error('Failed to copy to clipboard');
    }
  };

  const handleKeyDisplayClose = (open: boolean) => {
    if (!open) {
      setCreatedKeyResponse(null);
      setKeyCopied(false);
    }
    setKeyDisplayOpen(open);
  };

  // ── Revoke handler ──

  const handleRevokeConfirm = () => {
    if (!revokeTarget) return;
    revokeMutation.mutate(revokeTarget.id, {
      onSuccess: () => {
        toast.success(`API key "${revokeTarget.name}" revoked`);
        setRevokeTarget(null);
      },
      onError: (err: Error) => {
        toast.error(`Failed to revoke API key: ${err.message}`);
      },
    });
  };

  // ── Rotate handler ──

  const handleRotateConfirm = () => {
    if (!rotateTarget) return;
    rotateMutation.mutate(rotateTarget.id, {
      onSuccess: (response) => {
        setRotateTarget(null);
        setCreatedKeyResponse(response);
        setKeyDisplayOpen(true);
        setKeyCopied(false);
        toast.success(`API key "${rotateTarget.name}" rotated successfully`);
      },
      onError: (err: Error) => {
        toast.error(`Failed to rotate API key: ${err.message}`);
      },
    });
  };

  // ── Delete handler ──

  const handleDeleteConfirm = () => {
    if (!deleteTarget) return;
    deleteMutation.mutate(deleteTarget.id, {
      onSuccess: () => {
        toast.success(`API key "${deleteTarget.name}" deleted`);
        setDeleteTarget(null);
      },
      onError: (err: Error) => {
        toast.error(`Failed to delete API key: ${err.message}`);
      },
    });
  };

  // ── Render ──

  return (
    <div data-testid="apikeys-page">
      <PageBreadcrumbs items={[{ label: 'API Keys' }]} />

      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">API Keys</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Manage API keys for programmatic access to the schema registry.
          </p>
        </div>
        <Button
          onClick={() => setCreateDialogOpen(true)}
          data-testid="apikeys-create-btn"
        >
          <Plus className="mr-1.5 h-4 w-4" />
          Create API Key
        </Button>
      </div>

      {/* Search */}
      <div className="relative mb-4 max-w-sm">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          placeholder="Search by key name..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="pl-9"
          data-testid="apikeys-search-input"
        />
      </div>

      {/* Error state */}
      {fetchError && (
        <Alert variant="destructive" className="mb-4">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>
            Failed to load API keys: {fetchError.message}
          </AlertDescription>
        </Alert>
      )}

      {/* Table */}
      <div className="rounded-md border">
        <Table data-testid="apikeys-list-table">
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Key Prefix</TableHead>
              <TableHead>Role</TableHead>
              <TableHead>Owner</TableHead>
              <TableHead>Expires</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading && (
              <>
                {Array.from({ length: 3 }).map((_, i) => (
                  <TableRow key={`skeleton-${i}`}>
                    <TableCell><Skeleton className="h-4 w-32" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-20" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-16" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-24" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-20" /></TableCell>
                    <TableCell><Skeleton className="h-5 w-16" /></TableCell>
                    <TableCell><Skeleton className="ml-auto h-8 w-24" /></TableCell>
                  </TableRow>
                ))}
              </>
            )}

            {!isLoading && filteredKeys.length === 0 && (
              <TableRow>
                <TableCell colSpan={7} className="py-12 text-center">
                  <div className="flex flex-col items-center gap-2 text-muted-foreground">
                    <Key className="h-8 w-8" />
                    <p className="text-sm font-medium">
                      {searchQuery ? 'No API keys match your search' : 'No API keys yet'}
                    </p>
                    {!searchQuery && (
                      <p className="text-xs">
                        Create an API key to get started with programmatic access.
                      </p>
                    )}
                  </div>
                </TableCell>
              </TableRow>
            )}

            {!isLoading && filteredKeys.map((apiKey) => {
              const status = getKeyStatus(apiKey);
              return (
                <TableRow key={apiKey.id}>
                  <TableCell className="font-medium">
                    <div className="flex items-center gap-2">
                      <Key className="h-3.5 w-3.5 text-muted-foreground" />
                      {apiKey.name}
                    </div>
                  </TableCell>
                  <TableCell>
                    <code className="rounded bg-muted px-1.5 py-0.5 text-xs">
                      {apiKey.key_prefix}...
                    </code>
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline">{apiKey.role}</Badge>
                  </TableCell>
                  <TableCell>{apiKey.username}</TableCell>
                  <TableCell>
                    {status === 'expired' ? (
                      <span className="flex items-center gap-1 text-sm text-orange-600 dark:text-orange-400">
                        <AlertTriangle className="h-3.5 w-3.5" />
                        {formatDate(apiKey.expires_at)}
                      </span>
                    ) : (
                      <span className="text-sm">{formatDate(apiKey.expires_at)}</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <Badge variant={STATUS_BADGE_VARIANT[status]}>
                      {STATUS_LABELS[status]}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex items-center justify-end gap-1">
                      {status === 'active' && (
                        <>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => setRotateTarget(apiKey)}
                            title="Rotate key"
                            data-testid="apikey-rotate-btn"
                          >
                            <RotateCcw className="h-4 w-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => setRevokeTarget(apiKey)}
                            title="Revoke key"
                            data-testid="apikey-revoke-btn"
                          >
                            <Ban className="h-4 w-4" />
                          </Button>
                        </>
                      )}
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setDeleteTarget(apiKey)}
                        title="Delete key"
                        className="text-destructive hover:text-destructive"
                        data-testid="apikey-delete-btn"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </div>

      {/* ── Create API Key Dialog ── */}
      <Dialog open={createDialogOpen} onOpenChange={handleCreateDialogClose}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create API Key</DialogTitle>
            <DialogDescription>
              Create a new API key for programmatic access to the schema registry.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-2">
            {/* Name */}
            <div className="space-y-2">
              <Label htmlFor="apikey-name">Name</Label>
              <Input
                id="apikey-name"
                value={formName}
                onChange={(e) => setFormName(e.target.value)}
                placeholder="e.g., ci-pipeline-key"
                data-testid="apikey-form-name-input"
              />
            </div>

            {/* Role */}
            <div className="space-y-2">
              <Label>Role</Label>
              <Select value={formRole} onValueChange={setFormRole}>
                <SelectTrigger data-testid="apikey-form-role-select">
                  <SelectValue placeholder="Select a role" />
                </SelectTrigger>
                <SelectContent>
                  {ROLE_OPTIONS.map((role) => (
                    <SelectItem key={role} value={role}>
                      {role.charAt(0).toUpperCase() + role.slice(1)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {/* Expiration */}
            <div className="space-y-2">
              <Label>Expiration</Label>
              <Select value={formExpiry} onValueChange={setFormExpiry}>
                <SelectTrigger data-testid="apikey-form-expiry-select">
                  <SelectValue placeholder="Select expiration" />
                </SelectTrigger>
                <SelectContent>
                  {EXPIRY_PRESETS.map((preset) => (
                    <SelectItem key={preset.value} value={preset.value}>
                      {preset.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => handleCreateDialogClose(false)}
            >
              Cancel
            </Button>
            <Button
              onClick={handleCreateSubmit}
              disabled={!formName.trim() || !formRole || !formExpiry || createMutation.isPending}
              data-testid="apikey-form-submit-btn"
            >
              {createMutation.isPending ? 'Creating...' : 'Create Key'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ── One-Time Key Display Dialog ── */}
      <Dialog open={keyDisplayOpen} onOpenChange={handleKeyDisplayClose}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Key className="h-5 w-5" />
              API Key Created
            </DialogTitle>
            <DialogDescription>
              Your new API key has been generated. Copy it now — you will not be able to see it again.
            </DialogDescription>
          </DialogHeader>

          {createdKeyResponse && (
            <div className="space-y-4 py-2">
              {/* Warning */}
              <Alert className="border-yellow-500/50 bg-yellow-50 text-yellow-900 dark:bg-yellow-950/30 dark:text-yellow-200">
                <AlertTriangle className="h-4 w-4" />
                <AlertDescription>
                  This key will only be shown once. Copy it now and store it securely.
                </AlertDescription>
              </Alert>

              {/* Key display */}
              <div className="space-y-2">
                <Label>API Key</Label>
                <div
                  className="flex items-center gap-2 rounded-md border bg-muted p-3"
                  data-testid="apikey-created-key-display"
                >
                  <code className="flex-1 break-all font-mono text-sm">
                    {createdKeyResponse.key}
                  </code>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleCopyKey}
                    className="shrink-0"
                    data-testid="apikey-copy-btn"
                  >
                    <Copy className="mr-1.5 h-3.5 w-3.5" />
                    {keyCopied ? 'Copied' : 'Copy'}
                  </Button>
                </div>
              </div>

              {/* Key details */}
              <div className="grid grid-cols-2 gap-3 text-sm">
                <div>
                  <span className="text-muted-foreground">Name:</span>{' '}
                  <span className="font-medium">{createdKeyResponse.name}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">Role:</span>{' '}
                  <Badge variant="outline">{createdKeyResponse.role}</Badge>
                </div>
                <div>
                  <span className="text-muted-foreground">Prefix:</span>{' '}
                  <code className="text-xs">{createdKeyResponse.key_prefix}</code>
                </div>
                <div>
                  <span className="text-muted-foreground">Expires:</span>{' '}
                  <span>{formatDate(createdKeyResponse.expires_at)}</span>
                </div>
              </div>
            </div>
          )}

          <DialogFooter>
            <Button onClick={() => handleKeyDisplayClose(false)}>
              Done
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ── Revoke Confirmation ── */}
      <ConfirmDialog
        open={!!revokeTarget}
        onOpenChange={(open) => { if (!open) setRevokeTarget(null); }}
        title="Revoke API Key"
        description={`Are you sure you want to revoke the API key "${revokeTarget?.name}"? This key will immediately stop working. This action cannot be undone.`}
        confirmLabel="Revoke Key"
        destructive
        onConfirm={handleRevokeConfirm}
        isLoading={revokeMutation.isPending}
      />

      {/* ── Rotate Confirmation ── */}
      <ConfirmDialog
        open={!!rotateTarget}
        onOpenChange={(open) => { if (!open) setRotateTarget(null); }}
        title="Rotate API Key"
        description={`Are you sure you want to rotate the API key "${rotateTarget?.name}"? The current key will be invalidated and a new key will be generated.`}
        confirmLabel="Rotate Key"
        onConfirm={handleRotateConfirm}
        isLoading={rotateMutation.isPending}
      />

      {/* ── Delete Confirmation ── */}
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => { if (!open) setDeleteTarget(null); }}
        title="Delete API Key"
        description={`Are you sure you want to permanently delete the API key "${deleteTarget?.name}"? This action cannot be undone.`}
        confirmLabel="Delete Key"
        destructive
        confirmText={deleteTarget?.name}
        onConfirm={handleDeleteConfirm}
        isLoading={deleteMutation.isPending}
      />
    </div>
  );
}
