import { useState } from 'react';
import { useNavigate, useParams } from '@tanstack/react-router';
import {
  useKEK,
  useUpdateKEK,
  useDeleteKEK,
  useUndeleteKEK,
  useTestKEK,
  useDEKs,
  useCreateDEK,
  type CreateKEKRequest,
} from '@/api/queries';
import { PageBreadcrumbs } from '@/components/shared/breadcrumbs';
import { KeyValueEditor } from '@/components/shared/key-value-editor';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Checkbox } from '@/components/ui/checkbox';
import { Textarea } from '@/components/ui/textarea';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { toast } from 'sonner';
import {
  Loader2,
  AlertTriangle,
  ArrowLeft,
  TestTube,
  Trash2,
  RotateCcw,
  Pencil,
  Plus,
} from 'lucide-react';

export function KEKDetailPage() {
  const params = useParams({ strict: false }) as { name: string };
  const name = decodeURIComponent(params.name);
  const navigate = useNavigate();

  const { data: kek, isLoading, error } = useKEK(name);
  const { data: dekSubjects } = useDEKs(name);

  const updateKEK = useUpdateKEK();
  const deleteKEK = useDeleteKEK();
  const undeleteKEK = useUndeleteKEK();
  const testKEK = useTestKEK();
  const createDEK = useCreateDEK();

  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [createDEKDialogOpen, setCreateDEKDialogOpen] = useState(false);

  // Edit KEK form state
  const [editKmsType, setEditKmsType] = useState('');
  const [editKmsKeyId, setEditKmsKeyId] = useState('');
  const [editDoc, setEditDoc] = useState('');
  const [editShared, setEditShared] = useState(false);
  const [editKmsProps, setEditKmsProps] = useState<Record<string, string>>({});

  // Create DEK form state
  const [dekSubject, setDekSubject] = useState('');
  const [dekAlgorithm, setDekAlgorithm] = useState('AES256_GCM');

  function openEditDialog() {
    if (kek) {
      setEditKmsType(kek.kmsType);
      setEditKmsKeyId(kek.kmsKeyId);
      setEditDoc(kek.doc ?? '');
      setEditShared(kek.shared);
      setEditKmsProps(kek.kmsProps ? { ...kek.kmsProps } : {});
    }
    setEditDialogOpen(true);
  }

  function handleEditSubmit() {
    const payload: Partial<CreateKEKRequest> & { name: string } = {
      name,
      kmsType: editKmsType,
      kmsKeyId: editKmsKeyId,
      doc: editDoc || undefined,
      shared: editShared,
      kmsProps:
        Object.keys(editKmsProps).length > 0 ? editKmsProps : undefined,
    };
    updateKEK.mutate(payload, {
      onSuccess: () => {
        toast.success('KEK updated successfully');
        setEditDialogOpen(false);
      },
      onError: (err: Error) => {
        toast.error(`Failed to update KEK: ${err.message}`);
      },
    });
  }

  function handleTestKMS() {
    testKEK.mutate(name, {
      onSuccess: () => {
        toast.success('KMS connectivity test passed');
      },
      onError: (err: Error) => {
        toast.error(`KMS test failed: ${err.message}`);
      },
    });
  }

  function handleDelete() {
    deleteKEK.mutate(
      { name },
      {
        onSuccess: () => {
          toast.success('KEK deleted');
          navigate({ to: '/ui/encryption' });
        },
        onError: (err: Error) => {
          toast.error(`Failed to delete KEK: ${err.message}`);
        },
      },
    );
  }

  function handleUndelete() {
    undeleteKEK.mutate(name, {
      onSuccess: () => {
        toast.success('KEK restored');
      },
      onError: (err: Error) => {
        toast.error(`Failed to restore KEK: ${err.message}`);
      },
    });
  }

  function handleCreateDEK() {
    createDEK.mutate(
      {
        kekName: name,
        subject: dekSubject,
        algorithm: dekAlgorithm,
      },
      {
        onSuccess: () => {
          toast.success('DEK created successfully');
          setCreateDEKDialogOpen(false);
          setDekSubject('');
          setDekAlgorithm('AES256_GCM');
        },
        onError: (err: Error) => {
          toast.error(`Failed to create DEK: ${err.message}`);
        },
      },
    );
  }

  if (isLoading) {
    return (
      <div
        className="flex items-center justify-center py-12"
        data-testid="kek-detail-loading"
      >
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-4" data-testid="kek-detail-error">
        <PageBreadcrumbs
          items={[
            { label: 'Encryption Keys', href: '/ui/encryption' },
            { label: name },
          ]}
        />
        <Alert variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>
            Failed to load KEK: {error.message}
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  if (!kek) {
    return null;
  }

  return (
    <div className="space-y-6" data-testid="kek-detail-page">
      <PageBreadcrumbs
        items={[
          { label: 'Encryption Keys', href: '/ui/encryption' },
          { label: name },
        ]}
      />

      {/* KEK Details Card */}
      <Card data-testid="kek-details-card">
        <CardHeader>
          <CardTitle>KEK Details</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {kek.deleted && (
            <Alert variant="destructive" data-testid="kek-deleted-alert">
              <AlertTriangle className="h-4 w-4" />
              <AlertDescription>
                This KEK has been soft-deleted. You can restore it using the
                Undelete action.
              </AlertDescription>
            </Alert>
          )}

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <Label className="text-muted-foreground text-sm">Name</Label>
              <p className="font-medium" data-testid="kek-name">
                {kek.name}
              </p>
            </div>
            <div>
              <Label className="text-muted-foreground text-sm">KMS Type</Label>
              <div>
                <Badge variant="secondary" data-testid="kek-kms-type">
                  {kek.kmsType}
                </Badge>
              </div>
            </div>
            <div>
              <Label className="text-muted-foreground text-sm">
                KMS Key ID
              </Label>
              <p className="font-mono text-sm" data-testid="kek-kms-key-id">
                {kek.kmsKeyId}
              </p>
            </div>
            <div>
              <Label className="text-muted-foreground text-sm">Shared</Label>
              <div>
                <Badge
                  variant={kek.shared ? 'default' : 'outline'}
                  data-testid="kek-shared"
                >
                  {kek.shared ? 'Yes' : 'No'}
                </Badge>
              </div>
            </div>
            {kek.doc && (
              <div className="sm:col-span-2">
                <Label className="text-muted-foreground text-sm">
                  Documentation
                </Label>
                <p className="text-sm" data-testid="kek-doc">
                  {kek.doc}
                </p>
              </div>
            )}
            {kek.ts != null && (
              <div>
                <Label className="text-muted-foreground text-sm">Created</Label>
                <p className="text-sm" data-testid="kek-created">
                  {new Date(kek.ts).toLocaleString()}
                </p>
              </div>
            )}
          </div>

          <div className="flex flex-wrap gap-2 pt-4">
            <Button
              variant="outline"
              size="sm"
              onClick={openEditDialog}
              data-testid="kek-edit-button"
            >
              <Pencil className="mr-2 h-4 w-4" />
              Edit KEK
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={handleTestKMS}
              disabled={testKEK.isPending}
              data-testid="kek-test-button"
            >
              {testKEK.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <TestTube className="mr-2 h-4 w-4" />
              )}
              Test KMS
            </Button>
            {kek.deleted ? (
              <Button
                variant="outline"
                size="sm"
                onClick={handleUndelete}
                disabled={undeleteKEK.isPending}
                data-testid="kek-undelete-button"
              >
                {undeleteKEK.isPending ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <RotateCcw className="mr-2 h-4 w-4" />
                )}
                Undelete
              </Button>
            ) : (
              <Button
                variant="destructive"
                size="sm"
                onClick={handleDelete}
                disabled={deleteKEK.isPending}
                data-testid="kek-delete-button"
              >
                {deleteKEK.isPending ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <Trash2 className="mr-2 h-4 w-4" />
                )}
                Delete
              </Button>
            )}
            <Button
              variant="ghost"
              size="sm"
              onClick={() => navigate({ to: '/ui/encryption' })}
              data-testid="kek-back-button"
            >
              <ArrowLeft className="mr-2 h-4 w-4" />
              Back to List
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* KMS Properties Card */}
      {kek.kmsProps && Object.keys(kek.kmsProps).length > 0 && (
        <Card data-testid="kek-kms-props-card">
          <CardHeader>
            <CardTitle>KMS Properties</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Key</TableHead>
                  <TableHead>Value</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {Object.entries(kek.kmsProps).map(([key, value]) => (
                  <TableRow key={key}>
                    <TableCell className="font-mono text-sm">{key}</TableCell>
                    <TableCell className="font-mono text-sm">{value}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {/* DEK Subjects Card */}
      <Card data-testid="kek-dek-subjects-card">
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>DEK Subjects</CardTitle>
          <Button
            size="sm"
            onClick={() => setCreateDEKDialogOpen(true)}
            data-testid="create-dek-button"
          >
            <Plus className="mr-2 h-4 w-4" />
            Create DEK
          </Button>
        </CardHeader>
        <CardContent>
          {dekSubjects && dekSubjects.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Subject</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {dekSubjects.map((subject: string) => (
                  <TableRow key={subject}>
                    <TableCell>
                      <button
                        className="text-primary hover:underline"
                        onClick={() =>
                          navigate({
                            to: `/ui/encryption/${encodeURIComponent(name)}/deks/${encodeURIComponent(subject)}`,
                          })
                        }
                        data-testid={`dek-subject-link-${subject}`}
                      >
                        {subject}
                      </button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <p
              className="text-muted-foreground py-4 text-center text-sm"
              data-testid="no-deks-message"
            >
              No DEKs registered for this KEK
            </p>
          )}
        </CardContent>
      </Card>

      {/* Edit KEK Dialog */}
      <Dialog open={editDialogOpen} onOpenChange={setEditDialogOpen}>
        <DialogContent data-testid="edit-kek-dialog">
          <DialogHeader>
            <DialogTitle>Edit KEK</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="edit-kms-type">KMS Type</Label>
              <Select value={editKmsType} onValueChange={setEditKmsType}>
                <SelectTrigger
                  id="edit-kms-type"
                  data-testid="edit-kek-kms-type"
                >
                  <SelectValue placeholder="Select KMS type" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="aws-kms">aws-kms</SelectItem>
                  <SelectItem value="azure-kms">azure-kms</SelectItem>
                  <SelectItem value="gcp-kms">gcp-kms</SelectItem>
                  <SelectItem value="hcvault">hcvault</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-kms-key-id">KMS Key ID</Label>
              <Input
                id="edit-kms-key-id"
                value={editKmsKeyId}
                onChange={(e) => setEditKmsKeyId(e.target.value)}
                data-testid="edit-kek-kms-key-id"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-doc">Documentation</Label>
              <Textarea
                id="edit-doc"
                value={editDoc}
                onChange={(e) => setEditDoc(e.target.value)}
                data-testid="edit-kek-doc"
              />
            </div>
            <div className="flex items-center space-x-2">
              <Checkbox
                id="edit-shared"
                checked={editShared}
                onCheckedChange={(checked) => setEditShared(checked === true)}
                data-testid="edit-kek-shared"
              />
              <Label htmlFor="edit-shared">Shared</Label>
            </div>
            <div className="space-y-2">
              <Label>KMS Properties</Label>
              <KeyValueEditor
                value={editKmsProps}
                onChange={setEditKmsProps}
                data-testid="edit-kek-kms-props"
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setEditDialogOpen(false)}
              data-testid="edit-kek-cancel-button"
            >
              Cancel
            </Button>
            <Button
              onClick={handleEditSubmit}
              disabled={updateKEK.isPending}
              data-testid="edit-kek-submit-button"
            >
              {updateKEK.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              Save Changes
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Create DEK Dialog */}
      <Dialog open={createDEKDialogOpen} onOpenChange={setCreateDEKDialogOpen}>
        <DialogContent data-testid="create-dek-dialog">
          <DialogHeader>
            <DialogTitle>Create DEK</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="dek-subject">Subject</Label>
              <Input
                id="dek-subject"
                value={dekSubject}
                onChange={(e) => setDekSubject(e.target.value)}
                placeholder="Enter subject name"
                data-testid="create-dek-subject"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="dek-algorithm">Algorithm</Label>
              <Select value={dekAlgorithm} onValueChange={setDekAlgorithm}>
                <SelectTrigger
                  id="dek-algorithm"
                  data-testid="create-dek-algorithm"
                >
                  <SelectValue placeholder="Select algorithm" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="AES256_GCM">AES256_GCM</SelectItem>
                  <SelectItem value="AES256_SIV">AES256_SIV</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setCreateDEKDialogOpen(false)}
              data-testid="create-dek-cancel-button"
            >
              Cancel
            </Button>
            <Button
              onClick={handleCreateDEK}
              disabled={createDEK.isPending || !dekSubject.trim()}
              data-testid="create-dek-submit-button"
            >
              {createDEK.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              Create DEK
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
