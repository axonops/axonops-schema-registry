import { useState } from 'react';
import { useNavigate, useParams } from '@tanstack/react-router';
import { useDEK, useDEKVersions, useDeleteDEK } from '@/api/queries';
import { PageBreadcrumbs } from '@/components/shared/breadcrumbs';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Alert, AlertDescription } from '@/components/ui/alert';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { toast } from 'sonner';
import { Loader2, AlertTriangle, ArrowLeft, Trash2 } from 'lucide-react';

export function DEKDetailPage() {
  const params = useParams({ strict: false }) as {
    name: string;
    subject: string;
  };
  const navigate = useNavigate();
  const kekName = params.name;
  const subject = params.subject;

  const { data: dek, isLoading, isError, error } = useDEK(kekName, subject);
  const { data: versions } = useDEKVersions(kekName, subject);
  const deleteDEK = useDeleteDEK();

  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [permanentDelete, setPermanentDelete] = useState(false);
  const [expandKeyMaterial, setExpandKeyMaterial] = useState(false);

  const handleDelete = (permanent: boolean) => {
    setPermanentDelete(permanent);
    setShowDeleteDialog(true);
  };

  const confirmDelete = () => {
    deleteDEK.mutate(
      { kekName, subject, permanent: permanentDelete },
      {
        onSuccess: () => {
          toast.success(
            permanentDelete
              ? 'DEK permanently deleted'
              : 'DEK soft-deleted successfully'
          );
          setShowDeleteDialog(false);
          navigate({ to: '/ui/encryption/$name', params: { name: kekName } });
        },
        onError: (err: Error) => {
          toast.error(`Failed to delete DEK: ${err.message}`);
          setShowDeleteDialog(false);
        },
      }
    );
  };

  const formatTimestamp = (ts?: number) => {
    if (!ts) return 'N/A';
    return new Date(ts).toLocaleString();
  };

  const truncateKeyMaterial = (material: string, maxLength = 80) => {
    if (material.length <= maxLength) return material;
    return material.substring(0, maxLength) + '...';
  };

  if (isLoading) {
    return (
      <div
        className="flex items-center justify-center py-20"
        data-testid="dek-detail-loading"
      >
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (isError || !dek) {
    return (
      <div className="space-y-4" data-testid="dek-detail-error">
        <PageBreadcrumbs
          items={[
            { label: 'Encryption Keys', href: '/ui/encryption' },
            {
              label: kekName,
              href: `/ui/encryption/${encodeURIComponent(kekName)}`,
            },
            { label: subject },
          ]}
        />
        <Alert variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>
            {error?.message || `DEK "${subject}" not found for KEK "${kekName}".`}
          </AlertDescription>
        </Alert>
        <Button
          variant="outline"
          onClick={() =>
            navigate({
              to: '/ui/encryption/$name',
              params: { name: kekName },
            })
          }
          data-testid="dek-detail-back-button-error"
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to KEK
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6" data-testid="dek-detail-page">
      <PageBreadcrumbs
        items={[
          { label: 'Encryption Keys', href: '/ui/encryption' },
          {
            label: kekName,
            href: `/ui/encryption/${encodeURIComponent(kekName)}`,
          },
          { label: subject },
        ]}
      />

      {/* DEK Details Card */}
      <Card data-testid="dek-details-card">
        <CardHeader className="flex flex-row items-center justify-between space-y-0">
          <CardTitle>DEK Details</CardTitle>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              onClick={() =>
                navigate({
                  to: '/ui/encryption/$name',
                  params: { name: kekName },
                })
              }
              data-testid="dek-detail-back-button"
            >
              <ArrowLeft className="mr-2 h-4 w-4" />
              Back to KEK
            </Button>
            <Button
              variant="outline"
              onClick={() => handleDelete(false)}
              disabled={dek.deleted}
              data-testid="dek-soft-delete-button"
            >
              <Trash2 className="mr-2 h-4 w-4" />
              Delete DEK
            </Button>
            <Button
              variant="destructive"
              onClick={() => handleDelete(true)}
              data-testid="dek-permanent-delete-button"
            >
              <Trash2 className="mr-2 h-4 w-4" />
              Permanently Delete
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {dek.deleted && (
            <Alert variant="destructive" data-testid="dek-deleted-alert">
              <AlertTriangle className="h-4 w-4" />
              <AlertDescription>
                This DEK has been soft-deleted. It can be permanently deleted to
                remove it entirely.
              </AlertDescription>
            </Alert>
          )}

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div data-testid="dek-detail-subject">
              <p className="text-sm font-medium text-muted-foreground">
                Subject
              </p>
              <p className="text-sm">{dek.subject}</p>
            </div>
            <div data-testid="dek-detail-kek-name">
              <p className="text-sm font-medium text-muted-foreground">
                KEK Name
              </p>
              <Button
                variant="link"
                className="h-auto p-0 text-sm"
                onClick={() =>
                  navigate({
                    to: '/ui/encryption/$name',
                    params: { name: dek.kekName },
                  })
                }
                data-testid="dek-detail-kek-link"
              >
                {dek.kekName}
              </Button>
            </div>
            <div data-testid="dek-detail-version">
              <p className="text-sm font-medium text-muted-foreground">
                Version
              </p>
              <p className="text-sm">{dek.version}</p>
            </div>
            <div data-testid="dek-detail-algorithm">
              <p className="text-sm font-medium text-muted-foreground">
                Algorithm
              </p>
              <Badge variant="secondary">{dek.algorithm}</Badge>
            </div>
            <div data-testid="dek-detail-created">
              <p className="text-sm font-medium text-muted-foreground">
                Created
              </p>
              <p className="text-sm">{formatTimestamp(dek.ts)}</p>
            </div>
            <div data-testid="dek-detail-status">
              <p className="text-sm font-medium text-muted-foreground">
                Status
              </p>
              <Badge variant={dek.deleted ? 'destructive' : 'default'}>
                {dek.deleted ? 'Deleted' : 'Active'}
              </Badge>
            </div>
          </div>

          {dek.encryptedKeyMaterial && (
            <div data-testid="dek-detail-encrypted-key-material">
              <p className="mb-2 text-sm font-medium text-muted-foreground">
                Encrypted Key Material
              </p>
              <div className="rounded-md border bg-muted p-3">
                <code className="block break-all font-mono text-xs">
                  {expandKeyMaterial
                    ? dek.encryptedKeyMaterial
                    : truncateKeyMaterial(dek.encryptedKeyMaterial)}
                </code>
                {dek.encryptedKeyMaterial.length > 80 && (
                  <Button
                    variant="ghost"
                    size="sm"
                    className="mt-2 h-auto p-0 text-xs"
                    onClick={() => setExpandKeyMaterial(!expandKeyMaterial)}
                    data-testid="dek-detail-expand-key-toggle"
                  >
                    {expandKeyMaterial ? 'Show less' : 'Show more'}
                  </Button>
                )}
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* DEK Versions Card */}
      <Card data-testid="dek-versions-card">
        <CardHeader>
          <CardTitle>Versions</CardTitle>
        </CardHeader>
        <CardContent>
          {!versions || versions.length === 0 ? (
            <p
              className="text-sm text-muted-foreground"
              data-testid="dek-versions-empty"
            >
              No versions available for this DEK.
            </p>
          ) : (
            <Table data-testid="dek-versions-table">
              <TableHeader>
                <TableRow>
                  <TableHead>Version</TableHead>
                  <TableHead>Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {versions.map((version) => (
                  <TableRow
                    key={version}
                    className={
                      version === dek.version ? 'bg-muted/50' : undefined
                    }
                    data-testid={`dek-version-row-${version}`}
                  >
                    <TableCell className="font-medium">
                      {version}
                    </TableCell>
                    <TableCell>
                      {version === dek.version ? (
                        <Badge
                          variant="default"
                          data-testid={`dek-version-current-badge-${version}`}
                        >
                          Current
                        </Badge>
                      ) : (
                        <span className="text-sm text-muted-foreground">
                          —
                        </span>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Delete Confirmation Dialog */}
      <Dialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <DialogContent data-testid="dek-delete-dialog">
          <DialogHeader>
            <DialogTitle>
              {permanentDelete ? 'Permanently Delete DEK' : 'Delete DEK'}
            </DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            {permanentDelete
              ? `Are you sure you want to permanently delete DEK "${subject}" from KEK "${kekName}"? This action cannot be undone.`
              : `Are you sure you want to soft-delete DEK "${subject}" from KEK "${kekName}"? The DEK can be permanently deleted later.`}
          </p>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowDeleteDialog(false)}
              data-testid="dek-delete-cancel-button"
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={confirmDelete}
              disabled={deleteDEK.isPending}
              data-testid="dek-delete-confirm-button"
            >
              {deleteDEK.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              {permanentDelete ? 'Permanently Delete' : 'Delete'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
