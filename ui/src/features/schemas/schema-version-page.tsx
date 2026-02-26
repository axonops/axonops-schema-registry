import { useState } from 'react';
import { useParams, useNavigate } from '@tanstack/react-router';
import { useSubjectVersion, useSubjectVersions, useReferencedBy, useDeleteVersion, queryKeys, type SubjectVersion } from '@/api/queries';
import { useQuery } from '@tanstack/react-query';
import { apiFetch } from '@/api/client';
import { PageBreadcrumbs } from '@/components/shared/breadcrumbs';
import { SchemaEditor } from '@/components/schema-editor/schema-editor';
import { SchemaDiffViewer } from '@/components/schema-editor/schema-diff-viewer';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import type { SchemaType } from '@/components/schema-editor/monaco-config';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Copy, Download, AlertCircle, RefreshCw, Trash2, GitCompareArrows } from 'lucide-react';
import { toast } from 'sonner';

export function SchemaVersionPage() {
  const { subject, version: versionStr } = useParams({ strict: false }) as {
    subject: string;
    version: string;
  };
  const version = parseInt(versionStr, 10);
  const { data, isLoading, isError, error, refetch } = useSubjectVersion(subject, version);
  const { data: allVersions } = useSubjectVersions(subject);
  const { data: referencedBy } = useReferencedBy(subject, version);
  const navigate = useNavigate();
  const deleteMutation = useDeleteVersion(subject);

  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [deletePermanent, setDeletePermanent] = useState(false);
  const [showDiff, setShowDiff] = useState(false);
  const [diffVersion, setDiffVersion] = useState<string>('');

  // Load the comparison version for diff (only when a version is selected)
  const diffVersionNum = diffVersion ? parseInt(diffVersion, 10) : 0;
  const { data: diffData } = useQuery({
    queryKey: queryKeys.subjects.version(subject, diffVersionNum),
    queryFn: () => apiFetch<SubjectVersion>(
      `/subjects/${encodeURIComponent(subject)}/versions/${diffVersionNum}`
    ),
    enabled: !!subject && diffVersionNum > 0,
  });

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

  const handleDelete = (permanent: boolean) => {
    setDeletePermanent(permanent);
    setShowDeleteDialog(true);
  };

  const confirmDelete = () => {
    deleteMutation.mutate({ version, permanent: deletePermanent }, {
      onSuccess: () => {
        toast.success(
          deletePermanent
            ? `Permanently deleted version ${version}`
            : `Soft-deleted version ${version}`
        );
        setShowDeleteDialog(false);
        navigate({
          to: '/ui/subjects/$subject',
          params: { subject },
        });
      },
      onError: (err) => {
        toast.error(err instanceof Error ? err.message : 'Failed to delete version');
      },
    });
  };

  // Available versions for diff (all except current)
  const diffVersionOptions = (allVersions ?? []).filter(v => v !== version).sort((a, b) => b - a);

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

  const schemaType = (data.schemaType || 'AVRO') as SchemaType;

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
          <Button
            variant="outline"
            size="sm"
            onClick={() => handleDelete(false)}
            data-testid="version-soft-delete-btn"
          >
            <Trash2 className="mr-1 h-4 w-4" /> Soft Delete
          </Button>
          <Button
            variant="destructive"
            size="sm"
            onClick={() => handleDelete(true)}
            data-testid="version-permanent-delete-btn"
          >
            <Trash2 className="mr-1 h-4 w-4" /> Permanent Delete
          </Button>
        </div>
      </div>

      {/* Schema viewer with Monaco */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle className="text-sm font-medium">Schema</CardTitle>
        </CardHeader>
        <CardContent>
          <SchemaEditor
            value={formatSchema(data.schema)}
            schemaType={schemaType}
            readOnly
            height="350px"
            data-testid="schema-viewer"
          />
        </CardContent>
      </Card>

      {/* Version Diff */}
      {diffVersionOptions.length > 0 && (
        <Card className="mb-6">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-medium">
                <GitCompareArrows className="mr-1 inline h-4 w-4" />
                Compare with another version
              </CardTitle>
              <div className="flex items-center gap-2">
                <Select value={diffVersion} onValueChange={(v) => { setDiffVersion(v); setShowDiff(true); }}>
                  <SelectTrigger className="w-32" data-testid="diff-version-select">
                    <SelectValue placeholder="Select..." />
                  </SelectTrigger>
                  <SelectContent>
                    {diffVersionOptions.map(v => (
                      <SelectItem key={v} value={String(v)}>v{v}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {showDiff && (
                  <Button variant="ghost" size="sm" onClick={() => { setShowDiff(false); setDiffVersion(''); }}>
                    Hide
                  </Button>
                )}
              </div>
            </div>
          </CardHeader>
          {showDiff && diffData && (
            <CardContent>
              <div className="mb-2 flex gap-4 text-xs text-muted-foreground">
                <span>Left: v{diffVersionNum}</span>
                <span>Right: v{version} (current)</span>
              </div>
              <SchemaDiffViewer
                original={formatSchema(diffData.schema)}
                modified={formatSchema(data.schema)}
                schemaType={schemaType}
                height="350px"
                data-testid="schema-diff-viewer"
              />
            </CardContent>
          )}
        </Card>
      )}

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

      <ConfirmDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        title={deletePermanent ? 'Permanently Delete Version' : 'Soft-Delete Version'}
        description={
          deletePermanent
            ? `This will permanently delete version ${version} of "${subject}". This cannot be undone.`
            : `This will soft-delete version ${version} of "${subject}".`
        }
        confirmLabel={deletePermanent ? 'Delete Permanently' : 'Soft Delete'}
        destructive={deletePermanent}
        onConfirm={confirmDelete}
        isLoading={deleteMutation.isPending}
      />
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
