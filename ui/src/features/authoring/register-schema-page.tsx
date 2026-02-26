import { useState, useCallback, useMemo } from 'react';
import { useNavigate, useParams } from '@tanstack/react-router';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch, ApiClientError } from '@/api/client';
import { useSubjectVersion, queryKeys } from '@/api/queries';
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
import { Switch } from '@/components/ui/switch';
import { Separator } from '@/components/ui/separator';
import { toast } from 'sonner';
import { CheckCircle, XCircle, Plus, X, Loader2, FileCode } from 'lucide-react';

// ---------------------------------------------------------------------------
// Schema templates
// ---------------------------------------------------------------------------

interface SchemaTemplate {
  label: string;
  value: string;
  schema: string;
}

const AVRO_TEMPLATES: SchemaTemplate[] = [
  {
    label: 'Simple Record',
    value: 'avro-simple-record',
    schema: JSON.stringify(
      {
        type: 'record',
        name: 'ExampleRecord',
        namespace: 'com.example',
        fields: [
          { name: 'id', type: 'long' },
          { name: 'name', type: 'string' },
          { name: 'active', type: 'boolean' },
        ],
      },
      null,
      2
    ),
  },
  {
    label: 'Record with Nullable',
    value: 'avro-nullable-record',
    schema: JSON.stringify(
      {
        type: 'record',
        name: 'UserRecord',
        namespace: 'com.example',
        fields: [
          { name: 'id', type: 'long' },
          { name: 'username', type: 'string' },
          { name: 'email', type: ['null', 'string'], default: null },
          { name: 'age', type: ['null', 'int'], default: null },
        ],
      },
      null,
      2
    ),
  },
  {
    label: 'Enum Type',
    value: 'avro-enum',
    schema: JSON.stringify(
      {
        type: 'enum',
        name: 'Status',
        namespace: 'com.example',
        symbols: ['ACTIVE', 'INACTIVE', 'PENDING', 'DELETED'],
      },
      null,
      2
    ),
  },
];

const PROTOBUF_TEMPLATES: SchemaTemplate[] = [
  {
    label: 'proto3 Message',
    value: 'proto-message',
    schema: `syntax = "proto3";

package example;

message ExampleMessage {
  int64 id = 1;
  string name = 2;
  bool active = 3;
}`,
  },
  {
    label: 'Message with Enum',
    value: 'proto-enum',
    schema: `syntax = "proto3";

package example;

enum Status {
  STATUS_UNSPECIFIED = 0;
  STATUS_ACTIVE = 1;
  STATUS_INACTIVE = 2;
}

message UserMessage {
  int64 id = 1;
  string name = 2;
  Status status = 3;
}`,
  },
];

const JSON_TEMPLATES: SchemaTemplate[] = [
  {
    label: 'Object Schema',
    value: 'json-object',
    schema: JSON.stringify(
      {
        $schema: 'http://json-schema.org/draft-07/schema#',
        type: 'object',
        properties: {
          id: { type: 'integer' },
          name: { type: 'string', minLength: 1 },
          email: { type: 'string', format: 'email' },
        },
        required: ['id', 'name'],
        additionalProperties: false,
      },
      null,
      2
    ),
  },
  {
    label: 'Array Schema',
    value: 'json-array',
    schema: JSON.stringify(
      {
        $schema: 'http://json-schema.org/draft-07/schema#',
        type: 'array',
        items: {
          type: 'object',
          properties: {
            id: { type: 'integer' },
            value: { type: 'string' },
          },
          required: ['id', 'value'],
        },
        minItems: 1,
        uniqueItems: true,
      },
      null,
      2
    ),
  },
];

const TEMPLATES_BY_TYPE: Record<SchemaType, SchemaTemplate[]> = {
  AVRO: AVRO_TEMPLATES,
  PROTOBUF: PROTOBUF_TEMPLATES,
  JSON: JSON_TEMPLATES,
};

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface Reference {
  name: string;
  subject: string;
  version: string;
}

export function RegisterSchemaPage() {
  const params = useParams({ strict: false }) as { subject?: string };
  const isNewVersion = !!params.subject;
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [subject, setSubject] = useState(params.subject ?? '');
  const [schemaType, setSchemaType] = useState<SchemaType>('AVRO');
  const [schema, setSchema] = useState('');
  const [normalize, setNormalize] = useState(false);
  const [references, setReferences] = useState<Reference[]>([]);
  const [showReferences, setShowReferences] = useState(false);

  // Compat check state
  const [compatResult, setCompatResult] = useState<{
    checked: boolean;
    compatible: boolean;
    messages: string[];
  } | null>(null);

  // Load latest version for "Start from latest"
  const { data: latestVersion } = useSubjectVersion(params.subject ?? '', 'latest');

  const handleStartFromLatest = useCallback(() => {
    if (!latestVersion) return;
    try {
      const formatted = JSON.stringify(JSON.parse(latestVersion.schema), null, 2);
      setSchema(formatted);
    } catch {
      setSchema(latestVersion.schema);
    }
    setSchemaType(latestVersion.schemaType as SchemaType);
  }, [latestVersion]);

  // Template handling
  const availableTemplates = useMemo(
    () => TEMPLATES_BY_TYPE[schemaType] ?? [],
    [schemaType]
  );

  const handleTemplateSelect = useCallback(
    (templateValue: string) => {
      const template = availableTemplates.find((t) => t.value === templateValue);
      if (template) {
        setSchema(template.schema);
      }
    },
    [availableTemplates]
  );

  // Compatibility check
  const compatMutation = useMutation({
    mutationFn: async () => {
      const schemaStr = schemaType === 'PROTOBUF' ? schema : JSON.stringify(JSON.parse(schema));
      return apiFetch<{ is_compatible: boolean; messages?: string[] }>(
        `/compatibility/subjects/${encodeURIComponent(subject)}/versions?verbose=true`,
        {
          method: 'POST',
          body: JSON.stringify({
            schema: schemaStr,
            schemaType,
            references: references.filter(r => r.subject && r.name).map(r => ({
              name: r.name,
              subject: r.subject,
              version: r.version === 'latest' ? -1 : parseInt(r.version, 10),
            })),
          }),
        }
      );
    },
    onSuccess: (data) => {
      setCompatResult({
        checked: true,
        compatible: data.is_compatible,
        messages: data.messages ?? [],
      });
    },
    onError: (err) => {
      if (err instanceof ApiClientError) {
        setCompatResult({
          checked: true,
          compatible: false,
          messages: [err.message],
        });
      }
    },
  });

  // Register mutation
  const registerMutation = useMutation({
    mutationFn: async () => {
      const schemaStr = schemaType === 'PROTOBUF' ? schema : JSON.stringify(JSON.parse(schema));
      const qs = normalize ? '?normalize=true' : '';
      return apiFetch<{ id: number }>(
        `/subjects/${encodeURIComponent(subject)}/versions${qs}`,
        {
          method: 'POST',
          body: JSON.stringify({
            schema: schemaStr,
            schemaType,
            references: references.filter(r => r.subject && r.name).map(r => ({
              name: r.name,
              subject: r.subject,
              version: r.version === 'latest' ? -1 : parseInt(r.version, 10),
            })),
          }),
        }
      );
    },
    onSuccess: (data) => {
      toast.success(`Schema registered (ID: ${data.id})`);
      queryClient.invalidateQueries({ queryKey: queryKeys.subjects.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.schemas.all });
      navigate({ to: '/ui/subjects/$subject', params: { subject } });
    },
    onError: (err) => {
      if (err instanceof ApiClientError) {
        if (err.status === 409) {
          toast.warning(err.message);
        } else {
          toast.error(err.message);
        }
      } else {
        toast.error('Failed to register schema');
      }
    },
  });

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

  const breadcrumbs = isNewVersion
    ? [
        { label: 'Subjects', href: '/ui/subjects' },
        { label: params.subject!, href: `/ui/subjects/${encodeURIComponent(params.subject!)}` },
        { label: 'Register New Version' },
      ]
    : [
        { label: 'Subjects', href: '/ui/subjects' },
        { label: 'Register New Schema' },
      ];

  const canSubmit = subject.trim() && schema.trim() && !registerMutation.isPending;
  const editorIsEmpty = !schema.trim();

  return (
    <div data-testid="register-schema-page">
      <PageBreadcrumbs items={breadcrumbs} />

      <h1 className="mb-6 text-2xl font-bold">
        {isNewVersion ? `Register New Version — ${params.subject}` : 'Register New Schema'}
      </h1>

      {/* Subject */}
      <div className="mb-4 space-y-2">
        <Label htmlFor="subject">Subject</Label>
        <Input
          id="subject"
          value={subject}
          onChange={(e) => setSubject(e.target.value)}
          placeholder="e.g., orders-value"
          readOnly={isNewVersion}
          data-testid="register-subject-input"
        />
      </div>

      {/* Schema Type */}
      <div className="mb-4 space-y-2">
        <Label>Schema Type</Label>
        <Select
          value={schemaType}
          onValueChange={(v) => setSchemaType(v as SchemaType)}
        >
          <SelectTrigger data-testid="register-type-select">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="AVRO">AVRO</SelectItem>
            <SelectItem value="PROTOBUF">PROTOBUF</SelectItem>
            <SelectItem value="JSON">JSON</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* Normalize */}
      <div className="mb-4 flex items-center gap-2">
        <Switch
          id="normalize"
          checked={normalize}
          onCheckedChange={setNormalize}
          data-testid="register-normalize-toggle"
        />
        <Label htmlFor="normalize" className="text-sm">Normalize schema</Label>
      </div>

      {/* Schema Editor */}
      <div className="mb-4 space-y-2">
        <div className="flex items-center justify-between">
          <Label>Schema</Label>
          <div className="flex items-center gap-2">
            {isNewVersion && latestVersion && (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={handleStartFromLatest}
                data-testid="register-start-from-latest-btn"
              >
                Start from latest v{latestVersion.version}
              </Button>
            )}
          </div>
        </div>

        {/* Template selector — visible only when the editor is empty */}
        {editorIsEmpty && availableTemplates.length > 0 && (
          <div className="flex items-center gap-2" data-testid="register-template-section">
            <FileCode className="h-4 w-4 text-muted-foreground" />
            <span className="text-sm text-muted-foreground">Start from template:</span>
            <Select onValueChange={handleTemplateSelect} data-testid="register-template-select">
              <SelectTrigger className="w-52" data-testid="register-template-select">
                <SelectValue placeholder="Choose a template" />
              </SelectTrigger>
              <SelectContent>
                {availableTemplates.map((tpl) => (
                  <SelectItem key={tpl.value} value={tpl.value}>
                    {tpl.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}

        <SchemaEditor
          value={schema}
          onChange={setSchema}
          schemaType={schemaType}
          height="350px"
          data-testid="register-schema-editor"
        />
        {schema.trim() && (
          <div data-testid="register-validation-status">
            {isValidSyntax(schema, schemaType) ? (
              <Badge variant="outline" className="text-green-600">
                <CheckCircle className="mr-1 h-3 w-3" /> Valid {schemaType} schema
              </Badge>
            ) : (
              <Badge variant="outline" className="text-destructive">
                <XCircle className="mr-1 h-3 w-3" /> Invalid syntax
              </Badge>
            )}
          </div>
        )}
      </div>

      <Separator className="my-4" />

      {/* References */}
      <div className="mb-4" data-testid="register-references-section">
        <button
          type="button"
          className="flex items-center gap-1 text-sm font-medium"
          onClick={() => setShowReferences(!showReferences)}
        >
          {showReferences ? '▼' : '▶'} References ({references.length})
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
                  data-testid="register-reference-subject-input"
                />
                <Input
                  placeholder="Version"
                  value={ref.version}
                  onChange={(e) => updateReference(i, 'version', e.target.value)}
                  className="w-24"
                  data-testid="register-reference-version-input"
                />
                <Input
                  placeholder="Reference name"
                  value={ref.name}
                  onChange={(e) => updateReference(i, 'name', e.target.value)}
                  className="flex-1"
                  data-testid="register-reference-name-input"
                />
                <Button variant="ghost" size="icon" onClick={() => removeReference(i)}>
                  <X className="h-4 w-4" />
                </Button>
              </div>
            ))}
            <Button variant="outline" size="sm" onClick={addReference} data-testid="register-add-reference-btn">
              <Plus className="mr-1 h-4 w-4" /> Add Reference
            </Button>
          </div>
        )}
      </div>

      <Separator className="my-4" />

      {/* Actions */}
      <div className="flex gap-3">
        <Button
          variant="outline"
          onClick={() => compatMutation.mutate()}
          disabled={!canSubmit || compatMutation.isPending}
          data-testid="register-compat-check-btn"
        >
          {compatMutation.isPending && <Loader2 className="mr-1 h-4 w-4 animate-spin" />}
          Check Compatibility
        </Button>
        <Button
          onClick={() => registerMutation.mutate()}
          disabled={!canSubmit || registerMutation.isPending}
          data-testid="register-submit-btn"
        >
          {registerMutation.isPending && <Loader2 className="mr-1 h-4 w-4 animate-spin" />}
          Register
        </Button>
      </div>

      {/* Compatibility Result */}
      {compatResult?.checked && (
        <Card className="mt-4" data-testid="register-compat-result">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-sm font-medium">
              {compatResult.compatible ? (
                <>
                  <CheckCircle className="h-4 w-4 text-green-600" />
                  Compatible
                </>
              ) : (
                <>
                  <XCircle className="h-4 w-4 text-destructive" />
                  Incompatible
                </>
              )}
            </CardTitle>
          </CardHeader>
          {compatResult.messages.length > 0 && (
            <CardContent>
              <ul className="space-y-1 text-sm">
                {compatResult.messages.map((msg, i) => (
                  <li key={i} className="text-muted-foreground">{msg}</li>
                ))}
              </ul>
            </CardContent>
          )}
        </Card>
      )}
    </div>
  );
}

function isValidSyntax(schema: string, schemaType: SchemaType): boolean {
  if (schemaType === 'PROTOBUF') {
    // Basic check: non-empty and contains message/enum/syntax keyword
    return schema.trim().length > 0 && /\b(syntax|message|enum|service)\b/.test(schema);
  }
  try {
    JSON.parse(schema);
    return true;
  } catch {
    return false;
  }
}
