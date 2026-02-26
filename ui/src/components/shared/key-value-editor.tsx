import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Plus, Trash2 } from 'lucide-react';

interface KeyValueEditorProps {
  value: Record<string, string>;
  onChange: (value: Record<string, string>) => void;
  keyPlaceholder?: string;
  valuePlaceholder?: string;
  readOnly?: boolean;
}

export function KeyValueEditor({
  value,
  onChange,
  keyPlaceholder = 'Key',
  valuePlaceholder = 'Value',
  readOnly = false,
}: KeyValueEditorProps) {
  const [newKey, setNewKey] = useState('');
  const [newValue, setNewValue] = useState('');

  const entries = Object.entries(value);

  const handleAdd = () => {
    const trimmedKey = newKey.trim();
    if (!trimmedKey) return;
    onChange({ ...value, [trimmedKey]: newValue });
    setNewKey('');
    setNewValue('');
  };

  const handleRemove = (key: string) => {
    const next = { ...value };
    delete next[key];
    onChange(next);
  };

  const handleKeyChange = (oldKey: string, newKeyName: string) => {
    const next: Record<string, string> = {};
    for (const [k, v] of Object.entries(value)) {
      if (k === oldKey) {
        next[newKeyName] = v;
      } else {
        next[k] = v;
      }
    }
    onChange(next);
  };

  const handleValueChange = (key: string, newVal: string) => {
    onChange({ ...value, [key]: newVal });
  };

  const handleAddKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleAdd();
    }
  };

  return (
    <div className="space-y-2" data-testid="key-value-editor">
      {entries.map(([k, v], index) => (
        <div key={index} className="flex items-center gap-2">
          <Input
            value={k}
            onChange={(e) => handleKeyChange(k, e.target.value)}
            placeholder={keyPlaceholder}
            readOnly={readOnly}
            className="flex-1"
            data-testid={`kv-key-input-${index}`}
          />
          <Input
            value={v}
            onChange={(e) => handleValueChange(k, e.target.value)}
            placeholder={valuePlaceholder}
            readOnly={readOnly}
            className="flex-1"
            data-testid={`kv-value-input-${index}`}
          />
          {!readOnly && (
            <Button
              type="button"
              variant="ghost"
              size="icon"
              onClick={() => handleRemove(k)}
              data-testid={`kv-remove-btn-${index}`}
            >
              <Trash2 className="h-4 w-4 text-destructive" />
            </Button>
          )}
        </div>
      ))}

      {!readOnly && (
        <div className="flex items-center gap-2">
          <Input
            value={newKey}
            onChange={(e) => setNewKey(e.target.value)}
            onKeyDown={handleAddKeyDown}
            placeholder={keyPlaceholder}
            className="flex-1"
            data-testid="kv-new-key-input"
          />
          <Input
            value={newValue}
            onChange={(e) => setNewValue(e.target.value)}
            onKeyDown={handleAddKeyDown}
            placeholder={valuePlaceholder}
            className="flex-1"
            data-testid="kv-new-value-input"
          />
          <Button
            type="button"
            variant="outline"
            size="icon"
            onClick={handleAdd}
            disabled={!newKey.trim()}
            data-testid="kv-add-btn"
          >
            <Plus className="h-4 w-4" />
          </Button>
        </div>
      )}

      {entries.length === 0 && readOnly && (
        <p className="text-sm text-muted-foreground" data-testid="kv-empty">
          No entries
        </p>
      )}
    </div>
  );
}
