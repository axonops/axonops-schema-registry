import { useNavigate, useParams } from '@tanstack/react-router';
import { useSubjectVersions, useSubjectVersion, useSubjectConfig, useSubjectMode, useGlobalConfig } from '@/api/queries';
import { PageBreadcrumbs } from '@/components/shared/breadcrumbs';
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
import { AlertCircle, RefreshCw } from 'lucide-react';
import { Button } from '@/components/ui/button';

export function SubjectDetailPage() {
  const { subject } = useParams({ strict: false }) as { subject: string };
  const { data: versions, isLoading, isError, error, refetch } = useSubjectVersions(subject);
  const { data: latestVersion } = useSubjectVersion(subject, 'latest');
  const { data: subjectConfig } = useSubjectConfig(subject);
  const { data: subjectMode } = useSubjectMode(subject);
  const { data: globalConfig } = useGlobalConfig();
  const navigate = useNavigate();

  const breadcrumbs = [
    { label: 'Subjects', href: '/ui/subjects' },
    { label: subject },
  ];

  const compatLevel = subjectConfig?.compatibilityLevel ?? globalConfig?.compatibilityLevel ?? 'BACKWARD';
  const modeValue = subjectMode?.mode ?? 'READWRITE';

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
        <div className="text-sm text-muted-foreground">
          {versions?.length ?? 0} version{(versions?.length ?? 0) !== 1 ? 's' : ''}
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
