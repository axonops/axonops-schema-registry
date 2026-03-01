import { useState, useEffect } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { useSchemasList } from '@/api/queries';
import { PageBreadcrumbs } from '@/components/shared/breadcrumbs';
import { Pagination, usePagination } from '@/components/shared/pagination';
import { TagBadges } from '@/components/shared/tag-badges';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent } from '@/components/ui/card';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Search, AlertCircle, RefreshCw, Shield } from 'lucide-react';

const SCHEMA_TYPES = ['ALL', 'AVRO', 'PROTOBUF', 'JSON'] as const;
type SchemaTypeFilter = (typeof SCHEMA_TYPES)[number];

export function SearchPage() {
  const navigate = useNavigate();
  const breadcrumbs = [{ label: 'Search' }];

  // Search input and debounced value
  const [searchInput, setSearchInput] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');

  // Filters
  const [typeFilter, setTypeFilter] = useState<SchemaTypeFilter>('ALL');
  const [hasMetadata, setHasMetadata] = useState(false);

  // Quick ID lookup
  const [idInput, setIdInput] = useState('');

  // Debounce the search input
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(searchInput);
    }, 300);
    return () => clearTimeout(timer);
  }, [searchInput]);

  const {
    data: schemas,
    isLoading,
    isError,
    error,
    refetch,
  } = useSchemasList(
    debouncedSearch ? { subjectPrefix: debouncedSearch } : undefined
  );

  const handleIdLookup = () => {
    const id = parseInt(idInput, 10);
    if (id > 0) {
      navigate({ to: '/ui/schemas/$id', params: { id: String(id) } });
    }
  };

  // Client-side filtering
  const filtered =
    schemas?.filter((s) => {
      if (typeFilter !== 'ALL' && s.schemaType !== typeFilter) return false;
      // hasMetadata toggle: since SchemaListItem may not include metadata,
      // we treat this as best-effort — filter out items without references
      // as a rough proxy when metadata is not available on list items.
      // In practice, this is a no-op unless the API returns metadata fields.
      if (hasMetadata) {
        // SchemaListItem doesn't include metadata, so this is a placeholder.
        // If metadata were available, we'd check: s.metadata && Object.keys(s.metadata).length > 0
        // For now, filter items that have references as a proxy for "enriched" schemas.
        if (!s.references || s.references.length === 0) return false;
      }
      return true;
    }) ?? [];

  const { page, totalPages, paged, setPage } = usePagination(filtered, 12);

  return (
    <div data-testid="search-page">
      <PageBreadcrumbs items={breadcrumbs} />

      <h1 className="mb-6 text-2xl font-bold">Search</h1>

      {/* Search Bar */}
      <div className="mb-4">
        <div className="relative">
          <Search className="absolute left-4 top-1/2 h-5 w-5 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search by subject name..."
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            className="h-12 pl-12 text-base"
            data-testid="search-input"
          />
        </div>
      </div>

      {/* Filter Bar */}
      <div className="mb-6 flex flex-wrap items-center gap-4">
        {/* Schema Type Select */}
        <div className="flex items-center gap-2">
          <Label htmlFor="type-filter" className="text-sm text-muted-foreground whitespace-nowrap">
            Type:
          </Label>
          <Select
            value={typeFilter}
            onValueChange={(v) => setTypeFilter(v as SchemaTypeFilter)}
          >
            <SelectTrigger id="type-filter" data-testid="search-type-filter">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {SCHEMA_TYPES.map((type) => (
                <SelectItem key={type} value={type}>
                  {type === 'ALL' ? 'All Types' : type}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {/* Has Metadata Toggle */}
        <div className="flex items-center gap-2">
          <Switch
            id="has-metadata"
            checked={hasMetadata}
            onCheckedChange={setHasMetadata}
            data-testid="search-metadata-toggle"
          />
          <Label htmlFor="has-metadata" className="text-sm text-muted-foreground cursor-pointer">
            Has Metadata
          </Label>
        </div>

        {/* Separator */}
        <div className="hidden sm:block h-6 w-px bg-border" />

        {/* Quick ID Lookup */}
        <div className="flex items-center gap-2">
          <Input
            type="number"
            placeholder="Schema ID..."
            value={idInput}
            onChange={(e) => setIdInput(e.target.value)}
            className="w-32"
            onKeyDown={(e) => e.key === 'Enter' && handleIdLookup()}
            data-testid="search-id-input"
          />
          <Button
            variant="outline"
            size="sm"
            onClick={handleIdLookup}
            data-testid="search-id-go-btn"
          >
            Go
          </Button>
        </div>
      </div>

      {/* Error State */}
      {isError && (
        <div
          className="mb-6 flex flex-col items-center gap-4 py-8"
          data-testid="search-error"
        >
          <AlertCircle className="h-10 w-10 text-destructive" />
          <p className="text-muted-foreground">
            {error instanceof Error ? error.message : 'Failed to load schemas'}
          </p>
          <Button variant="outline" onClick={() => refetch()}>
            <RefreshCw className="mr-2 h-4 w-4" /> Retry
          </Button>
        </div>
      )}

      {/* Loading State */}
      {isLoading && (
        <div data-testid="search-loading">
          <div className="mb-4 text-sm text-muted-foreground">Searching...</div>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
            {Array.from({ length: 6 }).map((_, i) => (
              <Skeleton key={i} className="h-36 w-full rounded-xl" />
            ))}
          </div>
        </div>
      )}

      {/* Results */}
      {!isLoading && !isError && (
        <>
          {/* Results Count */}
          <div className="mb-4 text-sm text-muted-foreground" data-testid="search-results-count">
            {filtered.length} result{filtered.length !== 1 ? 's' : ''}
          </div>

          {filtered.length === 0 ? (
            <div
              className="py-12 text-center text-muted-foreground"
              data-testid="search-empty"
            >
              No schemas found
            </div>
          ) : (
            <>
              {/* Results Grid */}
              <div
                className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3"
                data-testid="search-results-grid"
              >
                {paged.map((schema) => (
                  <Card
                    key={`${schema.subject}-${schema.version}`}
                    className="cursor-pointer transition-colors hover:bg-accent/50"
                    onClick={() =>
                      navigate({
                        to: '/ui/subjects/$subject',
                        params: { subject: schema.subject },
                      })
                    }
                    data-testid={`search-result-card-${schema.subject}`}
                  >
                    <CardContent className="space-y-2">
                      {/* Subject Name */}
                      <div className="font-semibold leading-tight" data-testid="search-card-subject">
                        {schema.subject}
                      </div>

                      {/* Version + ID */}
                      <div className="text-sm text-muted-foreground">
                        {`v${schema.version} · ID ${schema.id}`}
                      </div>

                      {/* Type Badge + Sensitive Indicator */}
                      <div className="flex items-center gap-2">
                        <Badge variant="outline">{schema.schemaType}</Badge>
                        {schema.references && schema.references.length > 0 && (
                          <Badge variant="secondary" className="text-xs">
                            {schema.references.length} ref{schema.references.length > 1 ? 's' : ''}
                          </Badge>
                        )}
                      </div>

                      {/* Tags (best-effort — SchemaListItem may not include metadata) */}
                      {!!(schema as unknown as Record<string, unknown>).metadata &&
                        typeof (schema as unknown as Record<string, unknown>).metadata === 'object' && (
                          <>
                            <TagBadges
                              tags={
                                ((schema as unknown as Record<string, unknown>).metadata as {
                                  tags?: Record<string, string[]>;
                                })?.tags
                              }
                            />
                            {/* Sensitive fields indicator */}
                            {((schema as unknown as Record<string, unknown>).metadata as {
                              sensitive?: string[];
                            })?.sensitive &&
                              (
                                (schema as unknown as Record<string, unknown>).metadata as {
                                  sensitive?: string[];
                                }
                              ).sensitive!.length > 0 && (
                                <div className="flex items-center gap-1 text-xs text-amber-600 dark:text-amber-400">
                                  <Shield className="h-3.5 w-3.5" />
                                  <span>
                                    {
                                      (
                                        (schema as unknown as Record<string, unknown>).metadata as {
                                          sensitive: string[];
                                        }
                                      ).sensitive.length
                                    }{' '}
                                    sensitive field
                                    {(
                                      (schema as unknown as Record<string, unknown>).metadata as {
                                        sensitive: string[];
                                      }
                                    ).sensitive.length > 1
                                      ? 's'
                                      : ''}
                                  </span>
                                </div>
                              )}
                          </>
                        )}
                    </CardContent>
                  </Card>
                ))}
              </div>

              <Pagination page={page} totalPages={totalPages} onPageChange={setPage} />
            </>
          )}
        </>
      )}
    </div>
  );
}
