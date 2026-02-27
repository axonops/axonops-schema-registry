import { useState, useEffect, useRef, useCallback } from 'react';
import { PageBreadcrumbs } from '@/components/shared/breadcrumbs';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Loader2, ExternalLink, FileText, BookOpen, Download } from 'lucide-react';

const breadcrumbs = [{ label: 'API Documentation' }];

const REDOC_CDN = 'https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js';

/**
 * The spec URL. In the Vite build the file is served from public/openapi.yaml
 * at the base path (/ui/). At runtime we try the bundled copy first, then
 * fall back to the backend's /openapi.yaml (requires docs_enabled: true).
 */
function getSpecUrl(): string {
  // Vite sets import.meta.env.BASE_URL to the `base` config value ("/ui/")
  return `${import.meta.env.BASE_URL}openapi.yaml`;
}

function getApiBase(): string {
  return window.location.origin;
}

/**
 * Wait for `window.Redoc` to be defined after the script loads.
 * The CDN script may need a tick after onload before the global is available.
 */
function waitForRedoc(maxWait = 5000): Promise<void> {
  return new Promise((resolve, reject) => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    if ((window as any).Redoc) {
      resolve();
      return;
    }
    const start = Date.now();
    const interval = setInterval(() => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      if ((window as any).Redoc) {
        clearInterval(interval);
        resolve();
      } else if (Date.now() - start > maxWait) {
        clearInterval(interval);
        reject(new Error('Redoc failed to initialize after script loaded'));
      }
    }, 50);
  });
}

export function ApiDocsPage() {
  const [redocReady, setRedocReady] = useState(false);
  const [redocError, setRedocError] = useState<string | null>(null);
  const [darkMode, setDarkMode] = useState(false);
  const redocContainerRef = useRef<HTMLDivElement>(null);

  // Detect dark mode from the document
  useEffect(() => {
    const isDark = document.documentElement.classList.contains('dark');
    setDarkMode(isDark);

    const observer = new MutationObserver(() => {
      setDarkMode(document.documentElement.classList.contains('dark'));
    });
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] });
    return () => observer.disconnect();
  }, []);

  // Load the Redoc standalone script and wait for the global
  useEffect(() => {
    let cancelled = false;

    async function loadRedoc() {
      try {
        // If script already in DOM, just wait for global
        if (document.querySelector('script[data-redoc-standalone]')) {
          await waitForRedoc();
          if (!cancelled) setRedocReady(true);
          return;
        }

        await new Promise<void>((resolve, reject) => {
          const script = document.createElement('script');
          script.src = REDOC_CDN;
          script.async = true;
          script.setAttribute('data-redoc-standalone', 'true');
          script.onload = () => resolve();
          script.onerror = () => reject(new Error('Failed to load Redoc script from CDN'));
          document.head.appendChild(script);
        });

        await waitForRedoc();
        if (!cancelled) setRedocReady(true);
      } catch (err) {
        if (!cancelled) {
          setRedocError(err instanceof Error ? err.message : 'Failed to load Redoc');
        }
      }
    }

    loadRedoc();
    return () => { cancelled = true; };
  }, []);

  const initRedoc = useCallback(() => {
    const container = redocContainerRef.current;
    if (!container || !redocReady) return;
    const specUrl = getSpecUrl();

    // Clear previous render
    container.innerHTML = '';

    const redocOptions = {
      scrollYOffset: 0,
      hideDownloadButton: false,
      expandResponses: '200,201',
      hideHostname: false,
      pathInMiddlePanel: true,
      sortPropsAlphabetically: true,
      requiredPropsFirst: true,
      nativeScrollbars: true,
      theme: {
        colors: {
          primary: { main: darkMode ? '#93c5fd' : '#2563eb' },
          text: { primary: darkMode ? '#e5e7eb' : '#1f2937' },
          http: {
            get: '#22c55e',
            post: '#3b82f6',
            put: '#f59e0b',
            delete: '#ef4444',
            patch: '#8b5cf6',
          },
        },
        typography: {
          fontFamily: 'ui-sans-serif, system-ui, sans-serif',
          fontSize: '14px',
          headings: { fontFamily: 'ui-sans-serif, system-ui, sans-serif' },
          code: { fontFamily: 'ui-monospace, monospace', fontSize: '13px' },
        },
        sidebar: {
          backgroundColor: darkMode ? '#0a0a0a' : '#ffffff',
          textColor: darkMode ? '#d1d5db' : '#374151',
          activeTextColor: darkMode ? '#93c5fd' : '#2563eb',
        },
        rightPanel: {
          backgroundColor: darkMode ? '#1a1a2e' : '#263238',
          textColor: '#ffffff',
        },
      },
    };

    try {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (window as any).Redoc.init(specUrl, redocOptions, container);
    } catch (err) {
      setRedocError(
        err instanceof Error ? err.message : 'Failed to initialize Redoc'
      );
    }
  }, [redocReady, darkMode]);

  // Initialize/re-initialize Redoc when ready or theme changes
  useEffect(() => {
    if (redocReady) {
      initRedoc();
    }
  }, [redocReady, initRedoc]);

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
              The Redoc reference below is bundled with the UI and always available.
            </p>
          </div>
        </CardContent>
      </Card>

      {/* Redoc inline render */}
      {redocError && (
        <Alert variant="destructive" className="mb-4">
          <AlertDescription>{redocError}</AlertDescription>
        </Alert>
      )}

      {!redocReady && !redocError && (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="mr-2 h-6 w-6 animate-spin text-muted-foreground" />
          <span className="text-sm text-muted-foreground">Loading API documentation...</span>
        </div>
      )}

      <div
        ref={redocContainerRef}
        data-testid="api-docs-redoc-container"
        className="rounded-lg border overflow-hidden [&_.redoc-wrap]:!bg-transparent"
      />
    </div>
  );
}
