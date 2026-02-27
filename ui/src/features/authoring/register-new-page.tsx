import { useState, useCallback, useMemo } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch, ApiClientError } from '@/api/client';
import { useSubjects, useSchemaTypes, queryKeys } from '@/api/queries';
import type { SchemaMetadata } from '@/api/queries';
import { PageBreadcrumbs } from '@/components/shared/breadcrumbs';
import { SchemaEditor } from '@/components/schema-editor/schema-editor';
import type { SchemaType } from '@/components/schema-editor/monaco-config';
import { KeyValueEditor } from '@/components/shared/key-value-editor';
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
import {
  CheckCircle,
  XCircle,
  Plus,
  X,
  Loader2,
  FileCode,
  Tags,
  Settings2,
  ShieldAlert,
} from 'lucide-react';

// ---------------------------------------------------------------------------
// Schema templates (same set used in register-schema-page)
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
// Types
// ---------------------------------------------------------------------------

interface Reference {
  name: string;
  subject: string;
  version: string;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function RegisterNewPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  // Form state
  const [subject, setSubject] = useState('');
  const [schemaType, setSchemaType] = useState<SchemaType>('AVRO');
  const [schema, setSchema] = useState('');
  const [normalize, setNormalize] = useState(false);
  const [references, setReferences] = useState<Reference[]>([]);
  const [showReferences, setShowReferences] = useState(false);

  // Metadata state
  const [showMetadata, setShowMetadata] = useState(false);
  const [tags, setTags] = useState<Record<string, string>>({});
  const [properties, setProperties] = useState<Record<string, string>>({});
  const [sensitiveFields, setSensitiveFields] = useState('');

  // Compat check state
  const [compatResult, setCompatResult] = useState<{
    checked: boolean;
    compatible: boolean;
    messages: string[];
  } | null>(null);

  // Fetch existing subjects for autocomplete
  const { data: existingSubjects } = useSubjects();

  // Fetch supported schema types
  const { data: schemaTypes } = useSchemaTypes();
  const availableSchemaTypes = useMemo(
    () => (schemaTypes ?? ['AVRO', 'PROTOBUF', 'JSON']) as SchemaType[],
    [schemaTypes]
  );

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

  // Build metadata payload
  const buildMetadata = useCallback((): SchemaMetadata | undefined => {
    const hasTags = Object.keys(tags).length > 0;
    const hasProperties = Object.keys(properties).length > 0;
    const hasSensitive = sensitiveFields.trim().length > 0;

    if (!hasTags && !hasProperties && !hasSensitive) return undefined;

    const metadata: SchemaMetadata = {};

    if (hasTags) {
      // Convert comma-separated string values to arrays
      const tagMap: Record<string, string[]> = {};
      for (const [key, value] of Object.entries(tags)) {
        tagMap[key] = value
          .split(',')
          .map((v) => v.trim())
          .filter(Boolean);
      }
      metadata.tags = tagMap;
    }

    if (hasProperties) {
      metadata.properties = { ...properties };
    }

    if (hasSensitive) {
      metadata.sensitive = sensitiveFields
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean);
    }

    return metadata;
  }, [tags, properties, sensitiveFields]);

  // Build references payload
  const buildReferences = useCallback(() => {
    return references
      .filter((r) => r.subject && r.name)
      .map((r) => ({
        name: r.name,
        subject: r.subject,
        version: r.version === 'latest' ? -1 : parseInt(r.version, 10),
      }));
  }, [references]);

  // Compatibility check
  const compatMutation = useMutation({
    mutationFn: async () => {
      const schemaStr =
        schemaType === 'PROTOBUF' ? schema : JSON.stringify(JSON.parse(schema));
      return apiFetch<{ is_compatible: boolean; messages?: string[] }>(
        `/compatibility/subjects/${encodeURIComponent(subject)}/versions?verbose=true`,
        {
          method: 'POST',
          body: JSON.stringify({
            schema: schemaStr,
            schemaType,
            references: buildReferences(),
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
      const schemaStr =
        schemaType === 'PROTOBUF' ? schema : JSON.stringify(JSON.parse(schema));
      const qs = normalize ? '?normalize=true' : '';

      const body: Record<string, unknown> = {
        schema: schemaStr,
        schemaType,
        references: buildReferences(),
      };

      const metadata = buildMetadata();
      if (metadata) {
        body.metadata = metadata;
      }

      return apiFetch<{ id: number }>(
        `/subjects/${encodeURIComponent(subject)}/versions${qs}`,
        {
          method: 'POST',
          body: JSON.stringify(body),
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

  // Reference helpers
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

  const breadcrumbs = [{ label: 'Register Schema' }];
  const canSubmit = subject.trim() && schema.trim() && !registerMutation.isPending;
  const editorIsEmpty = !schema.trim();

  // Check whether metadata has any content
  const metadataCount =
    Object.keys(tags).length +
    Object.keys(properties).length +
    (sensitiveFields.trim() ? 1 : 0);

  return (
    <div data-testid="register-new-page">
      <PageBreadcrumbs items={breadcrumbs} />

      <h1 className="mb-6 text-2xl font-bold">Register Schema</h1>

      {/* Subject */}
      <Card className="mb-4">
        <CardHeader>
          <CardTitle className="text-sm font-medium">Subject</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            <Label htmlFor="register-new-subject">Subject Name</Label>
            <Input
              id="register-new-subject"
              list="register-new-subject-suggestions"
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
              placeholder="e.g., orders-value"
              data-testid="register-new-subject-input"
            />
            {existingSubjects && existingSubjects.length > 0 && (
              <datalist id="register-new-subject-suggestions">
                {existingSubjects.map((s) => (
                  <option key={s} value={s} />
                ))}
              </datalist>
            )}
            <p className="text-xs text-muted-foreground">
              Enter a new subject name or select an existing one to register a new
              version.
            </p>
          </div>
        </CardContent>
      </Card>

      {/* Schema Type */}
      <Card className="mb-4">
        <CardHeader>
          <CardTitle className="text-sm font-medium">Schema Type</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label>Type</Label>
            <Select
              value={schemaType}
              onValueChange={(v) => setSchemaType(v as SchemaType)}
            >
              <SelectTrigger data-testid="register-new-type-select">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {availableSchemaTypes.map((t) => (
                  <SelectItem key={t} value={t}>
                    {t}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Normalize */}
          <div className="flex items-center gap-2">
            <Switch
              id="register-new-normalize"
              checked={normalize}
              onCheckedChange={setNormalize}
              data-testid="register-new-normalize-toggle"
            />
            <Label htmlFor="register-new-normalize" className="text-sm">
              Normalize schema
            </Label>
          </div>
        </CardContent>
      </Card>

      {/* Schema Editor */}
      <Card className="mb-4">
        <CardHeader>
          <CardTitle className="text-sm font-medium">Schema</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          {/* Template selector -- visible only when editor is empty */}
          {editorIsEmpty && availableTemplates.length > 0 && (
            <div
              className="flex items-center gap-2"
              data-testid="register-new-template-section"
            >
              <FileCode className="h-4 w-4 text-muted-foreground" />
              <span className="text-sm text-muted-foreground">
                Start from template:
              </span>
              <Select
                onValueChange={handleTemplateSelect}
                data-testid="register-new-template-select"
              >
                <SelectTrigger className="w-52" data-testid="register-new-template-select">
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
            data-testid="register-new-schema-editor"
          />

          {schema.trim() && (
            <div data-testid="register-new-validation-status">
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
        </CardContent>
      </Card>

      {/* References */}
      <Card className="mb-4">
        <CardContent className="pt-4">
          <div data-testid="register-new-references-section">
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
                      data-testid="register-new-reference-subject-input"
                    />
                    <Input
                      placeholder="Version"
                      value={ref.version}
                      onChange={(e) => updateReference(i, 'version', e.target.value)}
                      className="w-24"
                      data-testid="register-new-reference-version-input"
                    />
                    <Input
                      placeholder="Reference name"
                      value={ref.name}
                      onChange={(e) => updateReference(i, 'name', e.target.value)}
                      className="flex-1"
                      data-testid="register-new-reference-name-input"
                    />
                    <Button variant="ghost" size="icon" onClick={() => removeReference(i)}>
                      <X className="h-4 w-4" />
                    </Button>
                  </div>
                ))}
                <Button
                  variant="outline"
                  size="sm"
                  onClick={addReference}
                  data-testid="register-new-add-reference-btn"
                >
                  <Plus className="mr-1 h-4 w-4" /> Add Reference
                </Button>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Metadata */}
      <Card className="mb-4">
        <CardContent className="pt-4">
          <div data-testid="register-new-metadata-section">
            <button
              type="button"
              className="flex items-center gap-1 text-sm font-medium"
              onClick={() => setShowMetadata(!showMetadata)}
              data-testid="register-new-metadata-toggle"
            >
              {showMetadata ? '▼' : '▶'} Metadata
              {metadataCount > 0 && (
                <Badge variant="secondary" className="ml-1 text-xs">
                  {metadataCount}
                </Badge>
              )}
            </button>
            {showMetadata && (
              <div className="mt-4 space-y-6">
                {/* Tags */}
                <div data-testid="register-new-metadata-tags">
                  <div className="mb-2 flex items-center gap-2">
                    <Tags className="h-4 w-4 text-muted-foreground" />
                    <Label className="text-sm font-medium">Tags</Label>
                  </div>
                  <p className="mb-2 text-xs text-muted-foreground">
                    Key-value pairs where values can be comma-separated lists. For
                    example, key: "team" value: "payments,platform".
                  </p>
                  <KeyValueEditor
                    value={tags}
                    onChange={setTags}
                    keyPlaceholder="Tag name"
                    valuePlaceholder="Values (comma-separated)"
                  />
                </div>

                <Separator />

                {/* Properties */}
                <div data-testid="register-new-metadata-properties">
                  <div className="mb-2 flex items-center gap-2">
                    <Settings2 className="h-4 w-4 text-muted-foreground" />
                    <Label className="text-sm font-medium">Properties</Label>
                  </div>
                  <p className="mb-2 text-xs text-muted-foreground">
                    Arbitrary key-value string pairs for custom metadata.
                  </p>
                  <KeyValueEditor
                    value={properties}
                    onChange={setProperties}
                    keyPlaceholder="Property name"
                    valuePlaceholder="Property value"
                  />
                </div>

                <Separator />

                {/* Sensitive Fields */}
                <div data-testid="register-new-metadata-sensitive">
                  <div className="mb-2 flex items-center gap-2">
                    <ShieldAlert className="h-4 w-4 text-muted-foreground" />
                    <Label htmlFor="register-new-sensitive-fields" className="text-sm font-medium">
                      Sensitive Fields
                    </Label>
                  </div>
                  <p className="mb-2 text-xs text-muted-foreground">
                    Comma-separated list of field names that contain PII or sensitive
                    data (e.g., "email, ssn, phone_number").
                  </p>
                  <Input
                    id="register-new-sensitive-fields"
                    value={sensitiveFields}
                    onChange={(e) => setSensitiveFields(e.target.value)}
                    placeholder="e.g., email, ssn, phone_number"
                    data-testid="register-new-sensitive-input"
                  />
                </div>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      <Separator className="my-4" />

      {/* Actions */}
      <div className="flex gap-3">
        <Button
          variant="outline"
          onClick={() => compatMutation.mutate()}
          disabled={!canSubmit || compatMutation.isPending}
          data-testid="register-new-compat-check-btn"
        >
          {compatMutation.isPending && (
            <Loader2 className="mr-1 h-4 w-4 animate-spin" />
          )}
          Check Compatibility
        </Button>
        <Button
          onClick={() => registerMutation.mutate()}
          disabled={!canSubmit || registerMutation.isPending}
          data-testid="register-new-submit-btn"
        >
          {registerMutation.isPending && (
            <Loader2 className="mr-1 h-4 w-4 animate-spin" />
          )}
          Register
        </Button>
      </div>

      {/* Compatibility Result */}
      {compatResult?.checked && (
        <Card className="mt-4" data-testid="register-new-compat-result">
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
                  <li key={i} className="text-muted-foreground">
                    {msg}
                  </li>
                ))}
              </ul>
            </CardContent>
          )}
        </Card>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function isValidSyntax(schema: string, schemaType: SchemaType): boolean {
  if (schemaType === 'PROTOBUF') {
    return (
      schema.trim().length > 0 && /\b(syntax|message|enum|service)\b/.test(schema)
    );
  }
  try {
    JSON.parse(schema);
    return true;
  } catch {
    return false;
  }
}
