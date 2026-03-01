import { Badge } from '@/components/ui/badge';

interface TagBadgesProps {
  tags: Record<string, string[]> | undefined;
  onTagClick?: (key: string, value: string) => void;
}

const keyColorClasses: string[] = [
  'bg-blue-100 text-blue-800 dark:bg-blue-900/40 dark:text-blue-300',
  'bg-purple-100 text-purple-800 dark:bg-purple-900/40 dark:text-purple-300',
  'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300',
  'bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300',
  'bg-rose-100 text-rose-800 dark:bg-rose-900/40 dark:text-rose-300',
  'bg-teal-100 text-teal-800 dark:bg-teal-900/40 dark:text-teal-300',
  'bg-indigo-100 text-indigo-800 dark:bg-indigo-900/40 dark:text-indigo-300',
  'bg-orange-100 text-orange-800 dark:bg-orange-900/40 dark:text-orange-300',
];

function getColorForKey(_key: string, index: number): string {
  return keyColorClasses[index % keyColorClasses.length];
}

export function TagBadges({ tags, onTagClick }: TagBadgesProps) {
  if (!tags || Object.keys(tags).length === 0) return null;

  const entries = Object.entries(tags);

  return (
    <div className="flex flex-wrap gap-1.5" data-testid="tag-badges">
      {entries.map(([key, values], keyIndex) =>
        values.map((value) => (
          <Badge
            key={`${key}:${value}`}
            variant="outline"
            className={`${getColorForKey(key, keyIndex)} border-transparent ${
              onTagClick ? 'cursor-pointer hover:opacity-80' : ''
            }`}
            onClick={onTagClick ? () => onTagClick(key, value) : undefined}
            data-testid={`tag-badge-${key}-${value}`}
          >
            {key}:{value}
          </Badge>
        ))
      )}
    </div>
  );
}
