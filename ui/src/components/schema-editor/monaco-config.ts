import type * as monacoType from 'monaco-editor';

export type SchemaType = 'AVRO' | 'PROTOBUF' | 'JSON';

export function getMonacoLanguage(schemaType: SchemaType): string {
  switch (schemaType) {
    case 'AVRO':     return 'json';
    case 'JSON':     return 'json';
    case 'PROTOBUF': return 'protobuf';
  }
}

export function getMonacoOptions(readonly: boolean): monacoType.editor.IStandaloneEditorConstructionOptions {
  return {
    readOnly: readonly,
    minimap: { enabled: false },
    lineNumbers: 'on',
    scrollBeyondLastLine: false,
    wordWrap: 'on',
    wrappingIndent: 'indent',
    automaticLayout: true,
    tabSize: 2,
    formatOnPaste: true,
    formatOnType: true,
    renderWhitespace: 'selection',
    fontSize: 13,
    fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', Menlo, Monaco, monospace",
    scrollbar: {
      verticalScrollbarSize: 8,
      horizontalScrollbarSize: 8,
    },
  };
}

export function getFileExtension(schemaType: SchemaType): string {
  switch (schemaType) {
    case 'AVRO':     return '.avsc';
    case 'JSON':     return '.json';
    case 'PROTOBUF': return '.proto';
  }
}

export function getDownloadFilename(subject: string, version: number, schemaType: SchemaType): string {
  return `${subject}-v${version}${getFileExtension(schemaType)}`;
}

export function registerProtobufLanguage(monaco: typeof monacoType) {
  // Only register once
  const languages = monaco.languages.getLanguages();
  if (languages.some((l) => l.id === 'protobuf')) return;

  monaco.languages.register({ id: 'protobuf' });
  monaco.languages.setMonarchTokensProvider('protobuf', {
    keywords: [
      'syntax', 'package', 'import', 'option', 'message', 'enum', 'service',
      'rpc', 'returns', 'oneof', 'map', 'reserved', 'repeated', 'optional',
      'required', 'extend', 'extensions', 'to', 'max', 'true', 'false',
      'public', 'weak', 'stream',
    ],
    typeKeywords: [
      'double', 'float', 'int32', 'int64', 'uint32', 'uint64', 'sint32',
      'sint64', 'fixed32', 'fixed64', 'sfixed32', 'sfixed64', 'bool',
      'string', 'bytes',
    ],
    tokenizer: {
      root: [
        [/\/\/.*$/, 'comment'],
        [/\/\*/, 'comment', '@comment'],
        [/"([^"\\]|\\.)*$/, 'string.invalid'],
        [/"/, 'string', '@string'],
        [/[a-zA-Z_]\w*/, {
          cases: {
            '@keywords': 'keyword',
            '@typeKeywords': 'type',
            '@default': 'identifier'
          }
        }],
        [/[{}()\[\]]/, '@brackets'],
        [/[0-9]+/, 'number'],
        [/[;,.]/, 'delimiter'],
        [/=/, 'operator'],
      ],
      comment: [
        [/[^/*]+/, 'comment'],
        [/\*\//, 'comment', '@pop'],
        [/[/*]/, 'comment'],
      ],
      string: [
        [/[^\\"]+/, 'string'],
        [/\\./, 'string.escape'],
        [/"/, 'string', '@pop'],
      ],
    },
  } as monacoType.languages.IMonarchLanguage);
}
