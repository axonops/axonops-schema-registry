import { useSubjects } from '@/api/queries';
import { useSchemasList, useServerVersion } from '@/api/queries';

export function StatusBar() {
  const { data: subjects } = useSubjects();
  const { data: schemas } = useSchemasList();
  const { data: version } = useServerVersion();

  return (
    <footer
      className="flex h-8 items-center justify-between border-t bg-muted/50 px-4 text-xs text-muted-foreground"
      data-testid="status-bar"
    >
      <div className="flex items-center gap-4">
        {version && (
          <span data-testid="status-bar-version">v{version.version}</span>
        )}
        {subjects && (
          <span data-testid="status-bar-subjects">{subjects.length} subjects</span>
        )}
        {schemas && (
          <span data-testid="status-bar-schemas">{schemas.length} schemas</span>
        )}
      </div>
      <div>
        <span>AxonOps Schema Registry</span>
      </div>
    </footer>
  );
}
