import { useState, useEffect } from 'react';
import { useNavigate } from '@tanstack/react-router';
import {
  useGlobalConfig,
  useSubjects,
  useSubjectConfig,
  useSetGlobalConfig,
  useSetSubjectConfig,
  useDeleteSubjectConfig,
  useDeleteGlobalConfig,
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
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Label } from '@/components/ui/label';
import { toast } from 'sonner';
import { Info, RotateCcw, Loader2, Plus } from 'lucide-react';

const COMPATIBILITY_LEVELS = [
  {
    value: 'NONE',
    description: 'No compatibility checks are performed',
  },
  {
    value: 'BACKWARD',
    description: 'New schema can read data written by the old schema',
  },
  {
    value: 'BACKWARD_TRANSITIVE',
    description: 'New schema can read data written by all previous versions',
  },
  {
    value: 'FORWARD',
    description: 'Old schema can read data written by the new schema',
  },
  {
    value: 'FORWARD_TRANSITIVE',
    description: 'All previous versions can read data written by the new schema',
  },
  {
    value: 'FULL',
    description: 'Both backward and forward compatible with the previous version',
  },
  {
    value: 'FULL_TRANSITIVE',
    description: 'Both backward and forward compatible with all previous versions',
  },
] as const;

const breadcrumbs = [{ label: 'Compatibility Configuration' }];

interface SubjectConfigRowTrackedProps {
  subject: string;
  onReset: (subject: string) => void;
  isResetting: boolean;
  onOverrideStatus: (subject: string, hasOverride: boolean) => void;
}

function SubjectConfigRowTracked({
  subject,
  onReset,
  isResetting,
  onOverrideStatus,
}: SubjectConfigRowTrackedProps) {
  const navigate = useNavigate();
  const { data: config, isLoading } = useSubjectConfig(subject);

  const hasOverride = !isLoading && config !== null && config !== undefined;

  useEffect(() => {
    if (!isLoading) {
      onOverrideStatus(subject, hasOverride);
    }
  }, [subject, hasOverride, isLoading, onOverrideStatus]);

  if (!hasOverride) {
    return null;
  }

  return (
    <TableRow>
      <TableCell>
        <button
          className="text-primary underline-offset-4 hover:underline cursor-pointer font-medium"
          onClick={() =>
            navigate({ to: '/ui/subjects/$subject', params: { subject } })
          }
        >
          {subject}
        </button>
      </TableCell>
      <TableCell>
        <Badge variant="secondary">{config!.compatibilityLevel}</Badge>
      </TableCell>
      <TableCell className="text-right">
        <Button
          variant="ghost"
          size="sm"
          className="whitespace-nowrap"
          onClick={() => onReset(subject)}
          disabled={isResetting}
        >
          {isResetting ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <RotateCcw className="mr-2 h-4 w-4" />
          )}
          Reset to Global
        </Button>
      </TableCell>
    </TableRow>
  );
}

interface SubjectOverridesTableProps {
  subjects: string[];
  totalSubjects: number;
  resettingSubject: string | null;
  onReset: (subject: string) => void;
}

function SubjectOverridesTable({
  subjects,
  totalSubjects,
  resettingSubject,
  onReset,
}: SubjectOverridesTableProps) {
  const [overrideCount, setOverrideCount] = useState(0);
  const [overrideTracker] = useState<Set<string>>(() => new Set());

  const handleOverrideStatus = (subject: string, hasOverride: boolean) => {
    const hadIt = overrideTracker.has(subject);
    if (hasOverride && !hadIt) {
      overrideTracker.add(subject);
      setOverrideCount(overrideTracker.size);
    } else if (!hasOverride && hadIt) {
      overrideTracker.delete(subject);
      setOverrideCount(overrideTracker.size);
    }
  };

  return (
    <>
      <p className="text-sm text-muted-foreground">
        {overrideCount} of {totalSubjects} subject
        {totalSubjects !== 1 ? 's' : ''} have overrides
      </p>

      <Table data-testid="config-overrides-table">
        <TableHeader>
          <TableRow>
            <TableHead>Subject</TableHead>
            <TableHead>Compatibility Level</TableHead>
            <TableHead className="text-right w-44">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {subjects.map((subject) => (
            <SubjectConfigRowTracked
              key={subject}
              subject={subject}
              onReset={onReset}
              isResetting={resettingSubject === subject}
              onOverrideStatus={handleOverrideStatus}
            />
          ))}
          {overrideCount === 0 && (
            <TableRow>
              <TableCell
                colSpan={3}
                className="text-center text-muted-foreground py-6"
              >
                No subjects have compatibility overrides. All subjects use the
                global default.
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </>
  );
}

export function GlobalConfigPage() {
  const { data: globalConfig, isLoading: isLoadingConfig } = useGlobalConfig();
  const { data: subjects } = useSubjects();
  const setGlobalConfig = useSetGlobalConfig();
  const deleteGlobalConfig = useDeleteGlobalConfig();
  const setSubjectConfig = useSetSubjectConfig();
  const deleteSubjectConfig = useDeleteSubjectConfig();

  const [selectedLevel, setSelectedLevel] = useState<string>('');
  const [resettingSubject, setResettingSubject] = useState<string | null>(null);
  const [showOverrideDialog, setShowOverrideDialog] = useState(false);
  const [overrideSubject, setOverrideSubject] = useState('');
  const [overrideLevel, setOverrideLevel] = useState('BACKWARD');

  useEffect(() => {
    if (globalConfig?.compatibilityLevel) {
      setSelectedLevel(globalConfig.compatibilityLevel);
    }
  }, [globalConfig?.compatibilityLevel]);

  const hasChanges = selectedLevel !== globalConfig?.compatibilityLevel;
  const totalSubjects = subjects?.length ?? 0;

  const handleSave = () => {
    if (!selectedLevel) return;
    setGlobalConfig.mutate(selectedLevel, {
      onSuccess: () => {
        toast.success('Global compatibility level updated');
      },
      onError: (error) => {
        toast.error(
          error instanceof Error
            ? error.message
            : 'Failed to update global compatibility level'
        );
      },
    });
  };

  const handleResetGlobal = () => {
    deleteGlobalConfig.mutate(undefined, {
      onSuccess: () => {
        toast.success('Global compatibility reset to default (BACKWARD)');
      },
      onError: (error) => {
        toast.error(error instanceof Error ? error.message : 'Failed to reset global config');
      },
    });
  };

  const handleSetOverride = () => {
    if (!overrideSubject || !overrideLevel) return;
    setSubjectConfig.mutate(
      { subject: overrideSubject, compatibility: overrideLevel },
      {
        onSuccess: () => {
          toast.success(`Set compatibility for "${overrideSubject}" to ${overrideLevel}`);
          setShowOverrideDialog(false);
          setOverrideSubject('');
          setOverrideLevel('BACKWARD');
        },
        onError: (error) => {
          toast.error(error instanceof Error ? error.message : 'Failed to set subject override');
        },
      }
    );
  };

  const handleResetSubject = (subject: string) => {
    setResettingSubject(subject);
    deleteSubjectConfig.mutate(subject, {
      onSuccess: () => {
        toast.success(`Reset compatibility for "${subject}" to global default`);
        setResettingSubject(null);
      },
      onError: (error) => {
        toast.error(
          error instanceof Error
            ? error.message
            : `Failed to reset compatibility for "${subject}"`
        );
        setResettingSubject(null);
      },
    });
  };

  return (
    <div data-testid="global-config-page">
      <PageBreadcrumbs items={breadcrumbs} />

      <div className="space-y-6">
        {/* Global Compatibility Level */}
        <Card>
          <CardHeader>
            <CardTitle>Global Compatibility Level</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-start gap-3 rounded-md border border-blue-200 bg-blue-50 p-3 dark:border-blue-800 dark:bg-blue-950">
              <Info className="mt-0.5 h-4 w-4 shrink-0 text-blue-600 dark:text-blue-400" />
              <p className="text-sm text-blue-800 dark:text-blue-300">
                Changing the global default affects all subjects that do not have
                a subject-level override.
              </p>
            </div>

            <div className="flex items-end gap-4">
              <div className="flex-1 max-w-sm space-y-2">
                <label
                  htmlFor="global-compat-level"
                  className="text-sm font-medium"
                >
                  Compatibility Level
                </label>
                <Select
                  value={selectedLevel}
                  onValueChange={setSelectedLevel}
                >
                  <SelectTrigger
                    id="global-compat-level"
                    data-testid="config-global-compat-select"
                  >
                    <SelectValue placeholder="Select compatibility level" />
                  </SelectTrigger>
                  <SelectContent>
                    {COMPATIBILITY_LEVELS.map((level) => (
                      <SelectItem key={level.value} value={level.value}>
                        <div className="flex flex-col">
                          <span className="font-medium">{level.value}</span>
                          <span className="text-xs text-muted-foreground">
                            {level.description}
                          </span>
                        </div>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <Button
                onClick={handleSave}
                disabled={
                  !hasChanges || setGlobalConfig.isPending || isLoadingConfig
                }
                data-testid="config-global-compat-save-btn"
              >
                {setGlobalConfig.isPending && (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                )}
                Save
              </Button>
              <Button
                variant="outline"
                onClick={handleResetGlobal}
                disabled={deleteGlobalConfig.isPending}
                data-testid="config-global-reset-btn"
              >
                {deleteGlobalConfig.isPending && (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                )}
                Reset to Default
              </Button>
            </div>

            {globalConfig && (
              <p className="text-sm text-muted-foreground">
                Current global level:{' '}
                <Badge variant="outline">
                  {globalConfig.compatibilityLevel}
                </Badge>
              </p>
            )}
          </CardContent>
        </Card>

        {/* Subject Overrides */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle>Subject Overrides</CardTitle>
              <Button
                size="sm"
                variant="outline"
                onClick={() => setShowOverrideDialog(true)}
                data-testid="config-set-override-btn"
              >
                <Plus className="mr-1 h-4 w-4" /> Set Override
              </Button>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            {subjects && subjects.length > 0 ? (
              <SubjectOverridesTable
                subjects={subjects}
                totalSubjects={totalSubjects}
                resettingSubject={resettingSubject}
                onReset={handleResetSubject}
              />
            ) : (
              <p className="text-sm text-muted-foreground">
                No subjects registered yet.
              </p>
            )}
          </CardContent>
        </Card>
      </div>

      <Dialog open={showOverrideDialog} onOpenChange={setShowOverrideDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Set Subject Compatibility Override</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label>Subject</Label>
              <Select value={overrideSubject} onValueChange={setOverrideSubject}>
                <SelectTrigger data-testid="config-override-subject-select">
                  <SelectValue placeholder="Select a subject" />
                </SelectTrigger>
                <SelectContent>
                  {(subjects ?? []).map((s) => (
                    <SelectItem key={s} value={s}>{s}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Compatibility Level</Label>
              <Select value={overrideLevel} onValueChange={setOverrideLevel}>
                <SelectTrigger data-testid="config-override-level-select">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {COMPATIBILITY_LEVELS.map((level) => (
                    <SelectItem key={level.value} value={level.value}>
                      {level.value}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowOverrideDialog(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleSetOverride}
              disabled={!overrideSubject || setSubjectConfig.isPending}
              data-testid="config-override-save-btn"
            >
              {setSubjectConfig.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Set Override
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
