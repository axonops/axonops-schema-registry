import { useState } from 'react';
import { useNavigate } from '@tanstack/react-router';
import {
  useKEKs,
  useKEK,
  useCreateKEK,
  useDeleteKEK,
  useUndeleteKEK,
  useTestKEK,
  type KEKResponse,
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
import { toast } from 'sonner';
import {
  Plus,
  Loader2,
  Search,
  Trash2,
  RotateCcw,
  ShieldCheck,
  TestTube,
} from 'lucide-react';

const KMS_TYPES = ['aws-kms', 'azure-kms', 'gcp-kms', 'hcvault'] as const;

function KEKRow({ name }: { name: string }) {
  const navigate = useNavigate();
  const { data: kek, isLoading } = useKEK(name);
  const deleteMutation = useDeleteKEK();
  const undeleteMutation = useUndeleteKEK();
  const testMutation = useTestKEK();

  const handleDelete = () => {
    deleteMutation.mutate(
      { name },
      {
        onSuccess: () => toast.success(`KEK "${name}" deleted`),
        onError: (err) =>
          toast.error(`Failed to delete KEK: ${err.message}`),
      },
    );
  };

  const handleUndelete = () => {
    undeleteMutation.mutate(name, {
      onSuccess: () => toast.success(`KEK "${name}" restored`),
      onError: (err) =>
        toast.error(`Failed to restore KEK: ${err.message}`),
    });
  };

  const handleTest = () => {
    testMutation.mutate(name, {
      onSuccess: () => toast.success(`KMS connectivity test passed for "${name}"`),
      onError: (err) =>
        toast.error(`KMS test failed: ${err.message}`),
    });
  };

  if (isLoading) {
    return (
      <TableRow data-testid={`kek-row-${name}-loading`}>
        <TableCell colSpan={6} className="text-center">
          <Loader2 className="inline-block h-4 w-4 animate-spin" />
        </TableCell>
      </TableRow>
    );
  }

  if (!kek) {
    return null;
  }

  const isDeleted = kek.deleted === true;

  return (
    <TableRow data-testid={`kek-row-${name}`}>
      <TableCell>
        <button
          className="text-sm font-medium text-primary underline-offset-4 hover:underline"
          onClick={() => navigate({ to: `/ui/encryption/${name}` })}
          data-testid={`kek-link-${name}`}
        >
          {kek.name}
        </button>
      </TableCell>
      <TableCell>
        <Badge variant="outline" data-testid={`kek-kms-type-${name}`}>
          {kek.kmsType}
        </Badge>
      </TableCell>
      <TableCell>
        <span className="font-mono text-xs" data-testid={`kek-kms-key-id-${name}`}>
          {kek.kmsKeyId}
        </span>
      </TableCell>
      <TableCell>
        <Badge
          variant={kek.shared ? 'default' : 'secondary'}
          data-testid={`kek-shared-${name}`}
        >
          {kek.shared ? 'Yes' : 'No'}
        </Badge>
      </TableCell>
      <TableCell>
        {isDeleted ? (
          <Badge variant="destructive" data-testid={`kek-status-${name}`}>
            Deleted
          </Badge>
        ) : (
          <Badge variant="default" data-testid={`kek-status-${name}`}>
            Active
          </Badge>
        )}
      </TableCell>
      <TableCell>
        <div className="flex items-center gap-2">
          {!isDeleted && (
            <>
              <Button
                variant="outline"
                size="sm"
                onClick={handleTest}
                disabled={testMutation.isPending}
                data-testid={`kek-test-btn-${name}`}
              >
                {testMutation.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <TestTube className="h-4 w-4" />
                )}
                <span className="ml-1">Test KMS</span>
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={handleDelete}
                disabled={deleteMutation.isPending}
                data-testid={`kek-delete-btn-${name}`}
              >
                {deleteMutation.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Trash2 className="h-4 w-4" />
                )}
                <span className="ml-1">Delete</span>
              </Button>
            </>
          )}
          {isDeleted && (
            <Button
              variant="outline"
              size="sm"
              onClick={handleUndelete}
              disabled={undeleteMutation.isPending}
              data-testid={`kek-undelete-btn-${name}`}
            >
              {undeleteMutation.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <RotateCcw className="h-4 w-4" />
              )}
              <span className="ml-1">Restore</span>
            </Button>
          )}
        </div>
      </TableCell>
    </TableRow>
  );
}

export function KEKsPage() {
  const [search, setSearch] = useState('');
  const [showDeleted, setShowDeleted] = useState(false);
  const [dialogOpen, setDialogOpen] = useState(false);

  // Create form state
  const [formName, setFormName] = useState('');
  const [formKmsType, setFormKmsType] = useState<string>('');
  const [formKmsKeyId, setFormKmsKeyId] = useState('');
  const [formDoc, setFormDoc] = useState('');
  const [formShared, setFormShared] = useState(false);
  const [formKmsProps, setFormKmsProps] = useState<Record<string, string>>({});

  const { data: kekNames = [], isLoading } = useKEKs(
    showDeleted ? { deleted: true } : undefined,
  );
  const createMutation = useCreateKEK();

  const filteredNames = kekNames.filter((name) =>
    name.toLowerCase().includes(search.toLowerCase()),
  );

  const resetForm = () => {
    setFormName('');
    setFormKmsType('');
    setFormKmsKeyId('');
    setFormDoc('');
    setFormShared(false);
    setFormKmsProps({});
  };

  const handleCreate = () => {
    if (!formName.trim()) {
      toast.error('Name is required');
      return;
    }
    if (!formKmsType) {
      toast.error('KMS Type is required');
      return;
    }
    if (!formKmsKeyId.trim()) {
      toast.error('KMS Key ID is required');
      return;
    }

    const request: CreateKEKRequest = {
      name: formName.trim(),
      kmsType: formKmsType,
      kmsKeyId: formKmsKeyId.trim(),
      shared: formShared,
    };

    if (formDoc.trim()) {
      request.doc = formDoc.trim();
    }

    if (Object.keys(formKmsProps).length > 0) {
      request.kmsProps = formKmsProps;
    }

    createMutation.mutate(request, {
      onSuccess: () => {
        toast.success(`KEK "${formName}" created`);
        setDialogOpen(false);
        resetForm();
      },
      onError: (err) => {
        toast.error(`Failed to create KEK: ${err.message}`);
      },
    });
  };

  return (
    <div className="space-y-6">
      <PageBreadcrumbs items={[{ label: 'Encryption Keys' }]} />

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="flex items-center gap-2">
              <ShieldCheck className="h-5 w-5" />
              Key Encryption Keys
            </CardTitle>
            <Button
              onClick={() => setDialogOpen(true)}
              data-testid="keks-create-btn"
            >
              <Plus className="mr-2 h-4 w-4" />
              Create KEK
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-4">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder="Search KEKs..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-10"
                data-testid="keks-search-input"
              />
            </div>
            <div className="flex items-center gap-2">
              <Checkbox
                id="show-deleted"
                checked={showDeleted}
                onCheckedChange={(checked) =>
                  setShowDeleted(checked === true)
                }
                data-testid="keks-show-deleted-toggle"
              />
              <Label
                htmlFor="show-deleted"
                className="cursor-pointer text-sm"
              >
                Show deleted
              </Label>
            </div>
          </div>

          {isLoading ? (
            <div
              className="flex items-center justify-center py-12"
              data-testid="keks-loading"
            >
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          ) : filteredNames.length === 0 ? (
            <div
              className="flex flex-col items-center justify-center py-12 text-center"
              data-testid="keks-empty-state"
            >
              <ShieldCheck className="mb-4 h-12 w-12 text-muted-foreground" />
              <h3 className="text-lg font-medium">No KEKs found</h3>
              <p className="mt-1 text-sm text-muted-foreground">
                {search
                  ? 'No KEKs match your search criteria.'
                  : 'Create your first Key Encryption Key to get started.'}
              </p>
              {!search && (
                <Button
                  className="mt-4"
                  onClick={() => setDialogOpen(true)}
                  data-testid="keks-empty-create-btn"
                >
                  <Plus className="mr-2 h-4 w-4" />
                  Create KEK
                </Button>
              )}
            </div>
          ) : (
            <Table data-testid="keks-table">
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>KMS Type</TableHead>
                  <TableHead>KMS Key ID</TableHead>
                  <TableHead>Shared</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredNames.map((name) => (
                  <KEKRow key={name} name={name} />
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent
          className="max-w-lg"
          data-testid="keks-create-dialog"
        >
          <DialogHeader>
            <DialogTitle>Create Key Encryption Key</DialogTitle>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="kek-name">Name</Label>
              <Input
                id="kek-name"
                placeholder="my-kek"
                value={formName}
                onChange={(e) => setFormName(e.target.value)}
                data-testid="keks-create-name-input"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="kek-kms-type">KMS Type</Label>
              <Select value={formKmsType} onValueChange={setFormKmsType}>
                <SelectTrigger
                  id="kek-kms-type"
                  data-testid="keks-create-kms-type-select"
                >
                  <SelectValue placeholder="Select KMS type" />
                </SelectTrigger>
                <SelectContent>
                  {KMS_TYPES.map((type) => (
                    <SelectItem key={type} value={type}>
                      {type}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="kek-kms-key-id">KMS Key ID</Label>
              <Input
                id="kek-kms-key-id"
                placeholder="arn:aws:kms:us-east-1:123456789:key/abcd-1234"
                value={formKmsKeyId}
                onChange={(e) => setFormKmsKeyId(e.target.value)}
                data-testid="keks-create-kms-key-id-input"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="kek-doc">Documentation (optional)</Label>
              <Textarea
                id="kek-doc"
                placeholder="Description of this KEK..."
                value={formDoc}
                onChange={(e) => setFormDoc(e.target.value)}
                rows={3}
                data-testid="keks-create-doc-input"
              />
            </div>

            <div className="flex items-center gap-2">
              <Checkbox
                id="kek-shared"
                checked={formShared}
                onCheckedChange={(checked) =>
                  setFormShared(checked === true)
                }
                data-testid="keks-create-shared-checkbox"
              />
              <Label htmlFor="kek-shared" className="cursor-pointer">
                Shared across subjects
              </Label>
            </div>

            <div className="space-y-2">
              <Label>KMS Properties (optional)</Label>
              <KeyValueEditor
                value={formKmsProps}
                onChange={setFormKmsProps}
                data-testid="keks-create-kms-props"
              />
            </div>
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setDialogOpen(false);
                resetForm();
              }}
              data-testid="keks-create-cancel-btn"
            >
              Cancel
            </Button>
            <Button
              onClick={handleCreate}
              disabled={createMutation.isPending}
              data-testid="keks-create-submit-btn"
            >
              {createMutation.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              Create KEK
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
