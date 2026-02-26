import { useParams, useNavigate } from '@tanstack/react-router';
import { useSchemaById, useSchemaSubjects } from '@/api/queries';
import { PageBreadcrumbs } from '@/components/shared/breadcrumbs';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { Copy, Download, AlertCircle, RefreshCw } from 'lucide-react';
import { toast } from 'sonner';

export function SchemaByIdPage() {
  const { id: idStr } = useParams({ strict: false }) as { id: string };
  const id = parseInt(idStr, 10);
  const { data: schema, isLoading, isError, error, refetch } = useSchemaById(id);
  const { data: subjects } = useSchemaSubjects(id);
  const navigate = useNavigate();

  const breadcrumbs = [
    { label: 'Schema Browser', href: '/ui/schemas' },
    { label: `Schema ${idStr}` },
  ];

  const handleCopy = () => {
    if (!schema) return;
    navigator.clipboard.writeText(formatSchema(schema.schema));
    toast.success('Schema copied to clipboard');
  };

  const handleDownload = () => {
    if (!schema) return;
    const ext = schema.schemaType === 'PROTOBUF' ? '.proto' : schema.schemaType === 'JSON' ? '.json' : '.avsc';
    const blob = new Blob([formatSchema(schema.schema)], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `schema-${id}${ext}`;
    a.click();
    URL.revokeObjectURL(url);
  };

  if (isLoading) {
    return (
      <div>
        <PageBreadcrumbs items={breadcrumbs} />
        <div className="space-y-4">
          <Skeleton className="h-8 w-48" />
          <Skeleton className="h-64 w-full" />
        </div>
      </div>
    );
  }

  if (isError) {
    return (
      <div>
        <PageBreadcrumbs items={breadcrumbs} />
        <div className="flex flex-col items-center justify-center gap-4 py-12">
          <AlertCircle className="h-10 w-10 text-destructive" />
          <p className="text-muted-foreground">
            {error instanceof Error ? error.message : 'Schema not found'}
          </p>
          <Button variant="outline" onClick={() => refetch()}>
            <RefreshCw className="mr-2 h-4 w-4" /> Retry
          </Button>
        </div>
      </div>
    );
  }

  if (!schema) return null;

  return (
    <div data-testid="schema-by-id-page">
      <PageBreadcrumbs items={breadcrumbs} />

      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Schema ID: {id}</h1>
          <Badge className="mt-1" data-testid="schema-by-id-type">{schema.schemaType}</Badge>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={handleCopy} data-testid="schema-copy-btn">
            <Copy className="mr-1 h-4 w-4" /> Copy
          </Button>
          <Button variant="outline" size="sm" onClick={handleDownload} data-testid="schema-download-btn">
            <Download className="mr-1 h-4 w-4" /> Download
          </Button>
        </div>
      </div>

      {/* Used in subjects */}
      {subjects && subjects.length > 0 && (
        <Card className="mb-6">
          <CardHeader>
            <CardTitle className="text-sm font-medium">Used in subjects</CardTitle>
          </CardHeader>
          <CardContent>
            <ul className="space-y-1" data-testid="schema-subjects-list">
              {subjects.map((sv) => (
                <li key={`${sv.subject}-${sv.version}`}>
                  <button
                    className="text-sm text-primary hover:underline"
                    onClick={() => navigate({
                      to: '/ui/subjects/$subject/versions/$version',
                      params: { subject: sv.subject, version: String(sv.version) },
                    })}
                  >
                    {sv.subject} v{sv.version}
                  </button>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}

      {/* Schema content */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium">Schema content</CardTitle>
        </CardHeader>
        <CardContent>
          <pre
            className="max-h-96 overflow-auto rounded bg-muted p-4 text-sm"
            data-testid="schema-viewer"
          >
            {formatSchema(schema.schema)}
          </pre>
        </CardContent>
      </Card>
    </div>
  );
}

function formatSchema(schema: string): string {
  try {
    return JSON.stringify(JSON.parse(schema), null, 2);
  } catch {
    return schema;
  }
}
