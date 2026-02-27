import { Shield } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { TagBadges } from '@/components/shared/tag-badges';
import type { SchemaMetadata } from '@/api/queries';

interface MetadataPanelProps {
  metadata: SchemaMetadata | undefined;
  title?: string;
}

function hasContent(metadata: SchemaMetadata | undefined): boolean {
  if (!metadata) return false;
  const hasTags = metadata.tags && Object.keys(metadata.tags).length > 0;
  const hasProperties = metadata.properties && Object.keys(metadata.properties).length > 0;
  const hasSensitive = metadata.sensitive && metadata.sensitive.length > 0;
  return !!(hasTags || hasProperties || hasSensitive);
}

export function MetadataPanel({ metadata, title = 'Metadata' }: MetadataPanelProps) {
  if (!hasContent(metadata)) {
    return (
      <Card data-testid="metadata-panel">
        <CardHeader>
          <CardTitle className="text-sm">{title}</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground" data-testid="metadata-empty">
            No metadata
          </p>
        </CardContent>
      </Card>
    );
  }

  const hasTags = metadata!.tags && Object.keys(metadata!.tags).length > 0;
  const hasProperties = metadata!.properties && Object.keys(metadata!.properties).length > 0;
  const hasSensitive = metadata!.sensitive && metadata!.sensitive.length > 0;

  return (
    <Card data-testid="metadata-panel">
      <CardHeader>
        <CardTitle className="text-sm">{title}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-5">
        {hasTags && (
          <div data-testid="metadata-tags-section">
            <h4 className="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">
              Tags
            </h4>
            <TagBadges tags={metadata!.tags} />
          </div>
        )}

        {hasProperties && (
          <div data-testid="metadata-properties-section">
            <h4 className="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">
              Properties
            </h4>
            <div className="overflow-hidden rounded-md border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b bg-muted/50">
                    <th className="px-3 py-1.5 text-left font-medium text-muted-foreground">
                      Key
                    </th>
                    <th className="px-3 py-1.5 text-left font-medium text-muted-foreground">
                      Value
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {Object.entries(metadata!.properties!).map(([key, value]) => (
                    <tr key={key} className="border-b last:border-b-0">
                      <td className="px-3 py-1.5 font-mono text-xs">{key}</td>
                      <td className="px-3 py-1.5 font-mono text-xs">{value}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {hasSensitive && (
          <div data-testid="metadata-sensitive-section">
            <h4 className="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">
              Sensitive Fields
            </h4>
            <div className="flex flex-wrap gap-1.5">
              {metadata!.sensitive!.map((field) => (
                <Badge
                  key={field}
                  variant="outline"
                  className="border-amber-300 bg-amber-50 text-amber-800 dark:border-amber-700 dark:bg-amber-900/30 dark:text-amber-300"
                  data-testid={`sensitive-field-${field}`}
                >
                  <Shield className="h-3 w-3" />
                  {field}
                </Badge>
              ))}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
