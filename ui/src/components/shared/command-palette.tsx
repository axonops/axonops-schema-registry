import { useState, useEffect, useRef, useMemo, useCallback } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { useSubjects } from '@/api/queries';
import { Dialog, DialogContent } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import {
  BookOpen,
  Search,
  Settings,
  ToggleLeft,
  Upload,
  Users,
  User,
  Info,
  Hash,
  LayoutDashboard,
  FilePlus2,
  CircleCheck,
  SearchCheck,
  ArrowRightLeft,
  ShieldCheck,
  FileText,
  Layers,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';

interface PageItem {
  label: string;
  path: string;
  icon: LucideIcon;
}

interface ResultItem {
  type: 'subject' | 'page';
  label: string;
  path: string;
  params?: Record<string, string>;
  icon: LucideIcon;
}

const pages: PageItem[] = [
  { label: 'Dashboard', path: '/ui/dashboard', icon: LayoutDashboard },
  { label: 'Subjects', path: '/ui/subjects', icon: BookOpen },
  { label: 'Register Schema', path: '/ui/register', icon: FilePlus2 },
  { label: 'Search', path: '/ui/search', icon: Search },
  { label: 'Compatibility', path: '/ui/config', icon: Settings },
  { label: 'Modes', path: '/ui/modes', icon: ToggleLeft },
  { label: 'Exporters', path: '/ui/exporters', icon: ArrowRightLeft },
  { label: 'Compat Check', path: '/ui/tools/compatibility', icon: CircleCheck },
  { label: 'Schema Lookup', path: '/ui/tools/lookup', icon: SearchCheck },
  { label: 'Encryption Keys', path: '/ui/encryption', icon: ShieldCheck },
  { label: 'Import', path: '/ui/import', icon: Upload },
  { label: 'Users', path: '/ui/admin/users', icon: Users },
  { label: 'API Docs', path: '/ui/api-docs', icon: FileText },
  { label: 'Contexts', path: '/ui/contexts', icon: Layers },
  { label: 'My Profile', path: '/ui/account', icon: User },
  { label: 'About', path: '/ui/about', icon: Info },
];

const MAX_RESULTS = 10;

export function CommandPalette() {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState('');
  const [selectedIndex, setSelectedIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();
  const { data: subjects } = useSubjects();

  // Global keyboard shortcut: Ctrl+K / Cmd+K
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setOpen((prev) => !prev);
      }
    }
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, []);

  // Reset state when dialog opens
  useEffect(() => {
    if (open) {
      setQuery('');
      setSelectedIndex(0);
    }
  }, [open]);

  // Compute filtered results
  const results = useMemo(() => {
    const lowerQuery = query.toLowerCase();
    const items: ResultItem[] = [];

    // Search subjects
    const matchingSubjects = (subjects ?? [])
      .filter((s) => s.toLowerCase().includes(lowerQuery))
      .slice(0, MAX_RESULTS);

    for (const subject of matchingSubjects) {
      items.push({
        type: 'subject',
        label: subject,
        path: '/ui/subjects/$subject',
        params: { subject },
        icon: Hash,
      });
    }

    // Search pages
    const matchingPages = pages.filter((p) =>
      p.label.toLowerCase().includes(lowerQuery),
    );

    for (const page of matchingPages) {
      items.push({
        type: 'page',
        label: page.label,
        path: page.path,
        icon: page.icon,
      });
    }

    return items.slice(0, MAX_RESULTS);
  }, [query, subjects]);

  // Group results by type for display
  const subjectResults = results.filter((r) => r.type === 'subject');
  const pageResults = results.filter((r) => r.type === 'page');

  // Clamp selected index when results change
  useEffect(() => {
    setSelectedIndex((prev) => Math.min(prev, Math.max(results.length - 1, 0)));
  }, [results]);

  const navigateToResult = useCallback(
    (item: ResultItem) => {
      setOpen(false);
      if (item.params) {
        navigate({ to: item.path, params: item.params });
      } else {
        navigate({ to: item.path });
      }
    },
    [navigate],
  );

  // Keyboard navigation inside the palette
  function handleKeyDown(e: React.KeyboardEvent) {
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setSelectedIndex((prev) =>
          prev < results.length - 1 ? prev + 1 : 0,
        );
        break;
      case 'ArrowUp':
        e.preventDefault();
        setSelectedIndex((prev) =>
          prev > 0 ? prev - 1 : results.length - 1,
        );
        break;
      case 'Enter':
        e.preventDefault();
        if (results[selectedIndex]) {
          navigateToResult(results[selectedIndex]);
        }
        break;
      case 'Escape':
        e.preventDefault();
        setOpen(false);
        break;
    }
  }

  // Scroll the selected item into view
  useEffect(() => {
    const container = listRef.current;
    if (!container) return;
    const selected = container.querySelector('[data-selected="true"]');
    if (selected) {
      selected.scrollIntoView({ block: 'nearest' });
    }
  }, [selectedIndex]);

  // Compute the flat index for a given group item
  function flatIndex(type: 'subject' | 'page', indexInGroup: number): number {
    if (type === 'subject') {
      return indexInGroup;
    }
    return subjectResults.length + indexInGroup;
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent
        data-testid="command-palette"
        showCloseButton={false}
        className="gap-0 overflow-hidden p-0 sm:max-w-lg"
        onKeyDown={handleKeyDown}
      >
        <div className="flex items-center border-b px-3">
          <Search className="text-muted-foreground mr-2 size-4 shrink-0" />
          <Input
            ref={inputRef}
            data-testid="command-palette-input"
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setSelectedIndex(0);
            }}
            placeholder="Search subjects and pages..."
            className="h-11 border-0 shadow-none focus-visible:ring-0"
            autoFocus
          />
          <kbd className="bg-muted text-muted-foreground pointer-events-none ml-2 hidden h-5 select-none items-center gap-1 rounded border px-1.5 font-mono text-[10px] font-medium opacity-100 sm:inline-flex">
            Esc
          </kbd>
        </div>

        <div ref={listRef} className="max-h-72 overflow-y-auto p-1">
          {results.length === 0 && (
            <div className="text-muted-foreground py-6 text-center text-sm">
              No results found.
            </div>
          )}

          {subjectResults.length > 0 && (
            <div>
              <div className="text-muted-foreground px-2 py-1.5 text-xs font-medium">
                Subjects
              </div>
              {subjectResults.map((item, i) => {
                const idx = flatIndex('subject', i);
                const isSelected = idx === selectedIndex;
                return (
                  <button
                    key={`subject-${item.label}`}
                    data-selected={isSelected}
                    className={`flex w-full cursor-pointer items-center gap-2 rounded-sm px-2 py-1.5 text-sm outline-none ${
                      isSelected
                        ? 'bg-accent text-accent-foreground'
                        : 'text-foreground hover:bg-accent/50'
                    }`}
                    onClick={() => navigateToResult(item)}
                    onMouseEnter={() => setSelectedIndex(idx)}
                  >
                    <item.icon className="text-muted-foreground size-4 shrink-0" />
                    <span className="truncate">{item.label}</span>
                  </button>
                );
              })}
            </div>
          )}

          {pageResults.length > 0 && (
            <div>
              <div className="text-muted-foreground px-2 py-1.5 text-xs font-medium">
                Pages
              </div>
              {pageResults.map((item, i) => {
                const idx = flatIndex('page', i);
                const isSelected = idx === selectedIndex;
                return (
                  <button
                    key={`page-${item.path}`}
                    data-selected={isSelected}
                    className={`flex w-full cursor-pointer items-center gap-2 rounded-sm px-2 py-1.5 text-sm outline-none ${
                      isSelected
                        ? 'bg-accent text-accent-foreground'
                        : 'text-foreground hover:bg-accent/50'
                    }`}
                    onClick={() => navigateToResult(item)}
                    onMouseEnter={() => setSelectedIndex(idx)}
                  >
                    <item.icon className="text-muted-foreground size-4 shrink-0" />
                    <span className="truncate">{item.label}</span>
                  </button>
                );
              })}
            </div>
          )}
        </div>

        <div className="border-t px-3 py-2">
          <div className="text-muted-foreground flex items-center gap-3 text-xs">
            <span className="flex items-center gap-1">
              <kbd className="bg-muted rounded border px-1 font-mono text-[10px]">
                &uarr;
              </kbd>
              <kbd className="bg-muted rounded border px-1 font-mono text-[10px]">
                &darr;
              </kbd>
              <span>navigate</span>
            </span>
            <span className="flex items-center gap-1">
              <kbd className="bg-muted rounded border px-1 font-mono text-[10px]">
                &crarr;
              </kbd>
              <span>select</span>
            </span>
            <span className="flex items-center gap-1">
              <kbd className="bg-muted rounded border px-1 font-mono text-[10px]">
                esc
              </kbd>
              <span>close</span>
            </span>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
