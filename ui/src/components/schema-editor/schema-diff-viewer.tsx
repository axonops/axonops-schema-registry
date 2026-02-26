import { useCallback } from 'react';
import { DiffEditor, type DiffOnMount } from '@monaco-editor/react';
import type * as monacoType from 'monaco-editor';
import { useTheme } from '@/context/theme-context';
import { getMonacoLanguage, registerProtobufLanguage, type SchemaType } from './monaco-config';

interface SchemaDiffViewerProps {
  original: string;
  modified: string;
  schemaType: SchemaType;
  height?: string;
  'data-testid'?: string;
}

export function SchemaDiffViewer({
  original,
  modified,
  schemaType,
  height = '400px',
  'data-testid': testId,
}: SchemaDiffViewerProps) {
  const { resolvedTheme } = useTheme();

  const handleMount: DiffOnMount = useCallback((_editor: monacoType.editor.IStandaloneDiffEditor, monaco: typeof monacoType) => {
    registerProtobufLanguage(monaco);
  }, []);

  const language = getMonacoLanguage(schemaType);

  return (
    <div data-testid={testId} className="rounded-md border overflow-hidden">
      <DiffEditor
        height={height}
        language={language}
        original={original}
        modified={modified}
        onMount={handleMount}
        theme={resolvedTheme === 'dark' ? 'vs-dark' : 'vs'}
        options={{
          readOnly: true,
          minimap: { enabled: false },
          scrollBeyondLastLine: false,
          automaticLayout: true,
          fontSize: 13,
          fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', Menlo, Monaco, monospace",
          renderSideBySide: true,
        }}
      />
    </div>
  );
}
