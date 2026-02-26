import { useState, useMemo } from 'react';
import { useUsers, useCreateUser, useUpdateUser, useDeleteUser } from '@/api/queries';
import type { User, CreateUserRequest, UpdateUserRequest } from '@/api/queries';
import { useAuth } from '@/context/auth-context';
import { PageBreadcrumbs } from '@/components/shared/breadcrumbs';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Switch } from '@/components/ui/switch';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Skeleton } from '@/components/ui/skeleton';
import { toast } from 'sonner';
import { Plus, Pencil, Trash2, Search, AlertCircle, RefreshCw } from 'lucide-react';

// ── Types ──

type UserRole = User['role'];

interface UserFormState {
  username: string;
  email: string;
  password: string;
  role: UserRole;
  enabled: boolean;
}

const INITIAL_FORM_STATE: UserFormState = {
  username: '',
  email: '',
  password: '',
  role: 'developer',
  enabled: true,
};

// ── Role Badge helpers ──

function roleBadgeVariant(role: UserRole): 'destructive' | 'default' | 'secondary' | 'outline' {
  switch (role) {
    case 'super_admin':
      return 'destructive';
    case 'admin':
      return 'default';
    case 'developer':
      return 'secondary';
    case 'readonly':
      return 'outline';
  }
}

function roleDisplayLabel(role: UserRole): string {
  switch (role) {
    case 'super_admin':
      return 'Super Admin';
    case 'admin':
      return 'Admin';
    case 'developer':
      return 'Developer';
    case 'readonly':
      return 'Read Only';
  }
}

// ── Available roles based on current user's role ──

function getAvailableRoles(currentUserRole: UserRole | undefined): UserRole[] {
  if (currentUserRole === 'super_admin') {
    return ['super_admin', 'admin', 'developer', 'readonly'];
  }
  // Admin users cannot assign or see super_admin
  return ['admin', 'developer', 'readonly'];
}

// ── Component ──

export function UsersPage() {
  const { user: currentUser } = useAuth();
  const { data: users, isLoading, isError, error, refetch } = useUsers();
  const createUser = useCreateUser();
  const updateUser = useUpdateUser();
  const deleteUser = useDeleteUser();

  // Search
  const [searchQuery, setSearchQuery] = useState('');

  // Dialog state
  const [formOpen, setFormOpen] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [formState, setFormState] = useState<UserFormState>(INITIAL_FORM_STATE);

  // Delete confirmation
  const [deleteTarget, setDeleteTarget] = useState<User | null>(null);

  const isSuperAdmin = currentUser?.role === 'super_admin';
  const availableRoles = getAvailableRoles(currentUser?.role);

  // ── Filtered users ──

  const filteredUsers = useMemo(() => {
    if (!users) return [];

    let visible = users;

    // Admin users cannot see super_admin users
    if (!isSuperAdmin) {
      visible = visible.filter((u) => u.role !== 'super_admin');
    }

    if (!searchQuery.trim()) return visible;

    const query = searchQuery.toLowerCase().trim();
    return visible.filter(
      (u) =>
        u.username.toLowerCase().includes(query) ||
        u.email.toLowerCase().includes(query)
    );
  }, [users, searchQuery, isSuperAdmin]);

  // ── Form handlers ──

  const openCreateDialog = () => {
    setEditingUser(null);
    setFormState(INITIAL_FORM_STATE);
    setFormOpen(true);
  };

  const openEditDialog = (user: User) => {
    setEditingUser(user);
    setFormState({
      username: user.username,
      email: user.email,
      password: '',
      role: user.role,
      enabled: user.enabled,
    });
    setFormOpen(true);
  };

  const closeFormDialog = () => {
    setFormOpen(false);
    setEditingUser(null);
    setFormState(INITIAL_FORM_STATE);
  };

  const handleFormSubmit = () => {
    if (editingUser) {
      // Update existing user
      const payload: UpdateUserRequest & { id: number } = {
        id: editingUser.id,
        email: formState.email,
        role: formState.role,
        enabled: formState.enabled,
      };
      if (formState.password.trim()) {
        payload.password = formState.password;
      }
      updateUser.mutate(payload, {
        onSuccess: () => {
          toast.success(`User "${editingUser.username}" updated successfully`);
          closeFormDialog();
        },
        onError: (err: Error) => {
          toast.error(`Failed to update user: ${err.message}`);
        },
      });
    } else {
      // Create new user
      const payload: CreateUserRequest = {
        username: formState.username,
        password: formState.password,
        email: formState.email,
        role: formState.role as CreateUserRequest['role'],
        enabled: formState.enabled,
      };
      createUser.mutate(payload, {
        onSuccess: () => {
          toast.success(`User "${formState.username}" created successfully`);
          closeFormDialog();
        },
        onError: (err: Error) => {
          toast.error(`Failed to create user: ${err.message}`);
        },
      });
    }
  };

  const handleDeleteConfirm = () => {
    if (!deleteTarget) return;
    deleteUser.mutate(deleteTarget.id, {
      onSuccess: () => {
        toast.success(`User "${deleteTarget.username}" deleted successfully`);
        setDeleteTarget(null);
      },
      onError: (err: Error) => {
        toast.error(`Failed to delete user: ${err.message}`);
      },
    });
  };

  const isFormValid = editingUser
    ? formState.email.trim() !== ''
    : formState.username.trim() !== '' &&
      formState.email.trim() !== '' &&
      formState.password.trim() !== '';

  const isFormSubmitting = createUser.isPending || updateUser.isPending;

  // ── Render ──

  return (
    <div data-testid="users-page">
      <PageBreadcrumbs items={[{ label: 'Users' }]} />

      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">Users</h1>
        <Button onClick={openCreateDialog} data-testid="users-create-btn">
          <Plus className="mr-1.5 h-4 w-4" />
          Create User
        </Button>
      </div>

      {/* Search */}
      <div className="relative mb-4 max-w-sm">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          placeholder="Search by username or email..."
          className="pl-9"
          data-testid="users-search-input"
        />
      </div>

      {/* Loading state */}
      {isLoading && (
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      )}

      {/* Error state */}
      {isError && (
        <div className="flex flex-col items-center justify-center gap-4 rounded-lg border border-destructive/30 bg-destructive/5 py-12 text-center">
          <AlertCircle className="h-10 w-10 text-destructive" />
          <div>
            <p className="font-medium text-destructive">Failed to load users</p>
            <p className="mt-1 text-sm text-muted-foreground">
              {error instanceof Error ? error.message : 'An unexpected error occurred'}
            </p>
          </div>
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCw className="mr-1.5 h-4 w-4" />
            Retry
          </Button>
        </div>
      )}

      {/* Users table */}
      {!isLoading && !isError && (
        <>
          {filteredUsers.length === 0 ? (
            <div className="flex flex-col items-center justify-center rounded-lg border py-12 text-center text-muted-foreground">
              <p className="font-medium">
                {searchQuery.trim()
                  ? 'No users match your search'
                  : 'No users found'}
              </p>
              <p className="mt-1 text-sm">
                {searchQuery.trim()
                  ? 'Try a different search term.'
                  : 'Create a user to get started.'}
              </p>
            </div>
          ) : (
            <div className="rounded-md border">
              <Table data-testid="users-list-table">
                <TableHeader>
                  <TableRow>
                    <TableHead>Username</TableHead>
                    <TableHead>Email</TableHead>
                    <TableHead>Role</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredUsers.map((u) => (
                    <TableRow key={u.id}>
                      <TableCell className="font-medium">{u.username}</TableCell>
                      <TableCell>{u.email}</TableCell>
                      <TableCell>
                        <Badge variant={roleBadgeVariant(u.role)}>
                          {roleDisplayLabel(u.role)}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Badge variant={u.enabled ? 'outline' : 'secondary'}>
                          {u.enabled ? 'Enabled' : 'Disabled'}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-1">
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => openEditDialog(u)}
                            data-testid="user-edit-btn"
                          >
                            <Pencil className="h-4 w-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => setDeleteTarget(u)}
                            data-testid="user-delete-btn"
                          >
                            <Trash2 className="h-4 w-4 text-destructive" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </>
      )}

      {/* ── Create / Edit Dialog ── */}
      <Dialog open={formOpen} onOpenChange={(open) => { if (!open) closeFormDialog(); }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {editingUser ? `Edit User: ${editingUser.username}` : 'Create User'}
            </DialogTitle>
          </DialogHeader>

          <div className="space-y-4 py-2">
            {/* Username */}
            <div className="space-y-2">
              <Label htmlFor="user-form-username">Username</Label>
              <Input
                id="user-form-username"
                value={formState.username}
                onChange={(e) =>
                  setFormState((prev) => ({ ...prev, username: e.target.value }))
                }
                placeholder="Enter username"
                disabled={!!editingUser}
                data-testid="user-form-username-input"
              />
            </div>

            {/* Email */}
            <div className="space-y-2">
              <Label htmlFor="user-form-email">Email</Label>
              <Input
                id="user-form-email"
                type="email"
                value={formState.email}
                onChange={(e) =>
                  setFormState((prev) => ({ ...prev, email: e.target.value }))
                }
                placeholder="Enter email address"
                data-testid="user-form-email-input"
              />
            </div>

            {/* Password */}
            <div className="space-y-2">
              <Label htmlFor="user-form-password">
                Password
                {editingUser && (
                  <span className="ml-1 text-xs font-normal text-muted-foreground">
                    (leave blank to keep current)
                  </span>
                )}
              </Label>
              <Input
                id="user-form-password"
                type="password"
                value={formState.password}
                onChange={(e) =>
                  setFormState((prev) => ({ ...prev, password: e.target.value }))
                }
                placeholder={editingUser ? 'Leave blank to keep current' : 'Enter password'}
                data-testid="user-form-password-input"
              />
            </div>

            {/* Role */}
            <div className="space-y-2">
              <Label>Role</Label>
              <Select
                value={formState.role}
                onValueChange={(value) =>
                  setFormState((prev) => ({ ...prev, role: value as UserRole }))
                }
              >
                <SelectTrigger data-testid="user-form-role-select">
                  <SelectValue placeholder="Select a role" />
                </SelectTrigger>
                <SelectContent>
                  {availableRoles.map((role) => (
                    <SelectItem key={role} value={role}>
                      {roleDisplayLabel(role)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {/* Enabled toggle */}
            <div className="flex items-center justify-between rounded-lg border p-3">
              <div>
                <Label htmlFor="user-form-enabled" className="cursor-pointer">
                  Enabled
                </Label>
                <p className="text-xs text-muted-foreground">
                  Disabled users cannot log in or access the registry.
                </p>
              </div>
              <Switch
                id="user-form-enabled"
                checked={formState.enabled}
                onCheckedChange={(checked) =>
                  setFormState((prev) => ({ ...prev, enabled: checked }))
                }
                data-testid="user-form-enabled-toggle"
              />
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={closeFormDialog}>
              Cancel
            </Button>
            <Button
              onClick={handleFormSubmit}
              disabled={!isFormValid || isFormSubmitting}
              data-testid="user-form-submit-btn"
            >
              {isFormSubmitting
                ? 'Saving...'
                : editingUser
                  ? 'Update User'
                  : 'Create User'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ── Delete Confirmation Dialog ── */}
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => { if (!open) setDeleteTarget(null); }}
        title="Delete User"
        description={`This will permanently delete the user "${deleteTarget?.username ?? ''}". This action cannot be undone.`}
        confirmLabel="Delete User"
        destructive
        confirmText={deleteTarget?.username}
        onConfirm={handleDeleteConfirm}
        isLoading={deleteUser.isPending}
      />
    </div>
  );
}
