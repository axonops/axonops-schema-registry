import { useState } from 'react';
import { useNavigate, useParams } from '@tanstack/react-router';
import {
  useSubjectVersions,
  useSubjectVersion,
  useSubjectConfig,
  useSubjectMode,
  useGlobalConfig,
  useDeleteSubject,
  useSubjectFullConfig,
  useSetSubjectConfig,
  useSetSubjectMode,
  useDeleteSubjectConfig,
  useDeleteSubjectMode,
} from '@/api/queries';
import { PageBreadcrumbs } from '@/components/shared/breadcrumbs';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { MetadataPanel } from '@/components/shared/metadata-panel';
import { RuleSetPanel } from '@/components/shared/rule-set-panel';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { AlertCircle, RefreshCw, Plus, Trash2, RotateCcw, Check, X } from 'lucide-react';
import { toast } from 'sonner';

const COMPATIBILITY_LEVELS = [
  'NONE',
  'BACKWARD',
  'BACKWARD_TRANSITIVE',
  'FORWARD',
  'FORWARD_TRANSITIVE',
  'FULL',
  'FULL_TRANSITIVE',
];

const MODE_VALUES = ['READWRITE', 'READONLY', 'IMPORT'];

export function SubjectDetailPage() {
  const { subject } = useParams({ strict: false }) as { subject: string };
  const { data: versions, isLoading, isError, error, refetch } = useSubjectVersions(subject);
  const { data: latestVersion } = useSubjectVersion(subject, 'latest');
  const { data: subjectConfig } = useSubjectConfig(subject);
  const { data: subjectMode } = useSubjectMode(subject);
  const { data: globalConfig } = useGlobalConfig();
  const { data: fullConfig } = useSubjectFullConfig(subject);
  const navigate = useNavigate();
  const deleteMutation = useDeleteSubject(subject);
  const setConfigMutation = useSetSubjectConfig();
  const setModeMutation = useSetSubjectMode();
  const deleteConfigMutation = useDeleteSubjectConfig();
  const deleteModeMutation = useDeleteSubjectMode();

  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [deletePermanent, setDeletePermanent] = useState(false);

  const breadcrumbs = [
    { label: 'Subjects', href: '/ui/subjects' },
    { label: subject },
  ];

  const compatLevel = subjectConfig?.compatibilityLevel ?? globalConfig?.compatibilityLevel ?? 'BACKWARD';
  const modeValue = subjectMode?.mode ?? 'READWRITE';
  const hasSubjectConfig = subjectConfig?.compatibilityLevel != null;
  const hasSubjectMode = subjectMode?.mode != null;

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

  const handleCompatChange = (value: string) => {
    setConfigMutation.mutate(
      { subject, compatibility: value },
      {
        onSuccess: () => toast.success(`Compatibility set to ${value}`),
        onError: (err) => toast.error(err instanceof Error ? err.message : 'Failed to update compatibility'),
      }
    );
  };

  const handleModeChange = (value: string) => {
    setModeMutation.mutate(
      { subject, mode: value },
      {
        onSuccess: () => toast.success(`Mode set to ${value}`),
        onError: (err) => toast.error(err instanceof Error ? err.message : 'Failed to update mode'),
      }
    );
  };

  const handleResetConfig = () => {
    deleteConfigMutation.mutate(subject, {
      onSuccess: () => toast.success('Compatibility reset to global default'),
      onError: (err) => toast.error(err instanceof Error ? err.message : 'Failed to reset compatibility'),
    });
  };

  const handleResetMode = () => {
    deleteModeMutation.mutate(subject, {
      onSuccess: () => toast.success('Mode reset to global default'),
      onError: (err) => toast.error(err instanceof Error ? err.message : 'Failed to reset mode'),
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

      <Tabs defaultValue="overview" className="mt-6">
        <TabsList>
          <TabsTrigger value="overview" data-testid="subject-tab-overview">
            Overview
          </TabsTrigger>
          <TabsTrigger value="config" data-testid="subject-tab-config">
            Configuration
          </TabsTrigger>
          <TabsTrigger value="metadata" data-testid="subject-tab-metadata">
            Metadata
          </TabsTrigger>
          <TabsTrigger value="rules" data-testid="subject-tab-rules">
            Rules
          </TabsTrigger>
        </TabsList>

        {/* ── Overview Tab ── */}
        <TabsContent value="overview" data-testid="subject-tab-content-overview">
          {latestVersion && (
            <div className="mb-6 rounded-md border p-4" data-testid="subject-detail-latest-preview">
              <h3 className="mb-2 text-sm font-medium text-muted-foreground">
                Latest Schema (v{latestVersion.version})
              </h3>
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
                {versions?.sort((a, b) => b - a).map((v) => (
                  <VersionRow
                    key={v}
                    subject={subject}
                    version={v}
                    isLatest={latestVersion?.version === v}
                  />
                ))}
              </TableBody>
            </Table>
          </div>
        </TabsContent>

        {/* ── Configuration Tab ── */}
        <TabsContent value="config" data-testid="subject-tab-content-config">
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">Subject Configuration</CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              {/* Compatibility Level */}
              <div className="flex items-center justify-between" data-testid="config-compatibility">
                <div className="space-y-0.5">
                  <label className="text-sm font-medium">Compatibility Level</label>
                  <p className="text-xs text-muted-foreground">
                    {hasSubjectConfig ? 'Subject-level override' : 'Inherited from global config'}
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <Select
                    value={compatLevel}
                    onValueChange={handleCompatChange}
                    disabled={setConfigMutation.isPending}
                  >
                    <SelectTrigger data-testid="config-compatibility-select">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {COMPATIBILITY_LEVELS.map((level) => (
                        <SelectItem key={level} value={level}>
                          {level}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {hasSubjectConfig && (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={handleResetConfig}
                      disabled={deleteConfigMutation.isPending}
                      data-testid="config-compatibility-reset"
                      title="Reset to global default"
                    >
                      <RotateCcw className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              </div>

              {/* Mode */}
              <div className="flex items-center justify-between" data-testid="config-mode">
                <div className="space-y-0.5">
                  <label className="text-sm font-medium">Mode</label>
                  <p className="text-xs text-muted-foreground">
                    {hasSubjectMode ? 'Subject-level override' : 'Inherited from global config'}
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <Select
                    value={modeValue}
                    onValueChange={handleModeChange}
                    disabled={setModeMutation.isPending}
                  >
                    <SelectTrigger data-testid="config-mode-select">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {MODE_VALUES.map((mode) => (
                        <SelectItem key={mode} value={mode}>
                          {mode}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {hasSubjectMode && (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={handleResetMode}
                      disabled={deleteModeMutation.isPending}
                      data-testid="config-mode-reset"
                      title="Reset to global default"
                    >
                      <RotateCcw className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              </div>

              {/* Alias */}
              {fullConfig?.alias != null && (
                <div className="flex items-center justify-between" data-testid="config-alias">
                  <div className="space-y-0.5">
                    <label className="text-sm font-medium">Alias</label>
                    <p className="text-xs text-muted-foreground">Read-only subject alias</p>
                  </div>
                  <span className="font-mono text-sm">{fullConfig.alias || '(none)'}</span>
                </div>
              )}

              {/* Compatibility Group */}
              {fullConfig?.compatibilityGroup != null && (
                <div className="flex items-center justify-between" data-testid="config-compat-group">
                  <div className="space-y-0.5">
                    <label className="text-sm font-medium">Compatibility Group</label>
                    <p className="text-xs text-muted-foreground">
                      Schemas in the same group are checked together
                    </p>
                  </div>
                  <span className="font-mono text-sm">
                    {fullConfig.compatibilityGroup || '(none)'}
                  </span>
                </div>
              )}

              {/* Normalize */}
              <div className="flex items-center justify-between" data-testid="config-normalize">
                <div className="space-y-0.5">
                  <label className="text-sm font-medium">Normalize</label>
                  <p className="text-xs text-muted-foreground">
                    Normalize schemas before registration
                  </p>
                </div>
                <span className="flex items-center gap-1.5 text-sm">
                  {fullConfig?.normalize ? (
                    <><Check className="h-4 w-4 text-green-600" /> Enabled</>
                  ) : (
                    <><X className="h-4 w-4 text-muted-foreground" /> Disabled</>
                  )}
                </span>
              </div>

              {/* Validate Fields */}
              <div className="flex items-center justify-between" data-testid="config-validate-fields">
                <div className="space-y-0.5">
                  <label className="text-sm font-medium">Validate Fields</label>
                  <p className="text-xs text-muted-foreground">
                    Validate schema fields during registration
                  </p>
                </div>
                <span className="flex items-center gap-1.5 text-sm">
                  {fullConfig?.validateFields ? (
                    <><Check className="h-4 w-4 text-green-600" /> Enabled</>
                  ) : (
                    <><X className="h-4 w-4 text-muted-foreground" /> Disabled</>
                  )}
                </span>
              </div>

              {/* Reset to Global Buttons */}
              {(hasSubjectConfig || hasSubjectMode) && (
                <div className="border-t pt-4">
                  <div className="flex items-center gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        if (hasSubjectConfig) handleResetConfig();
                        if (hasSubjectMode) handleResetMode();
                      }}
                      disabled={deleteConfigMutation.isPending || deleteModeMutation.isPending}
                      data-testid="config-reset-all"
                    >
                      <RotateCcw className="mr-1 h-4 w-4" /> Reset All to Global Defaults
                    </Button>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* ── Metadata Tab ── */}
        <TabsContent value="metadata" data-testid="subject-tab-content-metadata">
          <div className="space-y-4">
            <MetadataPanel
              metadata={latestVersion?.metadata}
              title="Latest Version Metadata"
            />

            <MetadataPanel
              metadata={fullConfig?.defaultMetadata}
              title="Default Metadata"
            />

            {fullConfig?.overrideMetadata && (
              <MetadataPanel
                metadata={fullConfig.overrideMetadata}
                title="Override Metadata"
              />
            )}
          </div>
        </TabsContent>

        {/* ── Rules Tab ── */}
        <TabsContent value="rules" data-testid="subject-tab-content-rules">
          <div className="space-y-4">
            <RuleSetPanel
              ruleSet={latestVersion?.ruleSet}
              title="Latest Version Rules"
            />

            <RuleSetPanel
              ruleSet={fullConfig?.defaultRuleSet}
              title="Default Rules"
            />

            {fullConfig?.overrideRuleSet && (
              <RuleSetPanel
                ruleSet={fullConfig.overrideRuleSet}
                title="Override Rules"
              />
            )}
          </div>
        </TabsContent>
      </Tabs>

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

function VersionRow({ subject, version, isLatest }: { subject: string; version: number; isLatest: boolean }) {
  const navigate = useNavigate();
  const { data, isLoading } = useSubjectVersion(subject, version);

  return (
    <TableRow
      className="cursor-pointer"
      onClick={() => navigate({
        to: '/ui/subjects/$subject/versions/$version',
        params: { subject, version: String(version) },
      })}
      data-testid={`subject-version-row-${version}`}
    >
      <TableCell>
        <span className="font-medium">v{version}</span>
        {isLatest && (
          <Badge variant="outline" className="ml-2 text-xs">latest</Badge>
        )}
      </TableCell>
      <TableCell>
        {isLoading ? <Skeleton className="h-4 w-16" /> : (data?.id ?? '—')}
      </TableCell>
      <TableCell>
        {isLoading ? <Skeleton className="h-4 w-16" /> : (data?.schemaType ?? '—')}
      </TableCell>
    </TableRow>
  );
}

function formatSchema(schema: string): string {
  try {
    return JSON.stringify(JSON.parse(schema), null, 2);
  } catch {
    return schema; // Protobuf or already formatted
  }
}
