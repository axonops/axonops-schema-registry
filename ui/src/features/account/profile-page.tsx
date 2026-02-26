import { useState } from 'react';
import { useAuth } from '@/context/auth-context';
import { useChangePassword } from '@/api/queries';
import { PageBreadcrumbs } from '@/components/shared/breadcrumbs';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Separator } from '@/components/ui/separator';
import { toast } from 'sonner';
import { User, Lock, Loader2 } from 'lucide-react';

// ── Helpers ──

const roleBadgeVariant: Record<string, 'default' | 'secondary' | 'outline'> = {
  super_admin: 'default',
  admin: 'default',
  developer: 'secondary',
  readonly: 'outline',
};

function formatRole(role: string): string {
  return role
    .split('_')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}

function formatAuthMethod(method: string): string {
  switch (method) {
    case 'ldap':
      return 'LDAP';
    case 'oidc':
      return 'OIDC / SSO';
    case 'api_key':
      return 'API Key';
    case 'basic':
      return 'Local (Password)';
    case 'dev-bypass':
      return 'Dev Bypass';
    default:
      return method;
  }
}

// ── Component ──

export function ProfilePage() {
  const { user } = useAuth();
  const changePassword = useChangePassword();

  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');

  // Validation
  const passwordMismatch =
    confirmPassword.length > 0 && newPassword !== confirmPassword;
  const passwordTooShort =
    newPassword.length > 0 && newPassword.length < 8;
  const canSubmitPassword =
    currentPassword.trim().length > 0 &&
    newPassword.length >= 8 &&
    newPassword === confirmPassword &&
    !changePassword.isPending;

  const isExternalAuth =
    user?.auth_method === 'ldap' || user?.auth_method === 'oidc';

  const handlePasswordSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSubmitPassword) return;

    changePassword.mutate(
      {
        current_password: currentPassword,
        new_password: newPassword,
      },
      {
        onSuccess: () => {
          toast.success('Password changed successfully');
          setCurrentPassword('');
          setNewPassword('');
          setConfirmPassword('');
        },
        onError: (err: Error) => {
          toast.error(`Failed to change password: ${err.message}`);
        },
      }
    );
  };

  return (
    <div data-testid="profile-page">
      <PageBreadcrumbs items={[{ label: 'My Profile' }]} />

      <h1 className="mb-6 text-2xl font-bold">My Profile</h1>

      {/* ── Account Information ── */}
      <Card className="mb-6" data-testid="profile-info">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <User className="h-5 w-5" />
            Account Information
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2">
            {/* Username */}
            <div className="space-y-1">
              <Label className="text-muted-foreground">Username</Label>
              <p className="text-sm font-medium">{user?.username ?? '—'}</p>
            </div>

            {/* Email */}
            <div className="space-y-1">
              <Label className="text-muted-foreground">Email</Label>
              <p className="text-sm font-medium">{user?.email || '—'}</p>
            </div>

            {/* Role */}
            <div className="space-y-1">
              <Label className="text-muted-foreground">Role</Label>
              <div>
                {user?.role ? (
                  <Badge
                    variant={roleBadgeVariant[user.role] ?? 'outline'}
                  >
                    {formatRole(user.role)}
                  </Badge>
                ) : (
                  <span className="text-sm">—</span>
                )}
              </div>
            </div>

            {/* Auth Method */}
            <div className="space-y-1">
              <Label className="text-muted-foreground">
                Authentication Method
              </Label>
              <p className="text-sm font-medium">
                {user?.auth_method
                  ? formatAuthMethod(user.auth_method)
                  : '—'}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* ── Change Password ── */}
      {!isExternalAuth && (
        <>
          <Separator className="mb-6" />

          <Card data-testid="profile-change-password-section">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Lock className="h-5 w-5" />
                Change Password
              </CardTitle>
            </CardHeader>
            <CardContent>
              <form onSubmit={handlePasswordSubmit} className="max-w-md space-y-4">
                {/* Current Password */}
                <div className="space-y-2">
                  <Label htmlFor="current-password">Current Password</Label>
                  <Input
                    id="current-password"
                    type="password"
                    value={currentPassword}
                    onChange={(e) => setCurrentPassword(e.target.value)}
                    placeholder="Enter your current password"
                    autoComplete="current-password"
                    data-testid="profile-current-password-input"
                  />
                </div>

                {/* New Password */}
                <div className="space-y-2">
                  <Label htmlFor="new-password">New Password</Label>
                  <Input
                    id="new-password"
                    type="password"
                    value={newPassword}
                    onChange={(e) => setNewPassword(e.target.value)}
                    placeholder="Minimum 8 characters"
                    autoComplete="new-password"
                    data-testid="profile-new-password-input"
                  />
                  {passwordTooShort && (
                    <p className="text-xs text-destructive">
                      Password must be at least 8 characters.
                    </p>
                  )}
                </div>

                {/* Confirm New Password */}
                <div className="space-y-2">
                  <Label htmlFor="confirm-password">
                    Confirm New Password
                  </Label>
                  <Input
                    id="confirm-password"
                    type="password"
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    placeholder="Re-enter your new password"
                    autoComplete="new-password"
                    data-testid="profile-confirm-password-input"
                  />
                  {passwordMismatch && (
                    <p className="text-xs text-destructive">
                      Passwords do not match.
                    </p>
                  )}
                </div>

                {/* Submit */}
                <Button
                  type="submit"
                  disabled={!canSubmitPassword}
                  data-testid="profile-password-submit-btn"
                >
                  {changePassword.isPending && (
                    <Loader2 className="mr-1.5 h-4 w-4 animate-spin" />
                  )}
                  Update Password
                </Button>
              </form>
            </CardContent>
          </Card>
        </>
      )}
    </div>
  );
}
