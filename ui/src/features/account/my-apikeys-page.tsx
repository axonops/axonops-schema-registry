import { useState, useMemo } from 'react';
import { useApiKeys, useCreateApiKey, useRevokeApiKey, useDeleteApiKey } from '@/api/queries';
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
import { Alert, AlertDescription } from '@/components/ui/alert';
import { toast } from 'sonner';
import { Plus, Copy, Ban, Trash2, Key, AlertTriangle } from 'lucide-react';

// ── Helpers ──

function getKeyStatus(key: ApiKey): 'active' | 'revoked' | 'expired' {
  if (key.revoked_at) return 'revoked';
  if (key.expires_at && new Date(key.expires_at) < new Date()) return 'expired';
  return 'active';
}

const statusBadgeVariant: Record<string, 'default' | 'secondary' | 'destructive' | 'outline'> = {
  active: 'default',
  revoked: 'destructive',
  expired: 'secondary',
};

type UserRole = 'super_admin' | 'admin' | 'developer' | 'readonly';
type ApiKeyRole = 'admin' | 'developer' | 'readonly';

const roleHierarchy: Record<UserRole, ApiKeyRole[]> = {
  super_admin: ['admin', 'developer', 'readonly'],
  admin: ['admin', 'developer', 'readonly'],
  developer: ['developer', 'readonly'],
  readonly: ['readonly'],
};

function formatRole(role: string): string {
  return role
    .split('_')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}

function formatDate(dateStr: string | null): string {
  if (!dateStr) return '—';
  return new Date(dateStr).toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

// ── Component ──

export function MyApiKeysPage() {
  const { user } = useAuth();
  const { data: allKeys, isLoading } = useApiKeys();
  const createApiKey = useCreateApiKey();
  const revokeApiKey = useRevokeApiKey();
  const deleteApiKey = useDeleteApiKey();

  // Filter keys to current user only
  const myKeys = useMemo(
    () => (allKeys ?? []).filter((k) => k.username === user?.username),
    [allKeys, user?.username]
  );

  // Create dialog state
  const [createOpen, setCreateOpen] = useState(false);
  const [newKeyName, setNewKeyName] = useState('');
  const [newKeyRole, setNewKeyRole] = useState<ApiKeyRole | ''>('');
  const [newKeyExpiry, setNewKeyExpiry] = useState('');
  const [createdKey, setCreatedKey] = useState<CreateApiKeyResponse | null>(null);

  // Revoke dialog state
  const [revokeTarget, setRevokeTarget] = useState<ApiKey | null>(null);

  // Delete dialog state
  const [deleteTarget, setDeleteTarget] = useState<ApiKey | null>(null);

  // Available roles based on current user's role
  const availableRoles = user?.role
    ? roleHierarchy[user.role as UserRole] ?? []
    : [];

  // ── Create key ──

  const handleCreateOpen = () => {
    setNewKeyName('');
    setNewKeyRole('');
    setNewKeyExpiry('');
    setCreatedKey(null);
    setCreateOpen(true);
  };

  const canCreate =
    newKeyName.trim().length > 0 &&
    newKeyRole !== '' &&
    !createApiKey.isPending;

  const handleCreate = () => {
    if (!canCreate) return;

    const req: CreateApiKeyRequest = {
      name: newKeyName.trim(),
      role: newKeyRole as ApiKeyRole,
    };

    if (newKeyExpiry) {
      const days = parseInt(newKeyExpiry, 10);
      if (!isNaN(days) && days > 0) {
        req.expires_in = days * 24 * 60 * 60; // seconds
      }
    }

    createApiKey.mutate(req, {
      onSuccess: (data) => {
        setCreatedKey(data);
        toast.success('API key created successfully');
      },
      onError: (err: Error) => {
        toast.error(`Failed to create API key: ${err.message}`);
      },
    });
  };

  const handleCopyKey = async () => {
    if (!createdKey?.key) return;
    try {
      await navigator.clipboard.writeText(createdKey.key);
      toast.success('API key copied to clipboard');
    } catch {
      toast.error('Failed to copy to clipboard');
    }
  };

  // ── Revoke key ──

  const handleRevoke = () => {
    if (!revokeTarget) return;
    revokeApiKey.mutate(revokeTarget.id, {
      onSuccess: () => {
        toast.success('API key revoked');
        setRevokeTarget(null);
      },
      onError: (err: Error) => {
        toast.error(`Failed to revoke API key: ${err.message}`);
      },
    });
  };

  // ── Delete key ──

  const handleDelete = () => {
    if (!deleteTarget) return;
    deleteApiKey.mutate(deleteTarget.id, {
      onSuccess: () => {
        toast.success('API key deleted');
        setDeleteTarget(null);
      },
      onError: (err: Error) => {
        toast.error(`Failed to delete API key: ${err.message}`);
      },
    });
  };

  // ── Render ──

  return (
    <div data-testid="my-apikeys-page">
      <PageBreadcrumbs items={[{ label: 'My API Keys' }]} />

      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">My API Keys</h1>
        <Button onClick={handleCreateOpen} data-testid="create-apikey-btn">
          <Plus className="mr-1.5 h-4 w-4" />
          Create API Key
        </Button>
      </div>

      {/* ── Keys Table ── */}
      {isLoading ? (
        <p className="text-sm text-muted-foreground">Loading API keys...</p>
      ) : myKeys.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-lg border border-dashed p-12 text-center">
          <Key className="mb-3 h-10 w-10 text-muted-foreground/60" />
          <p className="text-sm font-medium text-muted-foreground">
            You have no API keys yet.
          </p>
          <p className="mt-1 text-xs text-muted-foreground/70">
            Create one to authenticate programmatically.
          </p>
        </div>
      ) : (
        <div className="rounded-md border" data-testid="my-apikeys-table">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Key Prefix</TableHead>
                <TableHead>Role</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Created</TableHead>
                <TableHead>Expires</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {myKeys.map((key) => {
                const status = getKeyStatus(key);
                return (
                  <TableRow key={key.id} data-testid={`apikey-row-${key.id}`}>
                    <TableCell className="font-medium">{key.name}</TableCell>
                    <TableCell>
                      <code className="rounded bg-muted px-1.5 py-0.5 text-xs">
                        {key.key_prefix}...
                      </code>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">{formatRole(key.role)}</Badge>
                    </TableCell>
                    <TableCell>
                      <Badge variant={statusBadgeVariant[status]}>
                        {status.charAt(0).toUpperCase() + status.slice(1)}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {formatDate(key.created_at)}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {key.expires_at ? formatDate(key.expires_at) : 'Never'}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-1">
                        {status === 'active' && (
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => setRevokeTarget(key)}
                            data-testid={`revoke-apikey-${key.id}`}
                          >
                            <Ban className="mr-1 h-3.5 w-3.5" />
                            Revoke
                          </Button>
                        )}
                        <Button
                          variant="ghost"
                          size="sm"
                          className="text-destructive hover:text-destructive"
                          onClick={() => setDeleteTarget(key)}
                          data-testid={`delete-apikey-${key.id}`}
                        >
                          <Trash2 className="mr-1 h-3.5 w-3.5" />
                          Delete
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      )}

      {/* ── Create Dialog ── */}
      <Dialog
        open={createOpen}
        onOpenChange={(open) => {
          if (!open) {
            setCreateOpen(false);
            setCreatedKey(null);
          }
        }}
      >
        <DialogContent data-testid="create-apikey-dialog">
          <DialogHeader>
            <DialogTitle>
              {createdKey ? 'API Key Created' : 'Create API Key'}
            </DialogTitle>
            <DialogDescription>
              {createdKey
                ? 'Copy and save this key now. It will not be shown again.'
                : 'Create a new API key for programmatic access.'}
            </DialogDescription>
          </DialogHeader>

          {createdKey ? (
            /* ── One-time key display ── */
            <div className="space-y-4">
              <Alert
                className="border-yellow-500/50 bg-yellow-50 text-yellow-900 dark:bg-yellow-950/30 dark:text-yellow-200"
              >
                <AlertTriangle className="h-4 w-4" />
                <AlertDescription>
                  This is the only time the full key will be displayed. Make sure
                  to copy it now.
                </AlertDescription>
              </Alert>

              <div className="space-y-2">
                <Label>API Key</Label>
                <div className="flex items-center gap-2">
                  <Input
                    readOnly
                    value={createdKey.key}
                    className="font-mono text-sm"
                    data-testid="created-apikey-value"
                  />
                  <Button
                    variant="outline"
                    size="icon"
                    onClick={handleCopyKey}
                    data-testid="copy-apikey-btn"
                  >
                    <Copy className="h-4 w-4" />
                  </Button>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <span className="text-muted-foreground">Name:</span>{' '}
                  <span className="font-medium">{createdKey.name}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">Role:</span>{' '}
                  <Badge variant="outline">{formatRole(createdKey.role)}</Badge>
                </div>
              </div>

              <DialogFooter>
                <Button
                  onClick={() => {
                    setCreateOpen(false);
                    setCreatedKey(null);
                  }}
                >
                  Done
                </Button>
              </DialogFooter>
            </div>
          ) : (
            /* ── Create form ── */
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="key-name">Key Name</Label>
                <Input
                  id="key-name"
                  value={newKeyName}
                  onChange={(e) => setNewKeyName(e.target.value)}
                  placeholder="e.g., ci-pipeline, monitoring"
                  data-testid="apikey-name-input"
                />
              </div>

              <div className="space-y-2">
                <Label>Role</Label>
                <Select
                  value={newKeyRole}
                  onValueChange={(v) => setNewKeyRole(v as ApiKeyRole)}
                >
                  <SelectTrigger data-testid="apikey-role-select">
                    <SelectValue placeholder="Select a role" />
                  </SelectTrigger>
                  <SelectContent>
                    {availableRoles.map((role) => (
                      <SelectItem key={role} value={role}>
                        {formatRole(role)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="key-expiry">
                  Expiry (days){' '}
                  <span className="text-muted-foreground">(optional)</span>
                </Label>
                <Input
                  id="key-expiry"
                  type="number"
                  min={1}
                  value={newKeyExpiry}
                  onChange={(e) => setNewKeyExpiry(e.target.value)}
                  placeholder="Leave blank for no expiry"
                  data-testid="apikey-expiry-input"
                />
              </div>

              <DialogFooter>
                <Button
                  variant="outline"
                  onClick={() => setCreateOpen(false)}
                >
                  Cancel
                </Button>
                <Button
                  onClick={handleCreate}
                  disabled={!canCreate}
                  data-testid="apikey-create-submit-btn"
                >
                  {createApiKey.isPending ? 'Creating...' : 'Create Key'}
                </Button>
              </DialogFooter>
            </div>
          )}
        </DialogContent>
      </Dialog>

      {/* ── Revoke Confirm ── */}
      <ConfirmDialog
        open={!!revokeTarget}
        onOpenChange={(open) => {
          if (!open) setRevokeTarget(null);
        }}
        title="Revoke API Key"
        description={`Are you sure you want to revoke the key "${revokeTarget?.name ?? ''}"? This will immediately disable authentication using this key.`}
        confirmLabel="Revoke"
        destructive
        onConfirm={handleRevoke}
        isLoading={revokeApiKey.isPending}
      />

      {/* ── Delete Confirm ── */}
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null);
        }}
        title="Delete API Key"
        description={`Are you sure you want to permanently delete the key "${deleteTarget?.name ?? ''}"? This action cannot be undone.`}
        confirmLabel="Delete"
        destructive
        confirmText={deleteTarget?.name}
        onConfirm={handleDelete}
        isLoading={deleteApiKey.isPending}
      />
    </div>
  );
}
