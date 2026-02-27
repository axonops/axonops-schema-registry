import { useNavigate } from '@tanstack/react-router';
import { useSubjects, useSchemasList, useHealthReady } from '@/api/queries';
import type { SchemaListItem } from '@/api/queries';
import { PageBreadcrumbs } from '@/components/shared/breadcrumbs';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  BookOpen,
  Database,
  FileCode,
  Activity,
  FilePlus2,
  CircleCheck,
  SearchCheck,
  Search,
  FileText,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';

const breadcrumbs = [{ label: 'Dashboard' }];

export function DashboardPage() {
  const { data: subjects, isLoading: loadingSubjects } = useSubjects();
  const { data: schemas, isLoading: loadingSchemas } = useSchemasList();

  const { data: health, isLoading: loadingHealth } = useHealthReady();

  const navigate = useNavigate();

  const totalSubjects = subjects?.length ?? 0;
  const uniqueSchemaIds = schemas ? new Set(schemas.map((s) => s.id)).size : 0;

  const typeCounts = deriveTypeCounts(schemas);

  const recentSchemas = schemas
    ? [...schemas].sort((a, b) => b.id - a.id).slice(0, 10)
    : [];

  const isHealthy = health?.status === 'UP';

  return (
    <div data-testid="dashboard-page">
      <PageBreadcrumbs items={breadcrumbs} />

      <h1 className="mb-6 text-2xl font-bold">Dashboard</h1>

      {/* Row 1: Stats cards */}
      <div className="mb-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4" data-testid="dashboard-stats">
        <StatsCard
          icon={BookOpen}
          label="Total Subjects"
          value={totalSubjects}
          isLoading={loadingSubjects}
          testId="dashboard-stat-subjects"
        />
        <StatsCard
          icon={Database}
          label="Total Schemas"
          value={uniqueSchemaIds}
          isLoading={loadingSchemas}
          testId="dashboard-stat-schemas"
        />
        <SchemaTypesCard
          typeCounts={typeCounts}
          isLoading={loadingSchemas}
          testId="dashboard-stat-types"
        />
        <HealthCard
          isHealthy={isHealthy}
          isLoading={loadingHealth}
          testId="dashboard-stat-health"
        />
      </div>

      {/* Row 2: Two columns */}
      <div className="grid gap-6 md:grid-cols-2">
        {/* Left: Recent Schemas */}
        <Card data-testid="dashboard-recent-schemas">
          <CardHeader>
            <CardTitle className="text-sm font-medium">Recent Schemas</CardTitle>
          </CardHeader>
          <CardContent>
            {loadingSchemas ? (
              <div className="space-y-3">
                {Array.from({ length: 5 }).map((_, i) => (
                  <Skeleton key={i} className="h-8 w-full" />
                ))}
              </div>
            ) : recentSchemas.length === 0 ? (
              <p className="py-4 text-center text-sm text-muted-foreground">
                No schemas registered yet.
              </p>
            ) : (
              <div className="rounded-md border">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Subject</TableHead>
                      <TableHead>Version</TableHead>
                      <TableHead>Type</TableHead>
                      <TableHead className="text-right">ID</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {recentSchemas.map((schema) => (
                      <TableRow
                        key={`${schema.subject}-${schema.version}`}
                        className="cursor-pointer"
                        onClick={() =>
                          navigate({
                            to: '/ui/subjects/$subject/versions/$version',
                            params: {
                              subject: schema.subject,
                              version: String(schema.version),
                            },
                          })
                        }
                        data-testid={`dashboard-recent-row-${schema.id}`}
                      >
                        <TableCell className="max-w-[200px] truncate font-medium">
                          {schema.subject}
                        </TableCell>
                        <TableCell>{schema.version}</TableCell>
                        <TableCell>
                          <Badge variant="outline">{schema.schemaType}</Badge>
                        </TableCell>
                        <TableCell className="text-right">{schema.id}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Right: Quick Actions */}
        <Card data-testid="dashboard-quick-actions">
          <CardHeader>
            <CardTitle className="text-sm font-medium">Quick Actions</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-3">
              <QuickActionButton
                icon={FilePlus2}
                label="Register Schema"
                onClick={() => navigate({ to: '/ui/register' })}
              />
              <QuickActionButton
                icon={BookOpen}
                label="Browse Subjects"
                onClick={() => navigate({ to: '/ui/subjects' })}
              />
              <QuickActionButton
                icon={CircleCheck}
                label="Check Compatibility"
                onClick={() => navigate({ to: '/ui/tools/compatibility' })}
              />
              <QuickActionButton
                icon={SearchCheck}
                label="Look Up Schema"
                onClick={() => navigate({ to: '/ui/tools/lookup' })}
              />
              <QuickActionButton
                icon={Search}
                label="Search Schemas"
                onClick={() => navigate({ to: '/ui/search' })}
              />
              <QuickActionButton
                icon={FileText}
                label="API Documentation"
                onClick={() => navigate({ to: '/ui/api-docs' })}
              />
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

// ── Helper components ──

interface StatsCardProps {
  icon: LucideIcon;
  label: string;
  value: number | string;
  isLoading: boolean;
  testId: string;
}

function StatsCard({ icon: Icon, label, value, isLoading, testId }: StatsCardProps) {
  return (
    <Card data-testid={testId}>
      <CardContent className="p-6">
        <div className="flex items-start justify-between">
          <div>
            {isLoading ? (
              <Skeleton className="mb-1 h-9 w-16" />
            ) : (
              <div className="text-3xl font-bold">{value}</div>
            )}
            <div className="text-sm text-muted-foreground">{label}</div>
          </div>
          <Icon className="h-5 w-5 text-muted-foreground" />
        </div>
      </CardContent>
    </Card>
  );
}

interface SchemaTypesCardProps {
  typeCounts: Record<string, number>;
  isLoading: boolean;
  testId: string;
}

function SchemaTypesCard({ typeCounts, isLoading, testId }: SchemaTypesCardProps) {
  const entries = Object.entries(typeCounts).sort(([a], [b]) => a.localeCompare(b));
  const total = entries.reduce((sum, [, count]) => sum + count, 0);
  return (
    <Card data-testid={testId}>
      <CardContent className="p-6">
        <div className="flex items-start justify-between">
          <div>
            {isLoading ? (
              <Skeleton className="mb-1 h-9 w-16" />
            ) : (
              <div className="text-3xl font-bold">{total}</div>
            )}
            <div className="text-sm text-muted-foreground">Schema Types</div>
          </div>
          <FileCode className="h-5 w-5 text-muted-foreground" />
        </div>
        {!isLoading && entries.length > 0 && (
          <div className="mt-3 flex flex-wrap gap-1.5">
            {entries.map(([type, count]) => (
              <Badge key={type} variant="secondary" className="text-xs">
                {type}: {count}
              </Badge>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

interface HealthCardProps {
  isHealthy: boolean;
  isLoading: boolean;
  testId: string;
}

function HealthCard({ isHealthy, isLoading, testId }: HealthCardProps) {
  return (
    <Card data-testid={testId}>
      <CardContent className="p-6">
        <div className="flex items-start justify-between">
          <div>
            {isLoading ? (
              <Skeleton className="mb-1 h-9 w-24" />
            ) : (
              <div className="flex items-center gap-2">
                <span
                  className={`inline-block h-3 w-3 rounded-full ${isHealthy ? 'bg-green-500' : 'bg-red-500'}`}
                />
                <span
                  className={`text-xl font-semibold ${isHealthy ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'}`}
                >
                  {isHealthy ? 'Healthy' : 'Unhealthy'}
                </span>
              </div>
            )}
            <div className="text-sm text-muted-foreground">Health</div>
          </div>
          <Activity className="h-5 w-5 text-muted-foreground" />
        </div>
      </CardContent>
    </Card>
  );
}

interface QuickActionButtonProps {
  icon: LucideIcon;
  label: string;
  onClick: () => void;
}

function QuickActionButton({ icon: Icon, label, onClick }: QuickActionButtonProps) {
  return (
    <Button
      variant="outline"
      className="h-auto w-full justify-start gap-2 px-4 py-3"
      onClick={onClick}
    >
      <Icon className="h-4 w-4" />
      <span>{label}</span>
    </Button>
  );
}

// ── Helpers ──

function deriveTypeCounts(
  schemas: SchemaListItem[] | undefined
): Record<string, number> {
  if (!schemas) return {};
  const counts: Record<string, number> = {};
  for (const s of schemas) {
    const t = s.schemaType || 'AVRO';
    counts[t] = (counts[t] ?? 0) + 1;
  }
  return counts;
}

