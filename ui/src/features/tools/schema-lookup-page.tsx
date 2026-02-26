import { useState } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { useSchemaLookup } from '@/api/queries';
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
import { toast } from 'sonner';
import { SearchCheck, Loader2 } from 'lucide-react';

interface LookupResult {
  subject: string;
  id: number;
  version: number;
  schemaType: string;
  schema: string;
}

type LookupOutcome =
  | { kind: 'found'; data: LookupResult }
  | { kind: 'not-found' };

export function SchemaLookupPage() {
  const navigate = useNavigate();
  const lookupMutation = useSchemaLookup();

  const [subject, setSubject] = useState('');
  const [schemaType, setSchemaType] = useState<SchemaType>('AVRO');
  const [schemaContent, setSchemaContent] = useState('');
  const [outcome, setOutcome] = useState<LookupOutcome | null>(null);

  const canLookup =
    subject.trim() !== '' &&
    schemaContent.trim() !== '' &&
    !lookupMutation.isPending;

  const handleLookup = () => {
    setOutcome(null);
    lookupMutation.mutate(
      {
        subject,
        schema: schemaContent,
        schemaType,
      },
      {
        onSuccess: (data) => {
          setOutcome({ kind: 'found', data });
        },
        onError: (err) => {
          if (err instanceof ApiClientError && err.status === 404) {
            setOutcome({ kind: 'not-found' });
          } else {
            toast.error(
              err instanceof ApiClientError
                ? err.message
                : 'Schema lookup failed'
            );
          }
        },
      }
    );
  };

  return (
    <div data-testid="schema-lookup-page">
      <PageBreadcrumbs items={[{ label: 'Schema Lookup' }]} />

      <div className="mb-6">
        <h1 className="text-2xl font-bold">Schema Lookup</h1>
        <p className="text-sm text-muted-foreground">
          Check if a schema already exists in a subject
        </p>
      </div>

      {/* Lookup Parameters */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle className="text-lg">Lookup Parameters</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Subject */}
          <div className="space-y-2">
            <Label htmlFor="lookup-subject">Subject</Label>
            <Input
              id="lookup-subject"
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
              placeholder="e.g., orders-value"
              data-testid="schema-lookup-subject-input"
            />
          </div>

          {/* Schema Type */}
          <div className="space-y-2">
            <Label>Schema Type</Label>
            <Select
              value={schemaType}
              onValueChange={(v) => setSchemaType(v as SchemaType)}
            >
              <SelectTrigger data-testid="schema-lookup-type-select">
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
              height="300px"
              data-testid="schema-lookup-editor"
            />
          </div>
        </CardContent>
      </Card>

      {/* Lookup Button */}
      <Button
        onClick={handleLookup}
        disabled={!canLookup}
        data-testid="schema-lookup-btn"
      >
        {lookupMutation.isPending ? (
          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
        ) : (
          <SearchCheck className="mr-2 h-4 w-4" />
        )}
        Lookup Schema
      </Button>

      {/* Results */}
      {outcome?.kind === 'found' && (
        <Card className="mt-6 border-green-200 dark:border-green-800" data-testid="schema-lookup-result">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-lg">
              <Badge variant="outline" className="border-green-500 text-green-600 dark:text-green-400">
                Schema Found
              </Badge>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
              <dt className="font-medium text-muted-foreground">Subject</dt>
              <dd>{outcome.data.subject}</dd>

              <dt className="font-medium text-muted-foreground">Version</dt>
              <dd>
                <button
                  type="button"
                  className="text-primary underline-offset-4 hover:underline"
                  onClick={() =>
                    navigate({
                      to: '/ui/subjects/$subject/versions/$version',
                      params: {
                        subject: outcome.data.subject,
                        version: String(outcome.data.version),
                      },
                    })
                  }
                >
                  v{outcome.data.version}
                </button>
              </dd>

              <dt className="font-medium text-muted-foreground">Schema ID</dt>
              <dd>
                <button
                  type="button"
                  className="text-primary underline-offset-4 hover:underline"
                  onClick={() =>
                    navigate({
                      to: '/ui/schemas/$id',
                      params: { id: String(outcome.data.id) },
                    })
                  }
                >
                  {outcome.data.id}
                </button>
              </dd>
            </dl>
          </CardContent>
        </Card>
      )}

      {outcome?.kind === 'not-found' && (
        <Card className="mt-6 border-blue-200 dark:border-blue-800" data-testid="schema-lookup-result">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-lg">
              <Badge variant="outline" className="border-blue-500 text-blue-600 dark:text-blue-400">
                Schema Not Found
              </Badge>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              The provided schema does not match any existing version in subject{' '}
              <span className="font-medium text-foreground">{subject}</span>.
              You can register it as a new schema.
            </p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
