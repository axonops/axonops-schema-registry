import { useState, type FormEvent } from 'react';
import { useAuth } from '@/context/auth-context';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { ApiClientError } from '@/api/client';

export function LoginPage({ onSuccess }: { onSuccess?: () => void }) {
  const { login, isAuthenticated } = useAuth();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  if (isAuthenticated) {
    return null;
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setIsSubmitting(true);

    try {
      if (!username.trim()) {
        setError('Username is required');
        setIsSubmitting(false);
        return;
      }
      if (!password) {
        setError('Password is required');
        setIsSubmitting(false);
        return;
      }
      await login(username, password);
      onSuccess?.();
    } catch (err) {
      if (err instanceof ApiClientError) {
        if (err.status === 401) {
          setError('Invalid username or password');
        } else {
          setError(err.message || 'Unable to connect to the server. Please try again.');
        }
      } else {
        setError('Unable to connect to the server. Please try again.');
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center p-4" data-testid="login-page">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl" data-testid="login-title">
            Schema Registry
          </CardTitle>
          <CardDescription>
            Sign in to manage your schemas
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} data-testid="login-form">
            {error && (
              <div
                className="mb-4 rounded-md bg-destructive/10 p-3 text-sm text-destructive"
                data-testid="login-error"
              >
                {error}
              </div>
            )}

            <div className="mb-4 space-y-2">
              <Label htmlFor="username">Username</Label>
              <Input
                id="username"
                type="text"
                placeholder="Enter your username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                autoComplete="username"
                autoFocus
                data-testid="login-username-input"
              />
            </div>
            <div className="mb-6 space-y-2">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                type="password"
                placeholder="Enter your password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoComplete="current-password"
                data-testid="login-password-input"
              />
            </div>

            <Button
              type="submit"
              className="w-full"
              disabled={isSubmitting}
              data-testid="login-submit-btn"
            >
              {isSubmitting ? 'Signing in...' : 'Sign In'}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
