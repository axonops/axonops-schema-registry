import { useState, useRef, useCallback } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useGlobalMode, useSetGlobalMode, queryKeys } from '@/api/queries';
import { apiFetch } from '@/api/client';
import { PageBreadcrumbs } from '@/components/shared/breadcrumbs';
import { SchemaEditor } from '@/components/schema-editor/schema-editor';
import type { SchemaType } from '@/components/schema-editor/monaco-config';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { Progress } from '@/components/ui/progress';
import { toast } from 'sonner';
import { AlertTriangle, Upload, FileJson, CheckCircle, XCircle, Loader2 } from 'lucide-react';

// ── Types ──

interface BulkSchemaEntry {
  schema: string;
  schemaType: string;
  subject: string;
  id: number;
  version: number;
  references?: Array<{ name: string; subject: string; version: number }>;
}

interface ImportResultEntry {
  subject: string;
  version: number;
  id: number;
  success: boolean;
  error?: string;
}

type ImportMethod = 'single' | 'bulk';

// ── Component ──

export function ImportPage() {
  const queryClient = useQueryClient();

  // Mode check
  const { data: globalMode, isLoading: modeLoading } = useGlobalMode();
  const setModeMutation = useSetGlobalMode();

  // Import method toggle
  const [importMethod, setImportMethod] = useState<ImportMethod>('single');

  // Single schema form state
  const [subject, setSubject] = useState('');
  const [schemaId, setSchemaId] = useState('');
  const [version, setVersion] = useState('');
  const [schemaType, setSchemaType] = useState<SchemaType>('AVRO');
  const [schemaContent, setSchemaContent] = useState('');

  // Bulk import state
  const [bulkSchemas, setBulkSchemas] = useState<BulkSchemaEntry[] | null>(null);
  const [bulkFileName, setBulkFileName] = useState<string | null>(null);
  const [bulkParseError, setBulkParseError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Progress/results state
  const [isImporting, setIsImporting] = useState(false);
  const [importProgress, setImportProgress] = useState(0);
  const [importResults, setImportResults] = useState<ImportResultEntry[]>([]);

  const isImportMode = globalMode?.mode === 'IMPORT';

  // ── Switch to IMPORT mode ──

  const handleSwitchToImportMode = useCallback(() => {
    setModeMutation.mutate('IMPORT', {
      onSuccess: () => {
        toast.success('Registry switched to IMPORT mode');
      },
      onError: () => {
        toast.error('Failed to switch to IMPORT mode');
      },
    });
  }, [setModeMutation]);

  // ── File handling ──

  const parseBulkFile = useCallback((content: string, fileName: string) => {
    setBulkFileName(fileName);
    setBulkParseError(null);
    try {
      const parsed = JSON.parse(content);
      if (!Array.isArray(parsed)) {
        setBulkParseError('JSON file must contain an array of schema objects.');
        setBulkSchemas(null);
        return;
      }
      for (let i = 0; i < parsed.length; i++) {
        const entry = parsed[i];
        if (!entry.schema || !entry.schemaType || !entry.subject || entry.id == null || entry.version == null) {
          setBulkParseError(
            `Entry at index ${i} is missing required fields (schema, schemaType, subject, id, version).`
          );
          setBulkSchemas(null);
          return;
        }
      }
      setBulkSchemas(parsed as BulkSchemaEntry[]);
    } catch {
      setBulkParseError('Failed to parse JSON file. Please check the file format.');
      setBulkSchemas(null);
    }
  }, []);

  const handleFileDrop = useCallback(
    (e: React.DragEvent<HTMLDivElement>) => {
      e.preventDefault();
      e.stopPropagation();
      const file = e.dataTransfer.files[0];
      if (!file) return;
      const reader = new FileReader();
      reader.onload = (ev) => {
        const text = ev.target?.result;
        if (typeof text === 'string') {
          parseBulkFile(text, file.name);
        }
      };
      reader.readAsText(file);
    },
    [parseBulkFile]
  );

  const handleDragOver = useCallback((e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
  }, []);

  const handleFileSelect = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0];
      if (!file) return;
      const reader = new FileReader();
      reader.onload = (ev) => {
        const text = ev.target?.result;
        if (typeof text === 'string') {
          parseBulkFile(text, file.name);
        }
      };
      reader.readAsText(file);
      // Reset input so selecting the same file again triggers change
      e.target.value = '';
    },
    [parseBulkFile]
  );

  // ── Single schema import ──

  const singleImportMutation = useMutation({
    mutationFn: async () => {
      const body: Record<string, unknown> = {
        schema: schemaContent,
        schemaType,
        id: parseInt(schemaId, 10),
        version: parseInt(version, 10),
      };
      return apiFetch<{ id: number }>(
        `/subjects/${encodeURIComponent(subject)}/versions`,
        {
          method: 'POST',
          body: JSON.stringify(body),
        }
      );
    },
    onSuccess: (data) => {
      const result: ImportResultEntry = {
        subject,
        version: parseInt(version, 10),
        id: data.id,
        success: true,
      };
      setImportResults([result]);
      toast.success(`Schema imported successfully (ID: ${data.id})`);
      queryClient.invalidateQueries({ queryKey: queryKeys.subjects.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.schemas.all });
    },
    onError: (err: Error) => {
      const result: ImportResultEntry = {
        subject,
        version: parseInt(version, 10),
        id: parseInt(schemaId, 10),
        success: false,
        error: err.message,
      };
      setImportResults([result]);
      toast.error(`Import failed: ${err.message}`);
    },
  });

  // ── Bulk import ──

  const handleBulkImport = useCallback(async () => {
    if (!bulkSchemas || bulkSchemas.length === 0) return;

    setIsImporting(true);
    setImportProgress(0);
    setImportResults([]);

    const results: ImportResultEntry[] = [];

    for (let i = 0; i < bulkSchemas.length; i++) {
      const entry = bulkSchemas[i];
      try {
        const body: Record<string, unknown> = {
          schema: entry.schema,
          schemaType: entry.schemaType,
          id: entry.id,
          version: entry.version,
        };
        if (entry.references && entry.references.length > 0) {
          body.references = entry.references;
        }
        const data = await apiFetch<{ id: number }>(
          `/subjects/${encodeURIComponent(entry.subject)}/versions`,
          {
            method: 'POST',
            body: JSON.stringify(body),
          }
        );
        results.push({
          subject: entry.subject,
          version: entry.version,
          id: data.id,
          success: true,
        });
      } catch (err) {
        results.push({
          subject: entry.subject,
          version: entry.version,
          id: entry.id,
          success: false,
          error: err instanceof Error ? err.message : 'Unknown error',
        });
      }

      setImportProgress(Math.round(((i + 1) / bulkSchemas.length) * 100));
      setImportResults([...results]);
    }

    setIsImporting(false);

    const successCount = results.filter((r) => r.success).length;
    const failCount = results.length - successCount;

    if (failCount === 0) {
      toast.success(`All ${successCount} schemas imported successfully`);
    } else {
      toast.warning(`${successCount} succeeded, ${failCount} failed`);
    }

    queryClient.invalidateQueries({ queryKey: queryKeys.subjects.all });
    queryClient.invalidateQueries({ queryKey: queryKeys.schemas.all });
  }, [bulkSchemas, queryClient]);

  // ── Derived state ──

  const canSubmitSingle =
    subject.trim() !== '' &&
    schemaId.trim() !== '' &&
    version.trim() !== '' &&
    schemaContent.trim() !== '' &&
    !singleImportMutation.isPending;

  const canSubmitBulk =
    bulkSchemas !== null &&
    bulkSchemas.length > 0 &&
    !isImporting;

  const successCount = importResults.filter((r) => r.success).length;
  const failCount = importResults.filter((r) => !r.success).length;

  // ── Render ──

  return (
    <div data-testid="import-page">
      <PageBreadcrumbs items={[{ label: 'Import Schemas' }]} />

      <h1 className="mb-6 text-2xl font-bold">Import Schemas</h1>

      {/* Mode Warning */}
      {!modeLoading && !isImportMode && (
        <Alert
          className="mb-6 border-yellow-500/50 bg-yellow-50 text-yellow-900 dark:bg-yellow-950/30 dark:text-yellow-200"
          data-testid="import-mode-warning"
        >
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription className="flex items-center justify-between">
            <span>
              The registry is currently in <strong>{globalMode?.mode ?? 'READWRITE'}</strong> mode.
              Importing schemas with preserved IDs requires <strong>IMPORT</strong> mode.
            </span>
            <Button
              variant="outline"
              size="sm"
              className="ml-4 shrink-0 border-yellow-600 text-yellow-900 hover:bg-yellow-100 dark:border-yellow-500 dark:text-yellow-200 dark:hover:bg-yellow-900/50"
              onClick={handleSwitchToImportMode}
              disabled={setModeMutation.isPending}
              data-testid="import-switch-mode-btn"
            >
              {setModeMutation.isPending && <Loader2 className="mr-1 h-4 w-4 animate-spin" />}
              Switch to IMPORT mode
            </Button>
          </AlertDescription>
        </Alert>
      )}

      {/* Import Method Toggle */}
      <div className="mb-6" data-testid="import-method-radio">
        <Label className="mb-2 block text-sm font-medium">Import Method</Label>
        <div className="flex gap-2">
          <Button
            variant={importMethod === 'single' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setImportMethod('single')}
          >
            <FileJson className="mr-1.5 h-4 w-4" />
            Single Schema
          </Button>
          <Button
            variant={importMethod === 'bulk' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setImportMethod('bulk')}
          >
            <Upload className="mr-1.5 h-4 w-4" />
            Bulk JSON File
          </Button>
        </div>
      </div>

      <Separator className="mb-6" />

      {/* ── Single Schema Form ── */}
      {importMethod === 'single' && (
        <Card>
          <CardHeader>
            <CardTitle>Import Single Schema</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Subject */}
            <div className="space-y-2">
              <Label htmlFor="import-subject">Subject</Label>
              <Input
                id="import-subject"
                value={subject}
                onChange={(e) => setSubject(e.target.value)}
                placeholder="e.g., orders-value"
                data-testid="import-subject-input"
              />
            </div>

            {/* Schema ID + Version */}
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="import-id">Schema ID</Label>
                <Input
                  id="import-id"
                  type="number"
                  min={1}
                  value={schemaId}
                  onChange={(e) => setSchemaId(e.target.value)}
                  placeholder="e.g., 1"
                  data-testid="import-id-input"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="import-version">Version</Label>
                <Input
                  id="import-version"
                  type="number"
                  min={1}
                  value={version}
                  onChange={(e) => setVersion(e.target.value)}
                  placeholder="e.g., 1"
                  data-testid="import-version-input"
                />
              </div>
            </div>

            {/* Schema Type */}
            <div className="space-y-2">
              <Label>Schema Type</Label>
              <Select
                value={schemaType}
                onValueChange={(v) => setSchemaType(v as SchemaType)}
              >
                <SelectTrigger data-testid="import-type-select">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="AVRO">AVRO</SelectItem>
                  <SelectItem value="PROTOBUF">PROTOBUF</SelectItem>
                  <SelectItem value="JSON">JSON</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {/* Schema Editor */}
            <div className="space-y-2">
              <Label>Schema</Label>
              <SchemaEditor
                value={schemaContent}
                onChange={setSchemaContent}
                schemaType={schemaType}
                height="350px"
                data-testid="import-schema-editor"
              />
            </div>

            {/* Submit */}
            <Button
              onClick={() => singleImportMutation.mutate()}
              disabled={!canSubmitSingle}
              data-testid="import-submit-btn"
            >
              {singleImportMutation.isPending && (
                <Loader2 className="mr-1 h-4 w-4 animate-spin" />
              )}
              Import Schema
            </Button>
          </CardContent>
        </Card>
      )}

      {/* ── Bulk JSON File ── */}
      {importMethod === 'bulk' && (
        <Card>
          <CardHeader>
            <CardTitle>Bulk Import from JSON File</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Dropzone */}
            <div
              className="flex min-h-[160px] cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed border-muted-foreground/30 bg-muted/30 p-8 text-center transition-colors hover:border-muted-foreground/50 hover:bg-muted/50"
              onDrop={handleFileDrop}
              onDragOver={handleDragOver}
              onClick={() => fileInputRef.current?.click()}
              data-testid="import-file-dropzone"
            >
              <Upload className="mb-3 h-10 w-10 text-muted-foreground/60" />
              <p className="text-sm font-medium text-muted-foreground">
                Drag and drop a JSON file here, or click to browse
              </p>
              <p className="mt-1 text-xs text-muted-foreground/70">
                Expects an array of objects with schema, schemaType, subject, id, version fields
              </p>
              {bulkFileName && !bulkParseError && (
                <Badge variant="outline" className="mt-3">
                  <FileJson className="mr-1 h-3 w-3" />
                  {bulkFileName}
                </Badge>
              )}
              <input
                ref={fileInputRef}
                type="file"
                accept=".json,application/json"
                className="hidden"
                onChange={handleFileSelect}
              />
            </div>

            {/* Parse error */}
            {bulkParseError && (
              <Alert variant="destructive">
                <XCircle className="h-4 w-4" />
                <AlertDescription>{bulkParseError}</AlertDescription>
              </Alert>
            )}

            {/* Preview count */}
            {bulkSchemas && bulkSchemas.length > 0 && (
              <div
                className="flex items-center gap-2 text-sm"
                data-testid="import-preview-count"
              >
                <FileJson className="h-4 w-4 text-muted-foreground" />
                <span>
                  <strong>{bulkSchemas.length}</strong> schemas to import
                </span>
                <Badge variant="outline" className="ml-auto">
                  {bulkFileName}
                </Badge>
              </div>
            )}

            {/* Submit */}
            <Button
              onClick={handleBulkImport}
              disabled={!canSubmitBulk}
              data-testid="import-bulk-submit-btn"
            >
              {isImporting && <Loader2 className="mr-1 h-4 w-4 animate-spin" />}
              {isImporting
                ? `Importing... (${importProgress}%)`
                : `Import ${bulkSchemas?.length ?? 0} Schemas`}
            </Button>
          </CardContent>
        </Card>
      )}

      {/* ── Progress ── */}
      {(isImporting || importResults.length > 0) && (
        <div className="mt-6 space-y-4" data-testid="import-progress">
          {isImporting && (
            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm text-muted-foreground">
                <span>Importing schemas...</span>
                <span>{importProgress}%</span>
              </div>
              <Progress value={importProgress} />
            </div>
          )}

          {!isImporting && importResults.length > 0 && (
            <div className="flex items-center gap-3 text-sm">
              {successCount > 0 && (
                <Badge variant="outline" className="text-green-600">
                  <CheckCircle className="mr-1 h-3 w-3" />
                  {successCount} succeeded
                </Badge>
              )}
              {failCount > 0 && (
                <Badge variant="outline" className="text-destructive">
                  <XCircle className="mr-1 h-3 w-3" />
                  {failCount} failed
                </Badge>
              )}
            </div>
          )}
        </div>
      )}

      {/* ── Results Panel ── */}
      {importResults.length > 0 && (
        <Card className="mt-4" data-testid="import-results-panel">
          <CardHeader>
            <CardTitle className="text-sm font-medium">Import Results</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="max-h-[400px] space-y-2 overflow-y-auto">
              {importResults.map((result, i) => (
                <div
                  key={`${result.subject}-${result.version}-${i}`}
                  className={`flex items-center justify-between rounded-md border px-3 py-2 text-sm ${
                    result.success
                      ? 'border-green-200 bg-green-50 dark:border-green-900/50 dark:bg-green-950/20'
                      : 'border-red-200 bg-red-50 dark:border-red-900/50 dark:bg-red-950/20'
                  }`}
                >
                  <div className="flex items-center gap-2">
                    {result.success ? (
                      <CheckCircle className="h-4 w-4 shrink-0 text-green-600" />
                    ) : (
                      <XCircle className="h-4 w-4 shrink-0 text-destructive" />
                    )}
                    <span className="font-medium">{result.subject}</span>
                    <span className="text-muted-foreground">v{result.version}</span>
                  </div>
                  <div className="text-right">
                    {result.success ? (
                      <Badge variant="outline" className="text-green-600">
                        ID: {result.id}
                      </Badge>
                    ) : (
                      <span className="text-xs text-destructive">{result.error}</span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
