import { useParams, useNavigate } from '@tanstack/react-router';
import { useSubjectVersion, useReferencedBy } from '@/api/queries';
import { PageBreadcrumbs } from '@/components/shared/breadcrumbs';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Copy, Download, AlertCircle, RefreshCw } from 'lucide-react';
import { toast } from 'sonner';

export function SchemaVersionPage() {
  const { subject, version: versionStr } = useParams({ strict: false }) as {
    subject: string;
    version: string;
  };
  const version = parseInt(versionStr, 10);
  const { data, isLoading, isError, error, refetch } = useSubjectVersion(subject, version);
  const { data: referencedBy } = useReferencedBy(subject, version);
  const navigate = useNavigate();

  const breadcrumbs = [
    { label: 'Subjects', href: '/ui/subjects' },
    { label: subject, href: `/ui/subjects/${encodeURIComponent(subject)}` },
    { label: `Version ${versionStr}` },
  ];

  const handleCopy = () => {
    if (!data) return;
    navigator.clipboard.writeText(formatSchema(data.schema));
    toast.success('Schema copied to clipboard');
  };

  const handleDownload = () => {
    if (!data) return;
    const ext = data.schemaType === 'PROTOBUF' ? '.proto' : data.schemaType === 'JSON' ? '.json' : '.avsc';
    const blob = new Blob([formatSchema(data.schema)], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${subject}-v${version}${ext}`;
    a.click();
    URL.revokeObjectURL(url);
  };

  if (isLoading) {
    return (
      <div>
        <PageBreadcrumbs items={breadcrumbs} />
        <div className="space-y-4">
          <Skeleton className="h-8 w-64" />
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
            {error instanceof Error ? error.message : 'Failed to load schema version'}
          </p>
          <Button variant="outline" onClick={() => refetch()}>
            <RefreshCw className="mr-2 h-4 w-4" /> Retry
          </Button>
        </div>
      </div>
    );
  }

  if (!data) return null;

  return (
    <div data-testid="schema-version-page">
      <PageBreadcrumbs items={breadcrumbs} />

      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">
            {subject} <span className="text-muted-foreground">v{version}</span>
          </h1>
          <div className="mt-1 flex items-center gap-2">
            <Badge data-testid="schema-version-type">{data.schemaType}</Badge>
            <Badge variant="outline" data-testid="schema-version-id">ID: {data.id}</Badge>
            <Badge variant="secondary" data-testid="schema-version-status">Active</Badge>
          </div>
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

      {/* Schema viewer */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle className="text-sm font-medium">Schema</CardTitle>
        </CardHeader>
        <CardContent>
          <pre
            className="max-h-96 overflow-auto rounded bg-muted p-4 text-sm"
            data-testid="schema-viewer"
          >
            {formatSchema(data.schema)}
          </pre>
        </CardContent>
      </Card>

      {/* References */}
      {data.references && data.references.length > 0 && (
        <Card className="mb-6">
          <CardHeader>
            <CardTitle className="text-sm font-medium">References</CardTitle>
          </CardHeader>
          <CardContent>
            <ul className="space-y-1" data-testid="schema-references-list">
              {data.references.map((ref) => (
                <li key={`${ref.subject}-${ref.version}`}>
                  <button
                    className="text-sm text-primary hover:underline"
                    onClick={() => navigate({
                      to: '/ui/subjects/$subject/versions/$version',
                      params: { subject: ref.subject, version: String(ref.version) },
                    })}
                  >
                    {ref.name} → {ref.subject} v{ref.version}
                  </button>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}

      {/* Referenced By */}
      {referencedBy && referencedBy.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium">Referenced By</CardTitle>
          </CardHeader>
          <CardContent>
            <ul className="space-y-1" data-testid="schema-referencedby-list">
              {referencedBy.map((schemaId) => (
                <li key={schemaId}>
                  <button
                    className="text-sm text-primary hover:underline"
                    onClick={() => navigate({ to: '/ui/schemas/$id', params: { id: String(schemaId) } })}
                  >
                    Schema ID: {schemaId}
                  </button>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}
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
