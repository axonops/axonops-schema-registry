import { useState } from 'react';
import { useNavigate, useParams } from '@tanstack/react-router';
import { useSubjectVersions, useSubjectVersion, useSubjectConfig, useSubjectMode, useGlobalConfig, useDeleteSubject } from '@/api/queries';
import { PageBreadcrumbs } from '@/components/shared/breadcrumbs';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { AlertCircle, RefreshCw, Plus, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { toast } from 'sonner';

export function SubjectDetailPage() {
  const { subject } = useParams({ strict: false }) as { subject: string };
  const { data: versions, isLoading, isError, error, refetch } = useSubjectVersions(subject);
  const { data: latestVersion } = useSubjectVersion(subject, 'latest');
  const { data: subjectConfig } = useSubjectConfig(subject);
  const { data: subjectMode } = useSubjectMode(subject);
  const { data: globalConfig } = useGlobalConfig();
  const navigate = useNavigate();
  const deleteMutation = useDeleteSubject(subject);

  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [deletePermanent, setDeletePermanent] = useState(false);

  const breadcrumbs = [
    { label: 'Subjects', href: '/ui/subjects' },
    { label: subject },
  ];

  const compatLevel = subjectConfig?.compatibilityLevel ?? globalConfig?.compatibilityLevel ?? 'BACKWARD';
  const modeValue = subjectMode?.mode ?? 'READWRITE';

  const handleDelete = (permanent: boolean) => {
    setDeletePermanent(permanent);
    setShowDeleteDialog(true);
  };

  const confirmDelete = () => {
    deleteMutation.mutate({ permanent: deletePermanent }, {
      onSuccess: (deletedVersions) => {
        toast.success(
          deletePermanent
            ? `Permanently deleted subject "${subject}" (${deletedVersions.length} versions)`
            : `Soft-deleted subject "${subject}" (${deletedVersions.length} versions)`
        );
        setShowDeleteDialog(false);
        navigate({ to: '/ui/subjects' });
      },
      onError: (err) => {
        toast.error(err instanceof Error ? err.message : 'Failed to delete subject');
      },
    });
  };

  if (isLoading) {
    return (
      <div>
        <PageBreadcrumbs items={breadcrumbs} />
        <div className="space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-10 w-full" />
          ))}
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
            {error instanceof Error ? error.message : 'Failed to load subject'}
          </p>
          <Button variant="outline" onClick={() => refetch()}>
            <RefreshCw className="mr-2 h-4 w-4" /> Retry
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div data-testid="subject-detail-page">
      <PageBreadcrumbs items={breadcrumbs} />

      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold" data-testid="subject-detail-name">{subject}</h1>
          <div className="mt-1 flex items-center gap-2">
            <Badge variant="outline" data-testid="subject-detail-compat">
              {compatLevel}
            </Badge>
            <Badge variant="secondary" data-testid="subject-detail-mode">
              {modeValue}
            </Badge>
            {latestVersion && (
              <Badge data-testid="subject-detail-type">
                {latestVersion.schemaType}
              </Badge>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button
            size="sm"
            onClick={() => navigate({
              to: '/ui/subjects/$subject/register',
              params: { subject },
            })}
            data-testid="subject-register-btn"
          >
            <Plus className="mr-1 h-4 w-4" /> Register New Version
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => handleDelete(false)}
            data-testid="subject-soft-delete-btn"
          >
            <Trash2 className="mr-1 h-4 w-4" /> Soft Delete
          </Button>
          <Button
            variant="destructive"
            size="sm"
            onClick={() => handleDelete(true)}
            data-testid="subject-permanent-delete-btn"
          >
            <Trash2 className="mr-1 h-4 w-4" /> Permanent Delete
          </Button>
        </div>
      </div>

      {latestVersion && (
        <div className="mb-6 rounded-md border p-4" data-testid="subject-detail-latest-preview">
          <h3 className="mb-2 text-sm font-medium text-muted-foreground">Latest Schema (v{latestVersion.version})</h3>
          <pre className="max-h-48 overflow-auto rounded bg-muted p-3 text-xs">
            {formatSchema(latestVersion.schema)}
          </pre>
        </div>
      )}

      <div className="mb-2 text-sm text-muted-foreground">
        {versions?.length ?? 0} version{(versions?.length ?? 0) !== 1 ? 's' : ''}
      </div>

      <div className="rounded-md border">
        <Table data-testid="subject-versions-table">
          <TableHeader>
            <TableRow>
              <TableHead>Version</TableHead>
              <TableHead>Schema ID</TableHead>
              <TableHead>Type</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {versions?.sort((a, b) => b - a).map((version) => (
              <TableRow
                key={version}
                className="cursor-pointer"
                onClick={() => navigate({
                  to: '/ui/subjects/$subject/versions/$version',
                  params: { subject, version: String(version) },
                })}
                data-testid={`subject-version-row-${version}`}
              >
                <TableCell>
                  <span className="font-medium">v{version}</span>
                  {latestVersion && version === latestVersion.version && (
                    <Badge variant="outline" className="ml-2 text-xs">latest</Badge>
                  )}
                </TableCell>
                <TableCell>
                  {latestVersion && version === latestVersion.version
                    ? latestVersion.id
                    : '—'}
                </TableCell>
                <TableCell>
                  {latestVersion && version === latestVersion.version
                    ? latestVersion.schemaType
                    : '—'}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <ConfirmDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        title={deletePermanent ? 'Permanently Delete Subject' : 'Soft-Delete Subject'}
        description={
          deletePermanent
            ? `This will permanently delete "${subject}" and all its versions. This cannot be undone.`
            : `This will soft-delete "${subject}". It can be re-registered later.`
        }
        confirmLabel={deletePermanent ? 'Delete Permanently' : 'Soft Delete'}
        destructive={deletePermanent}
        confirmText={deletePermanent ? subject : undefined}
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
    return schema; // Protobuf or already formatted
  }
}
