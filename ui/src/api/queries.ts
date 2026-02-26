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
