import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch, ApiClientError } from './client';

// ── Query Keys ──
export const queryKeys = {
  subjects: {
    all: ['subjects'] as const,
    list: (opts?: { deleted?: boolean }) =>
      ['subjects', 'list', opts] as const,
    detail: (subject: string) => ['subjects', subject] as const,
    versions: (subject: string) => ['subjects', subject, 'versions'] as const,
    version: (subject: string, version: number | string) =>
      ['subjects', subject, 'versions', version] as const,
    config: (subject: string) => ['subjects', subject, 'config'] as const,
    mode: (subject: string) => ['subjects', subject, 'mode'] as const,
  },
  schemas: {
    all: ['schemas'] as const,
    list: (opts?: { subjectPrefix?: string }) =>
      ['schemas', 'list', opts] as const,
    byId: (id: number) => ['schemas', id] as const,
    subjects: (id: number) => ['schemas', id, 'subjects'] as const,
    versions: (id: number) => ['schemas', id, 'versions'] as const,
  },
  config: {
    global: ['config', 'global'] as const,
  },
  mode: {
    global: ['mode', 'global'] as const,
  },
  metadata: {
    version: ['metadata', 'version'] as const,
    clusterId: ['metadata', 'clusterId'] as const,
    schemaTypes: ['metadata', 'schemaTypes'] as const,
  },
  auth: {
    config: ['auth', 'config'] as const,
  },
  admin: {
    users: ['admin', 'users'] as const,
    apikeys: ['admin', 'apikeys'] as const,
  },
} as const;

// ── Subjects ──

export function useSubjects(opts?: { deleted?: boolean }) {
  const params = new URLSearchParams();
  if (opts?.deleted) params.set('deleted', 'true');
  const qs = params.toString() ? `?${params.toString()}` : '';
  return useQuery({
    queryKey: queryKeys.subjects.list(opts),
    queryFn: () => apiFetch<string[]>(`/subjects${qs}`),
  });
}

export function useSubjectVersions(subject: string) {
  return useQuery({
    queryKey: queryKeys.subjects.versions(subject),
    queryFn: () => apiFetch<number[]>(
      `/subjects/${encodeURIComponent(subject)}/versions`
    ),
    enabled: !!subject,
  });
}

export interface SubjectVersion {
  subject: string;
  id: number;
  version: number;
  schemaType: string;
  schema: string;
  references?: Array<{ name: string; subject: string; version: number }>;
}

export function useSubjectVersion(subject: string, version: number | string) {
  return useQuery({
    queryKey: queryKeys.subjects.version(subject, version),
    queryFn: () => apiFetch<SubjectVersion>(
      `/subjects/${encodeURIComponent(subject)}/versions/${version}`
    ),
    enabled: !!subject && version !== undefined,
  });
}

export interface CompatibilityConfig {
  compatibilityLevel: string;
}

export function useSubjectConfig(subject: string) {
  return useQuery({
    queryKey: queryKeys.subjects.config(subject),
    queryFn: async () => {
      try {
        return await apiFetch<CompatibilityConfig>(
          `/config/${encodeURIComponent(subject)}`
        );
      } catch (e) {
        if (e instanceof ApiClientError && e.status === 404) {
          return null; // inherits global
        }
        throw e;
      }
    },
    enabled: !!subject,
  });
}

export interface ModeConfig {
  mode: string;
}

export function useSubjectMode(subject: string) {
  return useQuery({
    queryKey: queryKeys.subjects.mode(subject),
    queryFn: async () => {
      try {
        return await apiFetch<ModeConfig>(
          `/mode/${encodeURIComponent(subject)}`
        );
      } catch (e) {
        if (e instanceof ApiClientError && e.status === 404) {
          return null;
        }
        throw e;
      }
    },
    enabled: !!subject,
  });
}

// ── Schemas ──

export interface SchemaById {
  schema: string;
  schemaType: string;
  references?: Array<{ name: string; subject: string; version: number }>;
}

export function useSchemaById(id: number) {
  return useQuery({
    queryKey: queryKeys.schemas.byId(id),
    queryFn: () => apiFetch<SchemaById>(`/schemas/ids/${id}`),
    enabled: id > 0,
  });
}

export interface SchemaSubjectVersion {
  subject: string;
  version: number;
}

export function useSchemaSubjects(id: number) {
  return useQuery({
    queryKey: queryKeys.schemas.subjects(id),
    queryFn: () => apiFetch<SchemaSubjectVersion[]>(`/schemas/ids/${id}/subjects`),
    enabled: id > 0,
  });
}

export function useSchemaVersions(id: number) {
  return useQuery({
    queryKey: queryKeys.schemas.versions(id),
    queryFn: () => apiFetch<SchemaSubjectVersion[]>(`/schemas/ids/${id}/versions`),
    enabled: id > 0,
  });
}

export interface SchemaListItem {
  subject: string;
  version: number;
  id: number;
  schemaType: string;
  schema?: string;
  references?: Array<{ name: string; subject: string; version: number }>;
}

export function useSchemasList(opts?: { subjectPrefix?: string }) {
  const params = new URLSearchParams();
  if (opts?.subjectPrefix) params.set('subjectPrefix', opts.subjectPrefix);
  const qs = params.toString() ? `?${params.toString()}` : '';
  return useQuery({
    queryKey: queryKeys.schemas.list(opts),
    queryFn: () => apiFetch<SchemaListItem[]>(`/schemas${qs}`),
  });
}

export function useSchemaTypes() {
  return useQuery({
    queryKey: queryKeys.metadata.schemaTypes,
    queryFn: () => apiFetch<string[]>('/schemas/types'),
  });
}

// ── Referenced By ──

export function useReferencedBy(subject: string, version: number) {
  return useQuery({
    queryKey: ['referencedby', subject, version] as const,
    queryFn: () => apiFetch<number[]>(
      `/subjects/${encodeURIComponent(subject)}/versions/${version}/referencedby`
    ),
    enabled: !!subject && version > 0,
  });
}

// ── Config ──

export function useGlobalConfig() {
  return useQuery({
    queryKey: queryKeys.config.global,
    queryFn: () => apiFetch<CompatibilityConfig>('/config'),
  });
}

// ── Mode ──

export function useGlobalMode() {
  return useQuery({
    queryKey: queryKeys.mode.global,
    queryFn: () => apiFetch<ModeConfig>('/mode'),
  });
}

// ── Metadata ──

export interface ServerVersion {
  version: string;
  commit: string;
  build_time?: string;
}

export function useServerVersion() {
  return useQuery({
    queryKey: queryKeys.metadata.version,
    queryFn: () => apiFetch<ServerVersion>('/v1/metadata/version'),
    staleTime: Infinity,
  });
}

export interface ClusterId {
  id: string;
}

export function useClusterId() {
  return useQuery({
    queryKey: queryKeys.metadata.clusterId,
    queryFn: () => apiFetch<ClusterId>('/v1/metadata/id'),
    staleTime: Infinity,
  });
}

// ── Mutations ──

export function useDeleteSubject(subject: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (opts?: { permanent?: boolean }) =>
      apiFetch<number[]>(
        `/subjects/${encodeURIComponent(subject)}${opts?.permanent ? '?permanent=true' : ''}`,
        { method: 'DELETE' }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.subjects.all });
    },
  });
}

export function useDeleteVersion(subject: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ version, permanent }: { version: number; permanent?: boolean }) =>
      apiFetch<number>(
        `/subjects/${encodeURIComponent(subject)}/versions/${version}${permanent ? '?permanent=true' : ''}`,
        { method: 'DELETE' }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.subjects.versions(subject) });
    },
  });
}

// ── Config Mutations ──

export function useSetGlobalConfig() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (compatibility: string) =>
      apiFetch<CompatibilityConfig>('/config', {
        method: 'PUT',
        body: JSON.stringify({ compatibility }),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.config.global });
    },
  });
}

export function useSetSubjectConfig() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ subject, compatibility }: { subject: string; compatibility: string }) =>
      apiFetch<CompatibilityConfig>(
        `/config/${encodeURIComponent(subject)}`,
        { method: 'PUT', body: JSON.stringify({ compatibility }) }
      ),
    onSuccess: (_data, { subject }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.subjects.config(subject) });
    },
  });
}

export function useDeleteSubjectConfig() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (subject: string) =>
      apiFetch<CompatibilityConfig>(
        `/config/${encodeURIComponent(subject)}`,
        { method: 'DELETE' }
      ),
    onSuccess: (_data, subject) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.subjects.config(subject) });
    },
  });
}

// ── Mode Mutations ──

export function useSetGlobalMode() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (mode: string) =>
      apiFetch<ModeConfig>('/mode', {
        method: 'PUT',
        body: JSON.stringify({ mode }),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.mode.global });
    },
  });
}

export function useSetSubjectMode() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ subject, mode }: { subject: string; mode: string }) =>
      apiFetch<ModeConfig>(
        `/mode/${encodeURIComponent(subject)}`,
        { method: 'PUT', body: JSON.stringify({ mode }) }
      ),
    onSuccess: (_data, { subject }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.subjects.mode(subject) });
    },
  });
}

export function useDeleteSubjectMode() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (subject: string) =>
      apiFetch<ModeConfig>(
        `/mode/${encodeURIComponent(subject)}`,
        { method: 'DELETE' }
      ),
    onSuccess: (_data, subject) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.subjects.mode(subject) });
    },
  });
}

// ── Import ──

export interface ImportResult {
  id?: number;
  subject?: string;
  version?: number;
  error?: string;
}

export function useImportSchema() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: {
      schema: string;
      schemaType: string;
      subject: string;
      id: number;
      version: number;
      references?: Array<{ name: string; subject: string; version: number }>;
    }) =>
      apiFetch<{ id: number }>('/subjects/' + encodeURIComponent(body.subject) + '/versions', {
        method: 'POST',
        body: JSON.stringify(body),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.subjects.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.schemas.all });
    },
  });
}

// ── Admin: Users ──

export interface User {
  id: number;
  username: string;
  email: string;
  role: 'super_admin' | 'admin' | 'developer' | 'readonly';
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateUserRequest {
  username: string;
  password: string;
  email: string;
  role: 'admin' | 'developer' | 'readonly';
  enabled?: boolean;
}

export interface UpdateUserRequest {
  password?: string;
  email?: string;
  role?: string;
  enabled?: boolean;
}

export function useUsers() {
  return useQuery({
    queryKey: queryKeys.admin.users,
    queryFn: () => apiFetch<User[]>('/admin/users'),
  });
}

export function useCreateUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateUserRequest) =>
      apiFetch<User>('/admin/users', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.admin.users });
    },
  });
}

export function useUpdateUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...data }: UpdateUserRequest & { id: number }) =>
      apiFetch<User>(`/admin/users/${id}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.admin.users });
    },
  });
}

export function useDeleteUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: number) =>
      apiFetch<void>(`/admin/users/${id}`, { method: 'DELETE' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.admin.users });
    },
  });
}

// ── Admin: API Keys ──

export interface ApiKey {
  id: number;
  key_prefix: string;
  name: string;
  role: 'admin' | 'developer' | 'readonly';
  username: string;
  created_at: string;
  expires_at: string | null;
  is_active: boolean;
  revoked_at: string | null;
}

export interface CreateApiKeyRequest {
  name: string;
  role: 'admin' | 'developer' | 'readonly';
  expires_in?: number;
}

export interface CreateApiKeyResponse extends ApiKey {
  key: string;
}

export function useApiKeys() {
  return useQuery({
    queryKey: queryKeys.admin.apikeys,
    queryFn: () => apiFetch<ApiKey[]>('/admin/apikeys'),
  });
}

export function useCreateApiKey() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateApiKeyRequest) =>
      apiFetch<CreateApiKeyResponse>('/admin/apikeys', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.admin.apikeys });
    },
  });
}

export function useRevokeApiKey() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: number) =>
      apiFetch<ApiKey>(`/admin/apikeys/${id}/revoke`, { method: 'POST' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.admin.apikeys });
    },
  });
}

export function useRotateApiKey() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: number) =>
      apiFetch<CreateApiKeyResponse>(`/admin/apikeys/${id}/rotate`, { method: 'POST' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.admin.apikeys });
    },
  });
}

export function useDeleteApiKey() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: number) =>
      apiFetch<void>(`/admin/apikeys/${id}`, { method: 'DELETE' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.admin.apikeys });
    },
  });
}

// ── Self-Service ──

export function useChangePassword() {
  return useMutation({
    mutationFn: (data: { current_password: string; new_password: string }) =>
      apiFetch<void>('/admin/account/password', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
  });
}

// ── Delete Global Config / Mode ──

export function useDeleteGlobalConfig() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiFetch<CompatibilityConfig>('/config', { method: 'DELETE' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.config.global });
    },
  });
}

export function useDeleteGlobalMode() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiFetch<ModeConfig>('/mode', { method: 'DELETE' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.mode.global });
    },
  });
}

// ── Compatibility Check ──

export interface CompatibilityCheckResult {
  is_compatible: boolean;
  messages?: string[];
}

export function useCheckCompatibility() {
  return useMutation({
    mutationFn: ({
      subject,
      version,
      schema,
      schemaType,
      references,
    }: {
      subject: string;
      version?: number | string;
      schema: string;
      schemaType: string;
      references?: Array<{ name: string; subject: string; version: number }>;
    }) => {
      const versionPath = version ? `/${version}` : '';
      return apiFetch<CompatibilityCheckResult>(
        `/compatibility/subjects/${encodeURIComponent(subject)}/versions${versionPath}?verbose=true`,
        {
          method: 'POST',
          body: JSON.stringify({ schema, schemaType, references }),
        }
      );
    },
  });
}

// ── Schema Lookup ──

export function useSchemaLookup() {
  return useMutation({
    mutationFn: ({
      subject,
      schema,
      schemaType,
      references,
    }: {
      subject: string;
      schema: string;
      schemaType: string;
      references?: Array<{ name: string; subject: string; version: number }>;
    }) =>
      apiFetch<SubjectVersion>(
        `/subjects/${encodeURIComponent(subject)}`,
        {
          method: 'POST',
          body: JSON.stringify({ schema, schemaType, references }),
        }
      ),
  });
}

// ── Roles ──

export interface RoleInfo {
  name: string;
  description?: string;
  permissions?: string[];
}

export function useRoles() {
  return useQuery({
    queryKey: ['admin', 'roles'] as const,
    queryFn: () => apiFetch<RoleInfo[]>('/admin/roles'),
    staleTime: Infinity,
  });
}

// ── Health ──

export interface HealthStatus {
  status: 'UP' | 'DOWN';
  reason?: string;
}

export function useHealthLive() {
  return useQuery({
    queryKey: ['health', 'live'] as const,
    queryFn: () => apiFetch<HealthStatus>('/health/live'),
    refetchInterval: 30_000,
  });
}

export function useHealthReady() {
  return useQuery({
    queryKey: ['health', 'ready'] as const,
    queryFn: () => apiFetch<HealthStatus>('/health/ready'),
    refetchInterval: 30_000,
  });
}

export function useHealthStartup() {
  return useQuery({
    queryKey: ['health', 'startup'] as const,
    queryFn: () => apiFetch<HealthStatus>('/health/startup'),
    refetchInterval: 30_000,
  });
}

// ── Contexts ──

export function useContexts() {
  return useQuery({
    queryKey: ['contexts'] as const,
    queryFn: () => apiFetch<string[]>('/contexts'),
  });
}

// ── Exporters ──

export interface ExporterResponse {
  name: string;
  contextType?: string;
  context?: string;
  subjects?: string[];
  subjectRenameFormat?: string;
  config?: Record<string, string>;
}

export interface ExporterStatusResponse {
  name: string;
  state: string;
  offset?: number;
  ts?: number;
  trace?: string;
}

export interface CreateExporterRequest {
  name: string;
  contextType?: string;
  context?: string;
  subjects?: string[];
  subjectRenameFormat?: string;
  config?: Record<string, string>;
}

export function useExporters() {
  return useQuery({
    queryKey: ['exporters'] as const,
    queryFn: () => apiFetch<string[]>('/exporters'),
  });
}

export function useExporter(name: string) {
  return useQuery({
    queryKey: ['exporters', name] as const,
    queryFn: () => apiFetch<ExporterResponse>(`/exporters/${encodeURIComponent(name)}`),
    enabled: !!name,
  });
}

export function useExporterStatus(name: string) {
  return useQuery({
    queryKey: ['exporters', name, 'status'] as const,
    queryFn: () => apiFetch<ExporterStatusResponse>(`/exporters/${encodeURIComponent(name)}/status`),
    enabled: !!name,
    refetchInterval: 10_000,
  });
}

export function useExporterConfig(name: string) {
  return useQuery({
    queryKey: ['exporters', name, 'config'] as const,
    queryFn: () => apiFetch<Record<string, string>>(`/exporters/${encodeURIComponent(name)}/config`),
    enabled: !!name,
  });
}

export function useCreateExporter() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateExporterRequest) =>
      apiFetch<ExporterResponse>('/exporters', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['exporters'] });
    },
  });
}

export function useUpdateExporter() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ name, ...data }: CreateExporterRequest) =>
      apiFetch<ExporterResponse>(`/exporters/${encodeURIComponent(name)}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['exporters'] });
    },
  });
}

export function useDeleteExporter() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (name: string) =>
      apiFetch<void>(`/exporters/${encodeURIComponent(name)}`, { method: 'DELETE' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['exporters'] });
    },
  });
}

export function usePauseExporter() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (name: string) =>
      apiFetch<void>(`/exporters/${encodeURIComponent(name)}/pause`, { method: 'PUT' }),
    onSuccess: (_d, name) => {
      queryClient.invalidateQueries({ queryKey: ['exporters', name, 'status'] });
    },
  });
}

export function useResumeExporter() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (name: string) =>
      apiFetch<void>(`/exporters/${encodeURIComponent(name)}/resume`, { method: 'PUT' }),
    onSuccess: (_d, name) => {
      queryClient.invalidateQueries({ queryKey: ['exporters', name, 'status'] });
    },
  });
}

export function useResetExporter() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (name: string) =>
      apiFetch<void>(`/exporters/${encodeURIComponent(name)}/reset`, { method: 'PUT' }),
    onSuccess: (_d, name) => {
      queryClient.invalidateQueries({ queryKey: ['exporters', name, 'status'] });
    },
  });
}

export function useUpdateExporterConfig() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ name, config }: { name: string; config: Record<string, string> }) =>
      apiFetch<void>(`/exporters/${encodeURIComponent(name)}/config`, {
        method: 'PUT',
        body: JSON.stringify(config),
      }),
    onSuccess: (_d, { name }) => {
      queryClient.invalidateQueries({ queryKey: ['exporters', name, 'config'] });
    },
  });
}

// ── DEK Registry: KEKs ──

export interface KEKResponse {
  name: string;
  kmsType: string;
  kmsKeyId: string;
  kmsProps?: Record<string, string>;
  doc?: string;
  shared: boolean;
  ts?: number;
  deleted?: boolean;
}

export interface CreateKEKRequest {
  name: string;
  kmsType: string;
  kmsKeyId: string;
  kmsProps?: Record<string, string>;
  doc?: string;
  shared: boolean;
}

export function useKEKs(opts?: { deleted?: boolean }) {
  const params = new URLSearchParams();
  if (opts?.deleted) params.set('deleted', 'true');
  const qs = params.toString() ? `?${params.toString()}` : '';
  return useQuery({
    queryKey: ['dek', 'keks', opts] as const,
    queryFn: () => apiFetch<string[]>(`/dek-registry/v1/keks${qs}`),
  });
}

export function useKEK(name: string) {
  return useQuery({
    queryKey: ['dek', 'keks', name] as const,
    queryFn: () => apiFetch<KEKResponse>(`/dek-registry/v1/keks/${encodeURIComponent(name)}`),
    enabled: !!name,
  });
}

export function useCreateKEK() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateKEKRequest) =>
      apiFetch<KEKResponse>('/dek-registry/v1/keks', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dek', 'keks'] });
    },
  });
}

export function useUpdateKEK() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ name, ...data }: Partial<CreateKEKRequest> & { name: string }) =>
      apiFetch<KEKResponse>(`/dek-registry/v1/keks/${encodeURIComponent(name)}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dek', 'keks'] });
    },
  });
}

export function useDeleteKEK() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ name, permanent }: { name: string; permanent?: boolean }) =>
      apiFetch<void>(
        `/dek-registry/v1/keks/${encodeURIComponent(name)}${permanent ? '?permanent=true' : ''}`,
        { method: 'DELETE' }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dek', 'keks'] });
    },
  });
}

export function useUndeleteKEK() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (name: string) =>
      apiFetch<void>(`/dek-registry/v1/keks/${encodeURIComponent(name)}/undelete`, {
        method: 'POST',
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dek', 'keks'] });
    },
  });
}

export function useTestKEK() {
  return useMutation({
    mutationFn: (name: string) =>
      apiFetch<void>(`/dek-registry/v1/keks/${encodeURIComponent(name)}/test`, {
        method: 'POST',
      }),
  });
}

// ── DEK Registry: DEKs ──

export interface DEKResponse {
  kekName: string;
  subject: string;
  version: number;
  algorithm: string;
  encryptedKeyMaterial?: string;
  keyMaterial?: string;
  ts?: number;
  deleted?: boolean;
}

export interface CreateDEKRequest {
  subject: string;
  version?: number;
  algorithm?: string;
  encryptedKeyMaterial?: string;
}

export function useDEKs(kekName: string) {
  return useQuery({
    queryKey: ['dek', 'keks', kekName, 'deks'] as const,
    queryFn: () => apiFetch<string[]>(`/dek-registry/v1/keks/${encodeURIComponent(kekName)}/deks`),
    enabled: !!kekName,
  });
}

export function useDEK(kekName: string, subject: string) {
  return useQuery({
    queryKey: ['dek', 'keks', kekName, 'deks', subject] as const,
    queryFn: () =>
      apiFetch<DEKResponse>(
        `/dek-registry/v1/keks/${encodeURIComponent(kekName)}/deks/${encodeURIComponent(subject)}`
      ),
    enabled: !!kekName && !!subject,
  });
}

export function useDEKVersions(kekName: string, subject: string) {
  return useQuery({
    queryKey: ['dek', 'keks', kekName, 'deks', subject, 'versions'] as const,
    queryFn: () =>
      apiFetch<number[]>(
        `/dek-registry/v1/keks/${encodeURIComponent(kekName)}/deks/${encodeURIComponent(subject)}/versions`
      ),
    enabled: !!kekName && !!subject,
  });
}

export function useCreateDEK() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ kekName, ...data }: CreateDEKRequest & { kekName: string }) =>
      apiFetch<DEKResponse>(`/dek-registry/v1/keks/${encodeURIComponent(kekName)}/deks`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dek'] });
    },
  });
}

export function useDeleteDEK() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ kekName, subject, permanent }: { kekName: string; subject: string; permanent?: boolean }) =>
      apiFetch<void>(
        `/dek-registry/v1/keks/${encodeURIComponent(kekName)}/deks/${encodeURIComponent(subject)}${permanent ? '?permanent=true' : ''}`,
        { method: 'DELETE' }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dek'] });
    },
  });
}

export function useUndeleteDEK() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ kekName, subject }: { kekName: string; subject: string }) =>
      apiFetch<void>(
        `/dek-registry/v1/keks/${encodeURIComponent(kekName)}/deks/${encodeURIComponent(subject)}/undelete`,
        { method: 'POST' }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dek'] });
    },
  });
}
