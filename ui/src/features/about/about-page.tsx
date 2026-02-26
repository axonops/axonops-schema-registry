import { useServerVersion, useClusterId, useSchemaTypes, useSubjects, useSchemasList } from '@/api/queries';
import { PageBreadcrumbs } from '@/components/shared/breadcrumbs';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { Badge } from '@/components/ui/badge';

export function AboutPage() {
  const { data: version, isLoading: loadingVersion } = useServerVersion();
  const { data: clusterId, isLoading: loadingCluster } = useClusterId();
  const { data: schemaTypes } = useSchemaTypes();
  const { data: subjects } = useSubjects();
  const { data: schemas } = useSchemasList();

  const breadcrumbs = [{ label: 'About' }];

  return (
    <div data-testid="about-page">
      <PageBreadcrumbs items={breadcrumbs} />

      <h1 className="mb-6 text-2xl font-bold">About</h1>

      <Card className="mb-6" data-testid="about-info-panel">
        <CardHeader>
          <CardTitle>AxonOps Schema Registry</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <InfoRow label="Version" testId="about-version">
            {loadingVersion ? <Skeleton className="h-4 w-24" /> : version?.version ?? '—'}
          </InfoRow>
          <InfoRow label="Commit" testId="about-commit">
            {loadingVersion ? <Skeleton className="h-4 w-32" /> : version?.commit ?? '—'}
          </InfoRow>
          <InfoRow label="Cluster ID" testId="about-cluster-id">
            {loadingCluster ? <Skeleton className="h-4 w-40" /> : clusterId?.id ?? '—'}
          </InfoRow>
          <InfoRow label="Schema Types" testId="about-schema-types">
            {schemaTypes ? (
              <div className="flex gap-1">
                {schemaTypes.map((t) => (
                  <Badge key={t} variant="outline">{t}</Badge>
                ))}
              </div>
            ) : (
              <Skeleton className="h-4 w-32" />
            )}
          </InfoRow>
          <InfoRow label="GitHub" testId="about-github">
            <a
              href="https://github.com/axonops/axonops-schema-registry"
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary hover:underline"
            >
              github.com/axonops/axonops-schema-registry
            </a>
          </InfoRow>
        </CardContent>
      </Card>

      <Card data-testid="about-stats-panel">
        <CardHeader>
          <CardTitle className="text-sm font-medium">Statistics</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
            <StatCard label="Subjects" value={subjects?.length ?? 0} testId="about-stat-subjects" />
            <StatCard label="Schemas" value={schemas?.length ?? 0} testId="about-stat-schemas" />
            <StatCard label="Types" value={schemaTypes?.length ?? 0} testId="about-stat-types" />
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function InfoRow({ label, children, testId }: { label: string; children: React.ReactNode; testId: string }) {
  return (
    <div className="flex items-center gap-4" data-testid={testId}>
      <span className="w-28 text-sm text-muted-foreground">{label}:</span>
      <span className="text-sm">{children}</span>
    </div>
  );
}

function StatCard({ label, value, testId }: { label: string; value: number; testId: string }) {
  return (
    <div className="rounded-lg border p-4 text-center" data-testid={testId}>
      <div className="text-3xl font-bold">{value}</div>
      <div className="text-sm text-muted-foreground">{label}</div>
    </div>
  );
}
