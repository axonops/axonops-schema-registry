import { useState, useEffect, useCallback } from 'react';
import { useNavigate } from '@tanstack/react-router';
import {
  useGlobalMode,
  useSubjects,
  useSubjectMode,
  useSetGlobalMode,
  useDeleteSubjectMode,
} from '@/api/queries';
import { PageBreadcrumbs } from '@/components/shared/breadcrumbs';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { toast } from 'sonner';
import { Info, RotateCcw, Loader2, AlertTriangle } from 'lucide-react';

const MODES = [
  {
    value: 'READWRITE',
    label: 'READWRITE',
    description: 'Normal operation — schemas can be registered and read',
  },
  {
    value: 'READONLY',
    label: 'READONLY',
    description: 'No new schemas can be registered. Read access only.',
  },
  {
    value: 'READONLY_OVERRIDE',
    label: 'READONLY_OVERRIDE',
    description:
      'Read-only for most subjects, but specific subjects can be set to READWRITE',
  },
  {
    value: 'IMPORT',
    label: 'IMPORT',
    description: 'Schemas can be imported with specific IDs — used for migration',
  },
] as const;

type ModeValue = (typeof MODES)[number]['value'];

const breadcrumbs = [{ label: 'Mode Configuration' }];

function SubjectModeRow({
  subject,
  onReset,
  isResetting,
}: {
  subject: string;
  onReset: (subject: string) => void;
  isResetting: boolean;
}) {
  const navigate = useNavigate();
  const { data: modeConfig, isLoading } = useSubjectMode(subject);

  if (isLoading || modeConfig === null || modeConfig === undefined) {
    return null;
  }

  const modeInfo = MODES.find((m) => m.value === modeConfig.mode);

  return (
    <TableRow>
      <TableCell>
        <button
          className="text-primary underline-offset-4 hover:underline font-medium cursor-pointer"
          onClick={() =>
            navigate({
              to: '/ui/subjects/$subject',
              params: { subject },
            })
          }
        >
          {subject}
        </button>
      </TableCell>
      <TableCell>
        <Badge variant="outline">{modeConfig.mode}</Badge>
        {modeInfo && (
          <span className="ml-2 text-xs text-muted-foreground">
            {modeInfo.description}
          </span>
        )}
      </TableCell>
      <TableCell className="text-right">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => onReset(subject)}
          disabled={isResetting}
        >
          {isResetting ? (
            <Loader2 className="mr-1 h-3 w-3 animate-spin" />
          ) : (
            <RotateCcw className="mr-1 h-3 w-3" />
          )}
          Reset to Global
        </Button>
      </TableCell>
    </TableRow>
  );
}

export function GlobalModesPage() {
  const {
    data: globalMode,
    isLoading: isLoadingGlobal,
    isError: isGlobalError,
    error: globalError,
  } = useGlobalMode();
  const { data: subjects } = useSubjects();

  const setGlobalModeMutation = useSetGlobalMode();
  const deleteSubjectModeMutation = useDeleteSubjectMode();

  const [selectedMode, setSelectedMode] = useState<ModeValue>('READWRITE');
  const [resettingSubject, setResettingSubject] = useState<string | null>(null);
  const [overrideCount, setOverrideCount] = useState(0);

  useEffect(() => {
    if (globalMode?.mode) {
      setSelectedMode(globalMode.mode as ModeValue);
    }
  }, [globalMode]);

  const hasChanges = globalMode?.mode !== selectedMode;

  const handleSaveGlobalMode = () => {
    setGlobalModeMutation.mutate(selectedMode, {
      onSuccess: () => {
        toast.success(`Global mode updated to ${selectedMode}`);
      },
      onError: (err) => {
        toast.error(
          err instanceof Error ? err.message : 'Failed to update global mode'
        );
      },
    });
  };

  const handleResetSubjectMode = (subject: string) => {
    setResettingSubject(subject);
    deleteSubjectModeMutation.mutate(subject, {
      onSuccess: () => {
        toast.success(
          `Mode override removed for "${subject}". It now inherits the global mode.`
        );
        setResettingSubject(null);
      },
      onError: (err) => {
        toast.error(
          err instanceof Error
            ? err.message
            : `Failed to reset mode for "${subject}"`
        );
        setResettingSubject(null);
      },
    });
  };

  const handleOverrideCountChange = useCallback((count: number) => {
    setOverrideCount(count);
  }, []);

  if (isLoadingGlobal) {
    return (
      <div>
        <PageBreadcrumbs items={breadcrumbs} />
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      </div>
    );
  }

  if (isGlobalError) {
    return (
      <div>
        <PageBreadcrumbs items={breadcrumbs} />
        <Alert variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>
            {globalError instanceof Error
              ? globalError.message
              : 'Failed to load global mode configuration'}
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  const totalSubjects = subjects?.length ?? 0;

  return (
    <div>
      <PageBreadcrumbs items={breadcrumbs} />

      <div className="space-y-6">
        {/* Global Mode Card */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Info className="h-5 w-5" />
              Global Mode
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <p className="text-sm text-muted-foreground">
              The global mode controls the default behavior for all subjects
              unless a subject has a specific override.
            </p>

            <div className="flex items-end gap-3">
              <div className="flex-1 max-w-sm">
                <Select
                  value={selectedMode}
                  onValueChange={(value) => setSelectedMode(value as ModeValue)}
                >
                  <SelectTrigger data-testid="mode-global-select">
                    <SelectValue placeholder="Select a mode" />
                  </SelectTrigger>
                  <SelectContent>
                    {MODES.map((mode) => (
                      <SelectItem key={mode.value} value={mode.value}>
                        <div className="flex flex-col">
                          <span className="font-medium">{mode.label}</span>
                          <span className="text-xs text-muted-foreground">
                            {mode.description}
                          </span>
                        </div>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <Button
                data-testid="mode-global-save-btn"
                onClick={handleSaveGlobalMode}
                disabled={!hasChanges || setGlobalModeMutation.isPending}
              >
                {setGlobalModeMutation.isPending && (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                )}
                Save
              </Button>
            </div>

            {selectedMode === 'IMPORT' && (
              <Alert>
                <AlertTriangle className="h-4 w-4" />
                <AlertDescription>
                  Import mode allows schemas to be registered with specific IDs.
                  This is intended for migration only.
                </AlertDescription>
              </Alert>
            )}

            {selectedMode === 'READONLY' && (
              <Alert>
                <AlertTriangle className="h-4 w-4" />
                <AlertDescription>
                  Read-only mode prevents any new schema registrations.
                </AlertDescription>
              </Alert>
            )}
          </CardContent>
        </Card>

        {/* Subject Overrides Card */}
        <Card>
          <CardHeader>
            <CardTitle>Subject Overrides</CardTitle>
            {totalSubjects > 0 && (
              <p className="text-sm text-muted-foreground">
                {overrideCount} of {totalSubjects} subject
                {totalSubjects !== 1 ? 's' : ''} have overrides
              </p>
            )}
          </CardHeader>
          <CardContent>
            {!subjects || subjects.length === 0 ? (
              <p className="text-sm text-muted-foreground py-4 text-center">
                No subjects found.
              </p>
            ) : (
              <SubjectOverridesTable
                subjects={subjects}
                onReset={handleResetSubjectMode}
                resettingSubject={resettingSubject}
                onOverrideCountChange={handleOverrideCountChange}
              />
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function SubjectOverridesTable({
  subjects,
  onReset,
  resettingSubject,
  onOverrideCountChange,
}: {
  subjects: string[];
  onReset: (subject: string) => void;
  resettingSubject: string | null;
  onOverrideCountChange: (count: number) => void;
}) {
  return (
    <Table data-testid="mode-overrides-table">
      <TableHeader>
        <TableRow>
          <TableHead>Subject</TableHead>
          <TableHead>Mode</TableHead>
          <TableHead className="text-right">Action</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        <OverrideRows
          subjects={subjects}
          onReset={onReset}
          resettingSubject={resettingSubject}
          onOverrideCountChange={onOverrideCountChange}
        />
      </TableBody>
    </Table>
  );
}

function OverrideRows({
  subjects,
  onReset,
  resettingSubject,
  onOverrideCountChange,
}: {
  subjects: string[];
  onReset: (subject: string) => void;
  resettingSubject: string | null;
  onOverrideCountChange: (count: number) => void;
}) {
  const [overrideSet, setOverrideSet] = useState<Set<string>>(new Set());

  const handleOverrideStatus = useCallback((subject: string, hasOverride: boolean) => {
    setOverrideSet((prev) => {
      const next = new Set(prev);
      if (hasOverride) {
        next.add(subject);
      } else {
        next.delete(subject);
      }
      if (next.size === prev.size && [...next].every((s) => prev.has(s))) return prev;
      return next;
    });
  }, []);

  useEffect(() => {
    onOverrideCountChange(overrideSet.size);
  }, [overrideSet.size, onOverrideCountChange]);

  return (
    <>
      {subjects.map((subject) => (
        <TrackedSubjectModeRow
          key={subject}
          subject={subject}
          onReset={onReset}
          isResetting={resettingSubject === subject}
          onOverrideStatus={handleOverrideStatus}
        />
      ))}
      {overrideSet.size === 0 && (
        <TableRow>
          <TableCell colSpan={3} className="text-center text-muted-foreground py-6">
            No subjects have mode overrides. All subjects use the global mode.
          </TableCell>
        </TableRow>
      )}
    </>
  );
}

function TrackedSubjectModeRow({
  subject,
  onReset,
  isResetting,
  onOverrideStatus,
}: {
  subject: string;
  onReset: (subject: string) => void;
  isResetting: boolean;
  onOverrideStatus: (subject: string, hasOverride: boolean) => void;
}) {
  const { data: modeConfig, isLoading } = useSubjectMode(subject);

  const hasOverride = !isLoading && modeConfig !== null && modeConfig !== undefined;

  useEffect(() => {
    if (!isLoading) {
      onOverrideStatus(subject, hasOverride);
    }
  }, [subject, hasOverride, isLoading, onOverrideStatus]);

  if (!hasOverride) {
    return null;
  }

  return (
    <SubjectModeRow
      subject={subject}
      onReset={onReset}
      isResetting={isResetting}
    />
  );
}
