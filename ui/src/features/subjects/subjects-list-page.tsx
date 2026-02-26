import { useState } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { useSubjects } from '@/api/queries';
import { PageBreadcrumbs } from '@/components/shared/breadcrumbs';
import { Pagination, usePagination } from '@/components/shared/pagination';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Skeleton } from '@/components/ui/skeleton';
import { Search, RefreshCw, AlertCircle } from 'lucide-react';

export function SubjectsListPage() {
  const [showDeleted, setShowDeleted] = useState(false);
  const [search, setSearch] = useState('');
  const { data: subjects, isLoading, isError, error, refetch } = useSubjects({ deleted: showDeleted });

  const navigate = useNavigate();

  const filtered = subjects?.filter((s) =>
    s.toLowerCase().includes(search.toLowerCase())
  ) ?? [];

  const { page, totalPages, paged, setPage } = usePagination(filtered);

  const breadcrumbs = [{ label: 'Subjects' }];

  if (isLoading) {
    return (
      <div data-testid="subjects-list-loading">
        <PageBreadcrumbs items={breadcrumbs} />
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-10 w-full" />
          ))}
        </div>
      </div>
    );
  }

  if (isError) {
    return (
      <div data-testid="subjects-list-error">
        <PageBreadcrumbs items={breadcrumbs} />
        <div className="flex flex-col items-center justify-center gap-4 py-12">
          <AlertCircle className="h-10 w-10 text-destructive" />
          <p className="text-muted-foreground">
            {error instanceof Error ? error.message : 'Failed to load subjects'}
          </p>
          <Button variant="outline" onClick={() => refetch()}>
            <RefreshCw className="mr-2 h-4 w-4" /> Retry
          </Button>
        </div>
      </div>
    );
  }

  if (!subjects || subjects.length === 0) {
    return (
      <div data-testid="subjects-list-empty">
        <PageBreadcrumbs items={breadcrumbs} />
        <div className="flex flex-col items-center justify-center gap-4 py-12">
          <p className="text-muted-foreground">No subjects found</p>
          <p className="text-sm text-muted-foreground">
            Register a schema to create your first subject.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div data-testid="subjects-list-page">
      <PageBreadcrumbs items={breadcrumbs} />

      <div className="mb-4 flex items-center justify-between gap-4">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search subjects..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
            data-testid="subjects-search-input"
          />
        </div>
        <div className="flex items-center gap-2">
          <Switch
            id="show-deleted"
            checked={showDeleted}
            onCheckedChange={setShowDeleted}
            data-testid="subjects-deleted-toggle"
          />
          <Label htmlFor="show-deleted" className="text-sm">
            Show deleted
          </Label>
        </div>
      </div>

      <div className="text-sm text-muted-foreground mb-2">
        {filtered.length} subject{filtered.length !== 1 ? 's' : ''}
      </div>

      <div className="rounded-md border">
        <Table data-testid="subjects-list-table">
          <TableHeader>
            <TableRow>
              <TableHead>Subject</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {paged.map((subject) => (
              <TableRow
                key={subject}
                className="cursor-pointer"
                onClick={() => navigate({ to: '/ui/subjects/$subject', params: { subject } })}
                data-testid={`subjects-row-${subject}`}
              >
                <TableCell>
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{subject}</span>
                    {subject.startsWith(':.') && (
                      <Badge variant="outline" className="text-xs">context</Badge>
                    )}
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <Pagination page={page} totalPages={totalPages} onPageChange={setPage} />
    </div>
  );
}
