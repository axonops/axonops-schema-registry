import { useContexts } from '@/api/queries';
import { PageBreadcrumbs } from '@/components/shared/breadcrumbs';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Loader2, Layers } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';

const breadcrumbs = [{ label: 'Contexts' }];

export function ContextsPage() {
  const { data: contexts, isLoading, isError, error } = useContexts();

  if (isLoading) {
    return (
      <div>
        <PageBreadcrumbs items={breadcrumbs} />
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      </div>
    );
  }

  if (isError) {
    return (
      <div>
        <PageBreadcrumbs items={breadcrumbs} />
        <Alert variant="destructive">
          <AlertDescription>
            {error instanceof Error ? error.message : 'Failed to load contexts'}
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  const contextList = contexts ?? [];
  const defaultContext = contextList.find((c) => c === '.' || c === '' || c === ':');
  const namedContexts = contextList.filter((c) => c !== '.' && c !== '' && c !== ':');

  return (
    <div data-testid="contexts-page">
      <PageBreadcrumbs items={breadcrumbs} />

      <div className="space-y-6">
        <Card data-testid="contexts-summary-card">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Layers className="h-5 w-5" />
              Schema Contexts
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <p className="text-sm text-muted-foreground">
              Contexts are logical namespaces that group subjects. The default context
              contains all subjects not assigned to a specific named context.
            </p>
            <div className="flex gap-2">
              <Badge variant="outline">{contextList.length} context{contextList.length !== 1 ? 's' : ''}</Badge>
            </div>
          </CardContent>
        </Card>

        {defaultContext !== undefined && (
          <Card data-testid="contexts-default-card">
            <CardHeader>
              <CardTitle className="text-sm font-medium">Default Context</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-center gap-2">
                <Badge variant="secondary">default</Badge>
                <span className="text-sm text-muted-foreground">
                  Contains all subjects not assigned to a named context
                </span>
              </div>
            </CardContent>
          </Card>
        )}

        <Card data-testid="contexts-named-card">
          <CardHeader>
            <CardTitle className="text-sm font-medium">Named Contexts</CardTitle>
          </CardHeader>
          <CardContent>
            {namedContexts.length === 0 ? (
              <p className="text-sm text-muted-foreground py-4 text-center">
                No named contexts. All subjects belong to the default context.
              </p>
            ) : (
              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                {namedContexts.map((ctx) => (
                  <div
                    key={ctx}
                    className="flex items-center gap-3 rounded-lg border p-4"
                    data-testid={`context-item-${ctx}`}
                  >
                    <Layers className="h-4 w-4 text-muted-foreground" />
                    <div>
                      <div className="font-medium text-sm">{ctx}</div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
