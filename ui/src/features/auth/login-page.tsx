import { useState, type FormEvent } from 'react';
import { useAuth } from '@/context/auth-context';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Separator } from '@/components/ui/separator';
import { ApiClientError } from '@/api/client';
import { ExternalLink } from 'lucide-react';

export function LoginPage({ onSuccess }: { onSuccess?: () => void }) {
  const { login, loginWithKey, authConfig, isAuthenticated } = useAuth();
  const [mode, setMode] = useState<'password' | 'apikey'>('password');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [apiKey, setApiKey] = useState('');
  const [error, setError] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  // If already authenticated, caller should redirect
  if (isAuthenticated) {
    return null;
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setIsSubmitting(true);

    try {
      if (mode === 'password') {
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
      } else {
        if (!apiKey.trim()) {
          setError('API key is required');
          setIsSubmitting(false);
          return;
        }
        await loginWithKey(apiKey);
      }
      onSuccess?.();
    } catch (err) {
      if (err instanceof ApiClientError) {
        if (err.status === 401) {
          setError(mode === 'password' ? 'Invalid username or password' : 'Invalid API key');
        } else if (err.status === 429) {
          setError('Too many login attempts. Please wait and try again.');
        } else {
          setError(err.message || 'Unable to connect to the registry. Please try again.');
        }
      } else {
        setError('Unable to connect to the registry. Please try again.');
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  const showApiKeyOption = authConfig?.methods?.includes('api_key');
  const showSsoOption = authConfig?.methods?.includes('oidc');

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

            {mode === 'password' ? (
              <>
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
              </>
            ) : (
              <div className="mb-6 space-y-2">
                <Label htmlFor="apikey">API Key</Label>
                <Input
                  id="apikey"
                  type="password"
                  placeholder="Enter your API key"
                  value={apiKey}
                  onChange={(e) => setApiKey(e.target.value)}
                  autoFocus
                  data-testid="login-apikey-input"
                />
              </div>
            )}

            <Button
              type="submit"
              className="w-full"
              disabled={isSubmitting}
              data-testid="login-submit-btn"
            >
              {isSubmitting ? 'Signing in...' : 'Sign In'}
            </Button>

            {showApiKeyOption && (
              <div className="mt-4 text-center">
                <button
                  type="button"
                  className="text-sm text-muted-foreground hover:text-foreground underline"
                  onClick={() => {
                    setMode(mode === 'password' ? 'apikey' : 'password');
                    setError('');
                  }}
                  data-testid="login-toggle-mode-btn"
                >
                  {mode === 'password' ? 'Use API Key instead' : 'Use username and password'}
                </button>
              </div>
            )}

            {showSsoOption && (
              <>
                <div className="relative my-6">
                  <Separator />
                  <span className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 bg-card px-2 text-xs text-muted-foreground">
                    or
                  </span>
                </div>
                <Button
                  type="button"
                  variant="outline"
                  className="w-full"
                  onClick={() => {
                    window.location.href = '/ui/auth/oidc/login';
                  }}
                  data-testid="login-sso-btn"
                >
                  <ExternalLink className="mr-2 h-4 w-4" />
                  Sign in with SSO
                </Button>
              </>
            )}
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
