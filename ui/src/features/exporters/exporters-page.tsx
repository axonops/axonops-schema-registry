import { useState } from 'react';
import { useNavigate } from '@tanstack/react-router';
import {
  useExporters,
  useExporter,
  useExporterStatus,
  useCreateExporter,
  useDeleteExporter,
  usePauseExporter,
  useResumeExporter,
  useResetExporter,
  type CreateExporterRequest,
} from '@/api/queries';
import { PageBreadcrumbs } from '@/components/shared/breadcrumbs';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { KeyValueEditor } from '@/components/shared/key-value-editor';
import { Button } from '@/components/ui/button';
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
import { Skeleton } from '@/components/ui/skeleton';
import {
  Search,
  Plus,
  Pause,
  Play,
  RotateCcw,
  Trash2,
  Loader2,
  AlertCircle,
  RefreshCw,
  ArrowRightLeft,
} from 'lucide-react';
import { toast } from 'sonner';

// ── State badge helpers ──

function stateBadgeVariant(state: string): 'default' | 'secondary' | 'destructive' | 'outline' {
  switch (state?.toUpperCase()) {
    case 'RUNNING':
      return 'default';
    case 'PAUSED':
      return 'secondary';
    case 'ERROR':
      return 'destructive';
    default:
      return 'outline';
  }
}

// ── Create form initial state ──

interface CreateFormState {
  name: string;
  contextType: string;
  context: string;
  subjects: string;
  subjectRenameFormat: string;
  config: Record<string, string>;
}

const INITIAL_CREATE_FORM: CreateFormState = {
  name: '',
  contextType: '',
  context: '',
  subjects: '',
  subjectRenameFormat: '',
  config: {},
};

// ── ExporterRow sub-component ──

function ExporterRow({
  name,
  onPause,
  onResume,
  onReset,
  onDelete,
  isPauseLoading,
  isResumeLoading,
  isResetLoading,
}: {
  name: string;
  onPause: (name: string) => void;
  onResume: (name: string) => void;
  onReset: (name: string) => void;
  onDelete: (name: string) => void;
  isPauseLoading: boolean;
  isResumeLoading: boolean;
  isResetLoading: boolean;
}) {
  const navigate = useNavigate();
  const { data: exporter, isLoading: isExporterLoading } = useExporter(name);
  const { data: status, isLoading: isStatusLoading } = useExporterStatus(name);

  const isLoading = isExporterLoading || isStatusLoading;
  const state = status?.state?.toUpperCase() ?? 'UNKNOWN';
  const isPaused = state === 'PAUSED';
  const subjectsCount = exporter?.subjects?.length ?? 0;

  return (
    <TableRow
      className="cursor-pointer"
      onClick={() => navigate({ to: '/ui/exporters/$name', params: { name } })}
      data-testid={`exporter-row-${name}`}
    >
      <TableCell className="font-medium">
        <div className="flex items-center gap-2">
          <ArrowRightLeft className="h-3.5 w-3.5 text-muted-foreground" />
          {name}
        </div>
      </TableCell>
      <TableCell>
        {isLoading ? (
          <Skeleton className="h-4 w-20" />
        ) : (
          exporter?.contextType || '—'
        )}
      </TableCell>
      <TableCell>
        {isLoading ? (
          <Skeleton className="h-5 w-16" />
        ) : (
          <Badge variant={stateBadgeVariant(state)} data-testid={`exporter-state-${name}`}>
            {state}
          </Badge>
        )}
      </TableCell>
      <TableCell>
        {isLoading ? (
          <Skeleton className="h-4 w-8" />
        ) : (
          subjectsCount
        )}
      </TableCell>
      <TableCell className="text-right">
        <div
          className="flex items-center justify-end gap-1"
          onClick={(e) => e.stopPropagation()}
        >
          {isPaused ? (
            <Button
              variant="ghost"
              size="icon"
              onClick={() => onResume(name)}
              disabled={isResumeLoading}
              title="Resume exporter"
              data-testid={`exporter-resume-btn-${name}`}
            >
              {isResumeLoading ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Play className="h-4 w-4" />
              )}
            </Button>
          ) : (
            <Button
              variant="ghost"
              size="icon"
              onClick={() => onPause(name)}
              disabled={isPauseLoading}
              title="Pause exporter"
              data-testid={`exporter-pause-btn-${name}`}
            >
              {isPauseLoading ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Pause className="h-4 w-4" />
              )}
            </Button>
          )}
          <Button
            variant="ghost"
            size="icon"
            onClick={() => onReset(name)}
            disabled={isResetLoading}
            title="Reset exporter"
            data-testid={`exporter-reset-btn-${name}`}
          >
            {isResetLoading ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <RotateCcw className="h-4 w-4" />
            )}
          </Button>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => onDelete(name)}
            title="Delete exporter"
            data-testid={`exporter-delete-btn-${name}`}
          >
            <Trash2 className="h-4 w-4 text-destructive" />
          </Button>
        </div>
      </TableCell>
    </TableRow>
  );
}

// ── Main component ──

export function ExportersPage() {
  const { data: exporterNames, isLoading, isError, error, refetch } = useExporters();
  const createExporter = useCreateExporter();
  const deleteExporter = useDeleteExporter();
  const pauseExporter = usePauseExporter();
  const resumeExporter = useResumeExporter();
  const resetExporter = useResetExporter();

  // Search
  const [search, setSearch] = useState('');

  // Create dialog
  const [createOpen, setCreateOpen] = useState(false);
  const [formState, setFormState] = useState<CreateFormState>(INITIAL_CREATE_FORM);

  // Delete confirmation
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);

  const breadcrumbs = [{ label: 'Exporters' }];

  const filtered = (exporterNames ?? []).filter((name) =>
    name.toLowerCase().includes(search.toLowerCase())
  );

  // ── Action handlers ──

  const handlePause = (name: string) => {
    pauseExporter.mutate(name, {
      onSuccess: () => toast.success(`Exporter "${name}" paused`),
      onError: (err: Error) => toast.error(`Failed to pause exporter: ${err.message}`),
    });
  };

  const handleResume = (name: string) => {
    resumeExporter.mutate(name, {
      onSuccess: () => toast.success(`Exporter "${name}" resumed`),
      onError: (err: Error) => toast.error(`Failed to resume exporter: ${err.message}`),
    });
  };

  const handleReset = (name: string) => {
    resetExporter.mutate(name, {
      onSuccess: () => toast.success(`Exporter "${name}" reset`),
      onError: (err: Error) => toast.error(`Failed to reset exporter: ${err.message}`),
    });
  };

  const handleDeleteConfirm = () => {
    if (!deleteTarget) return;
    deleteExporter.mutate(deleteTarget, {
      onSuccess: () => {
        toast.success(`Exporter "${deleteTarget}" deleted`);
        setDeleteTarget(null);
      },
      onError: (err: Error) => {
        toast.error(`Failed to delete exporter: ${err.message}`);
      },
    });
  };

  // ── Create form handlers ──

  const resetCreateForm = () => {
    setFormState(INITIAL_CREATE_FORM);
  };

  const handleCreateDialogClose = (open: boolean) => {
    if (!open) resetCreateForm();
    setCreateOpen(open);
  };

  const handleCreateSubmit = () => {
    const trimmedName = formState.name.trim();
    if (!trimmedName) return;

    const request: CreateExporterRequest = {
      name: trimmedName,
    };

    if (formState.contextType.trim()) {
      request.contextType = formState.contextType.trim();
    }
    if (formState.context.trim()) {
      request.context = formState.context.trim();
    }
    if (formState.subjects.trim()) {
      request.subjects = formState.subjects
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean);
    }
    if (formState.subjectRenameFormat.trim()) {
      request.subjectRenameFormat = formState.subjectRenameFormat.trim();
    }
    if (Object.keys(formState.config).length > 0) {
      request.config = formState.config;
    }

    createExporter.mutate(request, {
      onSuccess: () => {
        toast.success(`Exporter "${trimmedName}" created successfully`);
        handleCreateDialogClose(false);
      },
      onError: (err: Error) => {
        toast.error(`Failed to create exporter: ${err.message}`);
      },
    });
  };

  const isCreateFormValid = formState.name.trim() !== '';

  // ── Render: Loading ──

  if (isLoading) {
    return (
      <div data-testid="exporters-page-loading">
        <PageBreadcrumbs items={breadcrumbs} />
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      </div>
    );
  }

  // ── Render: Error ──

  if (isError) {
    return (
      <div data-testid="exporters-page-error">
        <PageBreadcrumbs items={breadcrumbs} />
        <div className="flex flex-col items-center justify-center gap-4 py-12">
          <AlertCircle className="h-10 w-10 text-destructive" />
          <p className="text-muted-foreground">
            {error instanceof Error ? error.message : 'Failed to load exporters'}
          </p>
          <Button variant="outline" onClick={() => refetch()}>
            <RefreshCw className="mr-2 h-4 w-4" /> Retry
          </Button>
        </div>
      </div>
    );
  }

  // ── Render: Main ──

  return (
    <div data-testid="exporters-page">
      <PageBreadcrumbs items={breadcrumbs} />

      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">Exporters</h1>
        <Button onClick={() => setCreateOpen(true)} data-testid="exporters-create-btn">
          <Plus className="mr-1.5 h-4 w-4" />
          Create Exporter
        </Button>
      </div>

      {/* Search */}
      <div className="relative mb-4 max-w-sm">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          placeholder="Search exporters..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="pl-9"
          data-testid="exporters-search-input"
        />
      </div>

      <div className="mb-2 text-sm text-muted-foreground">
        {filtered.length} exporter{filtered.length !== 1 ? 's' : ''}
      </div>

      {/* Table */}
      {filtered.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-lg border py-12 text-center text-muted-foreground">
          <ArrowRightLeft className="mb-2 h-8 w-8" />
          <p className="font-medium">
            {search.trim() ? 'No exporters match your search' : 'No exporters found'}
          </p>
          <p className="mt-1 text-sm">
            {search.trim()
              ? 'Try a different search term.'
              : 'Create an exporter to replicate schemas to another registry.'}
          </p>
        </div>
      ) : (
        <div className="rounded-md border">
          <Table data-testid="exporters-list-table">
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Context Type</TableHead>
                <TableHead>State</TableHead>
                <TableHead>Subjects</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filtered.map((name) => (
                <ExporterRow
                  key={name}
                  name={name}
                  onPause={handlePause}
                  onResume={handleResume}
                  onReset={handleReset}
                  onDelete={setDeleteTarget}
                  isPauseLoading={pauseExporter.isPending}
                  isResumeLoading={resumeExporter.isPending}
                  isResetLoading={resetExporter.isPending}
                />
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {/* ── Create Exporter Dialog ── */}
      <Dialog open={createOpen} onOpenChange={handleCreateDialogClose}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Create Exporter</DialogTitle>
          </DialogHeader>

          <div className="space-y-4 py-2">
            {/* Name */}
            <div className="space-y-2">
              <Label htmlFor="exporter-form-name">Name</Label>
              <Input
                id="exporter-form-name"
                value={formState.name}
                onChange={(e) =>
                  setFormState((prev) => ({ ...prev, name: e.target.value }))
                }
                placeholder="e.g., my-exporter"
                data-testid="exporter-form-name-input"
              />
            </div>

            {/* Context Type */}
            <div className="space-y-2">
              <Label htmlFor="exporter-form-context-type">Context Type</Label>
              <Input
                id="exporter-form-context-type"
                value={formState.contextType}
                onChange={(e) =>
                  setFormState((prev) => ({ ...prev, contextType: e.target.value }))
                }
                placeholder="e.g., CUSTOM"
                data-testid="exporter-form-context-type-input"
              />
            </div>

            {/* Context */}
            <div className="space-y-2">
              <Label htmlFor="exporter-form-context">Context</Label>
              <Input
                id="exporter-form-context"
                value={formState.context}
                onChange={(e) =>
                  setFormState((prev) => ({ ...prev, context: e.target.value }))
                }
                placeholder="e.g., my-context"
                data-testid="exporter-form-context-input"
              />
            </div>

            {/* Subjects */}
            <div className="space-y-2">
              <Label htmlFor="exporter-form-subjects">
                Subjects
                <span className="ml-1 text-xs font-normal text-muted-foreground">
                  (comma-separated)
                </span>
              </Label>
              <Input
                id="exporter-form-subjects"
                value={formState.subjects}
                onChange={(e) =>
                  setFormState((prev) => ({ ...prev, subjects: e.target.value }))
                }
                placeholder="e.g., subject1, subject2"
                data-testid="exporter-form-subjects-input"
              />
            </div>

            {/* Subject Rename Format */}
            <div className="space-y-2">
              <Label htmlFor="exporter-form-rename-format">Subject Rename Format</Label>
              <Input
                id="exporter-form-rename-format"
                value={formState.subjectRenameFormat}
                onChange={(e) =>
                  setFormState((prev) => ({ ...prev, subjectRenameFormat: e.target.value }))
                }
                placeholder="e.g., ${subject}-copy"
                data-testid="exporter-form-rename-format-input"
              />
            </div>

            {/* Config */}
            <div className="space-y-2">
              <Label>Config</Label>
              <KeyValueEditor
                value={formState.config}
                onChange={(config) =>
                  setFormState((prev) => ({ ...prev, config }))
                }
                keyPlaceholder="Config key"
                valuePlaceholder="Config value"
              />
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => handleCreateDialogClose(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleCreateSubmit}
              disabled={!isCreateFormValid || createExporter.isPending}
              data-testid="exporter-form-submit-btn"
            >
              {createExporter.isPending ? (
                <>
                  <Loader2 className="mr-1.5 h-4 w-4 animate-spin" />
                  Creating...
                </>
              ) : (
                'Create Exporter'
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ── Delete Confirmation ── */}
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => { if (!open) setDeleteTarget(null); }}
        title="Delete Exporter"
        description={`Are you sure you want to delete the exporter "${deleteTarget ?? ''}"? This action cannot be undone.`}
        confirmLabel="Delete Exporter"
        destructive
        confirmText={deleteTarget ?? undefined}
        onConfirm={handleDeleteConfirm}
        isLoading={deleteExporter.isPending}
      />
    </div>
  );
}
