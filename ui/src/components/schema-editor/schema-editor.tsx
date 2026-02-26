import { useRef, useCallback } from 'react';
import Editor, { type OnMount, type OnChange } from '@monaco-editor/react';
import type * as monacoType from 'monaco-editor';
import { useTheme } from '@/context/theme-context';
import {
  getMonacoLanguage,
  getMonacoOptions,
  registerProtobufLanguage,
  type SchemaType,
} from './monaco-config';

interface SchemaEditorProps {
  value: string;
  onChange?: (value: string) => void;
  schemaType: SchemaType;
  readOnly?: boolean;
  height?: string;
  'data-testid'?: string;
}

export function SchemaEditor({
  value,
  onChange,
  schemaType,
  readOnly = false,
  height = '400px',
  'data-testid': testId,
}: SchemaEditorProps) {
  const editorRef = useRef<monacoType.editor.IStandaloneCodeEditor | null>(null);
  const { resolvedTheme } = useTheme();

  const handleMount: OnMount = useCallback((editor, monaco) => {
    editorRef.current = editor;
    registerProtobufLanguage(monaco);
  }, []);

  const handleChange: OnChange = useCallback((val) => {
    onChange?.(val ?? '');
  }, [onChange]);

  const language = getMonacoLanguage(schemaType);
  const options = getMonacoOptions(readOnly);

  return (
    <div data-testid={testId} className="rounded-md border overflow-hidden">
      <Editor
        height={height}
        language={language}
        value={value}
        onChange={handleChange}
        onMount={handleMount}
        theme={resolvedTheme === 'dark' ? 'vs-dark' : 'vs'}
        options={options}
      />
    </div>
  );
}
