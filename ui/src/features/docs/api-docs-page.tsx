import { useState } from 'react';
import { PageBreadcrumbs } from '@/components/shared/breadcrumbs';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { ExternalLink, FileText, BookOpen, Download, Loader2 } from 'lucide-react';

const breadcrumbs = [{ label: 'API Documentation' }];

function getApiBase(): string {
  return window.location.origin;
}

function getDocsUrl(): string {
  return `${import.meta.env.BASE_URL}api-docs/index.html`;
}

export function ApiDocsPage() {
  const [iframeLoaded, setIframeLoaded] = useState(false);
  const [iframeError, setIframeError] = useState(false);
  const apiBase = getApiBase();

  return (
    <div data-testid="api-docs-page">
      <PageBreadcrumbs items={breadcrumbs} />

      {/* Quick links header */}
      <Card className="mb-6" data-testid="api-docs-links-card">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <FileText className="h-5 w-5" />
            API Documentation
          </CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground mb-4">
            AxonOps Schema Registry provides a Confluent-compatible REST API for managing
            schemas, subjects, compatibility, and configuration. The full API reference is
            rendered below from the OpenAPI specification.
          </p>

          <div className="flex flex-wrap gap-3">
            <Button variant="outline" size="sm" asChild data-testid="api-docs-swagger-link">
              <a href={`${apiBase}/docs`} target="_blank" rel="noopener noreferrer">
                <BookOpen className="mr-2 h-4 w-4" />
                Swagger UI
                <ExternalLink className="ml-2 h-3 w-3" />
              </a>
            </Button>
            <Button variant="outline" size="sm" asChild data-testid="api-docs-openapi-link">
              <a href={`${apiBase}/openapi.yaml`} target="_blank" rel="noopener noreferrer">
                <Download className="mr-2 h-4 w-4" />
                OpenAPI Spec (YAML)
                <ExternalLink className="ml-2 h-3 w-3" />
              </a>
            </Button>
          </div>

          <div className="mt-4 flex flex-wrap gap-2">
            <Badge variant="outline">OpenAPI 3.0</Badge>
            <Badge variant="outline">REST</Badge>
            <Badge variant="outline">Confluent Compatible</Badge>
            <Badge variant="secondary">Avro</Badge>
            <Badge variant="secondary">Protobuf</Badge>
            <Badge variant="secondary">JSON Schema</Badge>
          </div>

          <div className="mt-4 rounded-md border border-blue-200 bg-blue-50 p-3 dark:border-blue-800 dark:bg-blue-950">
            <p className="text-xs text-blue-800 dark:text-blue-300">
              <strong>Note:</strong> The Swagger UI link requires the backend to have{' '}
              <code className="rounded bg-blue-100 px-1 dark:bg-blue-900">server.docs_enabled: true</code>.
              The API reference below is bundled with the UI and always available.
            </p>
          </div>
        </CardContent>
      </Card>

      {/* Static HTML docs in iframe */}
      {iframeError && (
        <Alert variant="destructive" className="mb-4">
          <AlertDescription>
            Failed to load API documentation. Run <code>make docs-api</code> and rebuild the UI to generate the static documentation.
          </AlertDescription>
        </Alert>
      )}

      {!iframeLoaded && !iframeError && (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="mr-2 h-6 w-6 animate-spin text-muted-foreground" />
          <span className="text-sm text-muted-foreground">Loading API documentation...</span>
        </div>
      )}

      <div
        className="rounded-lg border overflow-hidden"
        data-testid="api-docs-iframe-container"
        style={{ display: iframeLoaded ? 'block' : 'none' }}
      >
        <iframe
          src={getDocsUrl()}
          title="API Documentation"
          className="w-full border-0"
          style={{ height: 'calc(100vh - 120px)', minHeight: '600px' }}
          onLoad={() => setIframeLoaded(true)}
          onError={() => setIframeError(true)}
          sandbox="allow-scripts allow-same-origin"
        />
      </div>
    </div>
  );
}
