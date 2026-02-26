import { useState } from 'react';
import { useCheckCompatibility } from '@/api/queries';
import { ApiClientError } from '@/api/client';
import { PageBreadcrumbs } from '@/components/shared/breadcrumbs';
import { SchemaEditor } from '@/components/schema-editor/schema-editor';
import type { SchemaType } from '@/components/schema-editor/monaco-config';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { toast } from 'sonner';
import { CircleCheck, CircleX, Loader2, AlertTriangle } from 'lucide-react';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface Reference {
  name: string;
  subject: string;
  version: string;
}

type CheckAgainstMode = 'all' | 'latest' | 'specific';

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function CompatibilityCheckPage() {
  const [subject, setSubject] = useState('');
  const [schemaType, setSchemaType] = useState<SchemaType>('AVRO');
  const [checkAgainstMode, setCheckAgainstMode] = useState<CheckAgainstMode>('all');
  const [specificVersion, setSpecificVersion] = useState('');
  const [schemaContent, setSchemaContent] = useState('');
  const [references, setReferences] = useState<Reference[]>([]);
  const [showReferences, setShowReferences] = useState(false);

  const [result, setResult] = useState<{
    checked: boolean;
    compatible: boolean;
    messages: string[];
  } | null>(null);

  const checkMutation = useCheckCompatibility();

  // ── Derived state ──

  const resolvedCheckAgainst = (() => {
    if (checkAgainstMode === 'all') return undefined;
    if (checkAgainstMode === 'latest') return 'latest';
    const parsed = parseInt(specificVersion, 10);
    return isNaN(parsed) ? undefined : parsed;
  })();

  const canSubmit =
    subject.trim() !== '' &&
    schemaContent.trim() !== '' &&
    !checkMutation.isPending &&
    (checkAgainstMode !== 'specific' || specificVersion.trim() !== '');

  // ── References ──

  const addReference = () => {
    setReferences([...references, { name: '', subject: '', version: 'latest' }]);
    setShowReferences(true);
  };

  const updateReference = (index: number, field: keyof Reference, value: string) => {
    const updated = [...references];
    updated[index] = { ...updated[index], [field]: value };
    setReferences(updated);
  };

  const removeReference = (index: number) => {
    setReferences(references.filter((_, i) => i !== index));
  };

  // ── Submit ──

  const handleCheck = () => {
    setResult(null);

    const refs = references
      .filter((r) => r.subject && r.name)
      .map((r) => ({
        name: r.name,
        subject: r.subject,
        version: r.version === 'latest' ? -1 : parseInt(r.version, 10),
      }));

    let schema: string;
    try {
      schema = schemaType === 'PROTOBUF' ? schemaContent : JSON.stringify(JSON.parse(schemaContent));
    } catch {
      toast.error('Invalid JSON in schema body. Please fix the syntax and try again.');
      return;
    }

    checkMutation.mutate(
      {
        subject,
        version: resolvedCheckAgainst,
        schema,
        schemaType,
        references: refs.length > 0 ? refs : undefined,
      },
      {
        onSuccess: (data) => {
          setResult({
            checked: true,
            compatible: data.is_compatible,
            messages: data.messages ?? [],
          });
        },
        onError: (err) => {
          if (err instanceof ApiClientError) {
            setResult({
              checked: true,
              compatible: false,
              messages: [err.message],
            });
          } else {
            toast.error('Compatibility check failed');
          }
        },
      }
    );
  };

  // ── Render ──

  return (
    <div data-testid="compat-check-page">
      <PageBreadcrumbs items={[{ label: 'Compatibility Check' }]} />

      <div className="mb-6">
        <h1 className="text-2xl font-bold">Compatibility Check</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Test whether a new schema is compatible with existing versions of a subject
          before registering it.
        </p>
      </div>

      {/* Check Parameters */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle>Check Parameters</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Subject */}
          <div className="space-y-2">
            <Label htmlFor="compat-subject">Subject</Label>
            <Input
              id="compat-subject"
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
              placeholder="e.g., orders-value"
              data-testid="compat-check-subject-input"
            />
          </div>

          {/* Schema Type */}
          <div className="space-y-2">
            <Label>Schema Type</Label>
            <Select
              value={schemaType}
              onValueChange={(v) => setSchemaType(v as SchemaType)}
            >
              <SelectTrigger data-testid="compat-check-type-select">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="AVRO">AVRO</SelectItem>
                <SelectItem value="PROTOBUF">PROTOBUF</SelectItem>
                <SelectItem value="JSON">JSON</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Check Against */}
          <div className="space-y-2">
            <Label>Check Against</Label>
            <div className="flex items-center gap-3">
              <Select
                value={checkAgainstMode}
                onValueChange={(v) => {
                  setCheckAgainstMode(v as CheckAgainstMode);
                  if (v !== 'specific') setSpecificVersion('');
                }}
              >
                <SelectTrigger className="w-48">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All versions</SelectItem>
                  <SelectItem value="latest">Latest</SelectItem>
                  <SelectItem value="specific">Specific version</SelectItem>
                </SelectContent>
              </Select>
              {checkAgainstMode === 'specific' && (
                <Input
                  type="number"
                  min={1}
                  value={specificVersion}
                  onChange={(e) => setSpecificVersion(e.target.value)}
                  placeholder="Version number"
                  className="w-40"
                  data-testid="compat-check-version-input"
                />
              )}
            </div>
          </div>

          {/* Schema Editor */}
          <div className="space-y-2">
            <Label>Schema</Label>
            <SchemaEditor
              value={schemaContent}
              onChange={setSchemaContent}
              schemaType={schemaType}
              height="300px"
              data-testid="compat-check-schema-editor"
            />
          </div>

          {/* References (collapsible) */}
          <div>
            <button
              type="button"
              className="flex items-center gap-1 text-sm font-medium"
              onClick={() => setShowReferences(!showReferences)}
            >
              {showReferences ? '\u25BC' : '\u25B6'} References ({references.length})
            </button>
            {showReferences && (
              <div className="mt-2 space-y-2">
                {references.map((ref, i) => (
                  <div key={i} className="flex items-center gap-2">
                    <Input
                      placeholder="Subject"
                      value={ref.subject}
                      onChange={(e) => updateReference(i, 'subject', e.target.value)}
                      className="flex-1"
                    />
                    <Input
                      placeholder="Version"
                      value={ref.version}
                      onChange={(e) => updateReference(i, 'version', e.target.value)}
                      className="w-24"
                    />
                    <Input
                      placeholder="Reference name"
                      value={ref.name}
                      onChange={(e) => updateReference(i, 'name', e.target.value)}
                      className="flex-1"
                    />
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => removeReference(i)}
                    >
                      <CircleX className="h-4 w-4" />
                    </Button>
                  </div>
                ))}
                <Button variant="outline" size="sm" onClick={addReference}>
                  Add Reference
                </Button>
              </div>
            )}
          </div>

          {/* Submit */}
          <Button
            onClick={handleCheck}
            disabled={!canSubmit}
            data-testid="compat-check-btn"
          >
            {checkMutation.isPending && (
              <Loader2 className="mr-1 h-4 w-4 animate-spin" />
            )}
            Check Compatibility
          </Button>
        </CardContent>
      </Card>

      {/* Results */}
      {result?.checked && (
        <Card data-testid="compat-check-result">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-sm font-medium">
              {result.compatible ? (
                <Badge className="gap-1 bg-green-100 text-green-800 hover:bg-green-100 dark:bg-green-900/30 dark:text-green-400">
                  <CircleCheck className="h-3.5 w-3.5" />
                  Compatible
                </Badge>
              ) : (
                <Badge variant="destructive" className="gap-1">
                  <CircleX className="h-3.5 w-3.5" />
                  Incompatible
                </Badge>
              )}
            </CardTitle>
          </CardHeader>
          {result.messages.length > 0 && (
            <CardContent>
              <Alert
                variant={result.compatible ? 'default' : 'destructive'}
                className={result.compatible ? '' : undefined}
              >
                <AlertTriangle className="h-4 w-4" />
                <AlertDescription>
                  <ul className="list-disc space-y-1 pl-4 text-sm">
                    {result.messages.map((msg, i) => (
                      <li key={i}>{msg}</li>
                    ))}
                  </ul>
                </AlertDescription>
              </Alert>
            </CardContent>
          )}
        </Card>
      )}
    </div>
  );
}
