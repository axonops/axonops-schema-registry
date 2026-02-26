import { useState } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { useSchemasList } from '@/api/queries';
import { PageBreadcrumbs } from '@/components/shared/breadcrumbs';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
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
import { Search, AlertCircle, RefreshCw } from 'lucide-react';

export function SchemaBrowserPage() {
  const [idInput, setIdInput] = useState('');
  const [filter, setFilter] = useState('');
  const { data: schemas, isLoading, isError, error, refetch } = useSchemasList();
  const navigate = useNavigate();

  const breadcrumbs = [{ label: 'Schema Browser' }];

  const handleIdLookup = () => {
    const id = parseInt(idInput, 10);
    if (id > 0) {
      navigate({ to: '/ui/schemas/$id', params: { id: String(id) } });
    }
  };

  const filtered = schemas?.filter((s) =>
    s.subject.toLowerCase().includes(filter.toLowerCase())
  ) ?? [];

  if (isLoading) {
    return (
      <div data-testid="schemas-list-loading">
        <PageBreadcrumbs items={breadcrumbs} />
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-10 w-full" />
          ))}
        </div>
      </div>
    );
  }

  return (
    <div data-testid="schema-browser-page">
      <PageBreadcrumbs items={breadcrumbs} />

      <h1 className="mb-6 text-2xl font-bold">Schema Browser</h1>

      {/* ID Lookup */}
      <div className="mb-6 flex items-center gap-2">
        <div className="text-sm text-muted-foreground">Lookup by Global ID:</div>
        <Input
          type="number"
          placeholder="Enter schema ID..."
          value={idInput}
          onChange={(e) => setIdInput(e.target.value)}
          className="w-48"
          onKeyDown={(e) => e.key === 'Enter' && handleIdLookup()}
          data-testid="schemas-id-input"
        />
        <Button variant="outline" onClick={handleIdLookup} data-testid="schemas-id-lookup-btn">
          Lookup
        </Button>
      </div>

      {isError && (
        <div className="mb-6 flex flex-col items-center gap-4 py-8" data-testid="schemas-list-error">
          <AlertCircle className="h-10 w-10 text-destructive" />
          <p className="text-muted-foreground">
            {error instanceof Error ? error.message : 'Failed to load schemas'}
          </p>
          <Button variant="outline" onClick={() => refetch()}>
            <RefreshCw className="mr-2 h-4 w-4" /> Retry
          </Button>
        </div>
      )}

      {/* Filter */}
      <div className="mb-4">
        <div className="relative max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Filter by subject..."
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="pl-9"
            data-testid="schemas-filter-input"
          />
        </div>
      </div>

      {filtered.length === 0 && !isLoading ? (
        <div className="py-8 text-center text-muted-foreground" data-testid="schemas-list-empty">
          No schemas found
        </div>
      ) : (
        <div className="rounded-md border">
          <Table data-testid="schemas-list-table">
            <TableHeader>
              <TableRow>
                <TableHead>ID</TableHead>
                <TableHead>Subject</TableHead>
                <TableHead>Version</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>References</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filtered.map((schema) => (
                <TableRow
                  key={`${schema.subject}-${schema.version}`}
                  className="cursor-pointer"
                  onClick={() => navigate({ to: '/ui/schemas/$id', params: { id: String(schema.id) } })}
                >
                  <TableCell>{schema.id}</TableCell>
                  <TableCell className="font-medium">{schema.subject}</TableCell>
                  <TableCell>v{schema.version}</TableCell>
                  <TableCell>
                    <Badge variant="outline">{schema.schemaType}</Badge>
                  </TableCell>
                  <TableCell>
                    {schema.references && schema.references.length > 0
                      ? `${schema.references.length} ref${schema.references.length > 1 ? 's' : ''}`
                      : '—'}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
