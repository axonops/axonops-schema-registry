import { useState } from 'react';
import { useParams, useNavigate } from '@tanstack/react-router';
import {
  useExporter,
  useExporterStatus,
  useExporterConfig,
  useDeleteExporter,
  usePauseExporter,
  useResumeExporter,
  useResetExporter,
  useUpdateExporterConfig,
} from '@/api/queries';
import { PageBreadcrumbs } from '@/components/shared/breadcrumbs';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { KeyValueEditor } from '@/components/shared/key-value-editor';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { Alert, AlertDescription } from '@/components/ui/alert';
import {
  Pause,
  Play,
  RotateCcw,
  Trash2,
  Loader2,
  AlertCircle,
  RefreshCw,
  Save,
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

function formatTimestamp(ts?: number): string {
  if (!ts) return '—';
  return new Date(ts).toLocaleString();
}

// ── Component ──

export function ExporterDetailPage() {
  const { name } = useParams({ strict: false }) as { name: string };
  const navigate = useNavigate();

  // Data fetching
  const {
    data: exporter,
    isLoading: isExporterLoading,
    isError: isExporterError,
    error: exporterError,
    refetch: refetchExporter,
  } = useExporter(name);
  const {
    data: status,
    isLoading: isStatusLoading,
    refetch: refetchStatus,
  } = useExporterStatus(name);
  const {
    data: config,
    isLoading: isConfigLoading,
    refetch: refetchConfig,
  } = useExporterConfig(name);

  // Mutations
  const deleteExporter = useDeleteExporter();
  const pauseExporter = usePauseExporter();
  const resumeExporter = useResumeExporter();
  const resetExporter = useResetExporter();
  const updateConfig = useUpdateExporterConfig();

  // Local state for editable config
  const [editableConfig, setEditableConfig] = useState<Record<string, string> | null>(null);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);

  // Derive the config being displayed/edited
  const displayConfig = editableConfig ?? config ?? {};
  const hasConfigChanges = editableConfig !== null;

  const breadcrumbs = [
    { label: 'Exporters', href: '/ui/exporters' },
    { label: name },
  ];

  const state = status?.state?.toUpperCase() ?? 'UNKNOWN';
  const isPaused = state === 'PAUSED';

  // ── Action handlers ──

  const handlePause = () => {
    pauseExporter.mutate(name, {
      onSuccess: () => {
        toast.success(`Exporter "${name}" paused`);
        refetchStatus();
      },
      onError: (err: Error) => toast.error(`Failed to pause exporter: ${err.message}`),
    });
  };

  const handleResume = () => {
    resumeExporter.mutate(name, {
      onSuccess: () => {
        toast.success(`Exporter "${name}" resumed`);
        refetchStatus();
      },
      onError: (err: Error) => toast.error(`Failed to resume exporter: ${err.message}`),
    });
  };

  const handleReset = () => {
    resetExporter.mutate(name, {
      onSuccess: () => {
        toast.success(`Exporter "${name}" reset`);
        refetchStatus();
      },
      onError: (err: Error) => toast.error(`Failed to reset exporter: ${err.message}`),
    });
  };

  const handleDeleteConfirm = () => {
    deleteExporter.mutate(name, {
      onSuccess: () => {
        toast.success(`Exporter "${name}" deleted`);
        setShowDeleteDialog(false);
        navigate({ to: '/ui/exporters' });
      },
      onError: (err: Error) => {
        toast.error(`Failed to delete exporter: ${err.message}`);
      },
    });
  };

  const handleSaveConfig = () => {
    if (!editableConfig) return;
    updateConfig.mutate(
      { name, config: editableConfig },
      {
        onSuccess: () => {
          toast.success('Exporter configuration updated');
          setEditableConfig(null);
          refetchConfig();
        },
        onError: (err: Error) => {
          toast.error(`Failed to update configuration: ${err.message}`);
        },
      }
    );
  };

  const handleConfigChange = (newConfig: Record<string, string>) => {
    setEditableConfig(newConfig);
  };

  // ── Render: Loading ──

  if (isExporterLoading) {
    return (
      <div data-testid="exporter-detail-loading">
        <PageBreadcrumbs items={breadcrumbs} />
        <div className="space-y-4">
          <Skeleton className="h-8 w-64" />
          <Skeleton className="h-48 w-full" />
          <Skeleton className="h-32 w-full" />
        </div>
      </div>
    );
  }

  // ── Render: Error ──

  if (isExporterError) {
    return (
      <div data-testid="exporter-detail-error">
        <PageBreadcrumbs items={breadcrumbs} />
        <div className="flex flex-col items-center justify-center gap-4 py-12">
          <AlertCircle className="h-10 w-10 text-destructive" />
          <p className="text-muted-foreground">
            {exporterError instanceof Error ? exporterError.message : 'Failed to load exporter'}
          </p>
          <Button variant="outline" onClick={() => refetchExporter()}>
            <RefreshCw className="mr-2 h-4 w-4" /> Retry
          </Button>
        </div>
      </div>
    );
  }

  if (!exporter) return null;

  // ── Render: Main ──

  return (
    <div data-testid="exporter-detail-page">
      <PageBreadcrumbs items={breadcrumbs} />

      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold" data-testid="exporter-detail-name">
            {name}
          </h1>
          <div className="mt-1 flex items-center gap-2">
            {isStatusLoading ? (
              <Skeleton className="h-5 w-16" />
            ) : (
              <Badge
                variant={stateBadgeVariant(state)}
                data-testid="exporter-detail-state"
              >
                {state}
              </Badge>
            )}
            {exporter.contextType && (
              <Badge variant="outline" data-testid="exporter-detail-context-type">
                {exporter.contextType}
              </Badge>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          {isPaused ? (
            <Button
              variant="outline"
              size="sm"
              onClick={handleResume}
              disabled={resumeExporter.isPending}
              data-testid="exporter-detail-resume-btn"
            >
              {resumeExporter.isPending ? (
                <Loader2 className="mr-1 h-4 w-4 animate-spin" />
              ) : (
                <Play className="mr-1 h-4 w-4" />
              )}
              Resume
            </Button>
          ) : (
            <Button
              variant="outline"
              size="sm"
              onClick={handlePause}
              disabled={pauseExporter.isPending}
              data-testid="exporter-detail-pause-btn"
            >
              {pauseExporter.isPending ? (
                <Loader2 className="mr-1 h-4 w-4 animate-spin" />
              ) : (
                <Pause className="mr-1 h-4 w-4" />
              )}
              Pause
            </Button>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={handleReset}
            disabled={resetExporter.isPending}
            data-testid="exporter-detail-reset-btn"
          >
            {resetExporter.isPending ? (
              <Loader2 className="mr-1 h-4 w-4 animate-spin" />
            ) : (
              <RotateCcw className="mr-1 h-4 w-4" />
            )}
            Reset
          </Button>
          <Button
            variant="destructive"
            size="sm"
            onClick={() => setShowDeleteDialog(true)}
            data-testid="exporter-detail-delete-btn"
          >
            <Trash2 className="mr-1 h-4 w-4" />
            Delete
          </Button>
        </div>
      </div>

      {/* Error trace alert */}
      {status?.trace && (
        <Alert variant="destructive" className="mb-6" data-testid="exporter-detail-error-trace">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>
            <span className="font-medium">Error trace:</span> {status.trace}
          </AlertDescription>
        </Alert>
      )}

      <div className="grid gap-6 md:grid-cols-2">
        {/* Exporter Details Card */}
        <Card data-testid="exporter-detail-info-card">
          <CardHeader>
            <CardTitle className="text-sm font-medium">Exporter Details</CardTitle>
          </CardHeader>
          <CardContent>
            <dl className="space-y-3">
              <div>
                <dt className="text-sm text-muted-foreground">Name</dt>
                <dd className="text-sm font-medium">{exporter.name}</dd>
              </div>
              <div>
                <dt className="text-sm text-muted-foreground">Context Type</dt>
                <dd className="text-sm font-medium">{exporter.contextType || '—'}</dd>
              </div>
              <div>
                <dt className="text-sm text-muted-foreground">Context</dt>
                <dd className="text-sm font-medium">{exporter.context || '—'}</dd>
              </div>
              <div>
                <dt className="text-sm text-muted-foreground">Subject Rename Format</dt>
                <dd className="text-sm font-medium">
                  {exporter.subjectRenameFormat ? (
                    <code className="rounded bg-muted px-1.5 py-0.5 text-xs">
                      {exporter.subjectRenameFormat}
                    </code>
                  ) : (
                    '—'
                  )}
                </dd>
              </div>
            </dl>
          </CardContent>
        </Card>

        {/* Status Card */}
        <Card data-testid="exporter-detail-status-card">
          <CardHeader>
            <CardTitle className="text-sm font-medium">Status</CardTitle>
          </CardHeader>
          <CardContent>
            {isStatusLoading ? (
              <div className="space-y-3">
                <Skeleton className="h-4 w-32" />
                <Skeleton className="h-4 w-24" />
                <Skeleton className="h-4 w-40" />
              </div>
            ) : (
              <dl className="space-y-3">
                <div>
                  <dt className="text-sm text-muted-foreground">State</dt>
                  <dd>
                    <Badge
                      variant={stateBadgeVariant(state)}
                      data-testid="exporter-status-state"
                    >
                      {state}
                    </Badge>
                  </dd>
                </div>
                <div>
                  <dt className="text-sm text-muted-foreground">Offset</dt>
                  <dd className="text-sm font-medium">
                    {status?.offset !== undefined ? status.offset : '—'}
                  </dd>
                </div>
                <div>
                  <dt className="text-sm text-muted-foreground">Timestamp</dt>
                  <dd className="text-sm font-medium">{formatTimestamp(status?.ts)}</dd>
                </div>
              </dl>
            )}
          </CardContent>
        </Card>

        {/* Subjects Card */}
        <Card data-testid="exporter-detail-subjects-card">
          <CardHeader>
            <CardTitle className="text-sm font-medium">
              Subjects
              {exporter.subjects && (
                <span className="ml-2 text-xs font-normal text-muted-foreground">
                  ({exporter.subjects.length})
                </span>
              )}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {!exporter.subjects || exporter.subjects.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                No subjects configured. All subjects will be exported.
              </p>
            ) : (
              <ul className="space-y-1" data-testid="exporter-subjects-list">
                {exporter.subjects.map((subject) => (
                  <li key={subject} className="text-sm">
                    <code className="rounded bg-muted px-1.5 py-0.5 text-xs">
                      {subject}
                    </code>
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>

        {/* Configuration Card */}
        <Card data-testid="exporter-detail-config-card">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-medium">Configuration</CardTitle>
              {hasConfigChanges && (
                <Button
                  size="sm"
                  onClick={handleSaveConfig}
                  disabled={updateConfig.isPending}
                  data-testid="exporter-config-save-btn"
                >
                  {updateConfig.isPending ? (
                    <Loader2 className="mr-1 h-4 w-4 animate-spin" />
                  ) : (
                    <Save className="mr-1 h-4 w-4" />
                  )}
                  Save Config
                </Button>
              )}
            </div>
          </CardHeader>
          <CardContent>
            {isConfigLoading ? (
              <div className="space-y-2">
                <Skeleton className="h-8 w-full" />
                <Skeleton className="h-8 w-full" />
              </div>
            ) : (
              <KeyValueEditor
                value={displayConfig}
                onChange={handleConfigChange}
                keyPlaceholder="Config key"
                valuePlaceholder="Config value"
              />
            )}
          </CardContent>
        </Card>
      </div>

      {/* ── Delete Confirmation ── */}
      <ConfirmDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        title="Delete Exporter"
        description={`Are you sure you want to delete the exporter "${name}"? This action cannot be undone.`}
        confirmLabel="Delete Exporter"
        destructive
        confirmText={name}
        onConfirm={handleDeleteConfirm}
        isLoading={deleteExporter.isPending}
      />
    </div>
  );
}
