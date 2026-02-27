import { useState } from 'react';
import { ChevronDown, ChevronRight } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import type { RuleSet, Rule } from '@/api/queries';

interface RuleSetPanelProps {
  ruleSet: RuleSet | undefined;
  title?: string;
}

const kindColors: Record<Rule['kind'], string> = {
  CONDITION:
    'bg-blue-100 text-blue-800 dark:bg-blue-900/40 dark:text-blue-300 border-transparent',
  TRANSFORM:
    'bg-purple-100 text-purple-800 dark:bg-purple-900/40 dark:text-purple-300 border-transparent',
};

const modeColors: Record<Rule['mode'], string> = {
  WRITE:
    'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300 border-transparent',
  READ:
    'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/40 dark:text-yellow-300 border-transparent',
  WRITEREAD:
    'bg-orange-100 text-orange-800 dark:bg-orange-900/40 dark:text-orange-300 border-transparent',
};

function hasRules(ruleSet: RuleSet | undefined): boolean {
  if (!ruleSet) return false;
  return (
    (ruleSet.migrationRules?.length ?? 0) > 0 ||
    (ruleSet.domainRules?.length ?? 0) > 0 ||
    (ruleSet.encodingRules?.length ?? 0) > 0
  );
}

function RuleRow({ rule }: { rule: Rule }) {
  const isDisabled = rule.disabled === true;

  return (
    <div
      className={`flex flex-col gap-1.5 rounded-md border p-3 ${
        isDisabled ? 'opacity-50' : ''
      }`}
      data-testid={`rule-row-${rule.name}`}
    >
      <div className="flex items-center gap-2 flex-wrap">
        <span
          className={`font-medium text-sm ${
            isDisabled ? 'line-through text-muted-foreground' : ''
          }`}
        >
          {rule.name}
        </span>
        <Badge variant="outline" className={kindColors[rule.kind]}>
          {rule.kind}
        </Badge>
        <Badge variant="outline" className={modeColors[rule.mode]}>
          {rule.mode}
        </Badge>
        {rule.type && (
          <span className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs">
            {rule.type}
          </span>
        )}
        {isDisabled && (
          <Badge variant="secondary" className="text-xs">
            disabled
          </Badge>
        )}
      </div>
      {rule.doc && (
        <p className="text-xs text-muted-foreground">{rule.doc}</p>
      )}
      {rule.expr && (
        <pre className="mt-0.5 overflow-hidden truncate rounded bg-muted px-2 py-1 font-mono text-xs text-muted-foreground">
          {rule.expr}
        </pre>
      )}
    </div>
  );
}

function RuleSection({
  label,
  rules,
  defaultOpen = true,
}: {
  label: string;
  rules: Rule[];
  defaultOpen?: boolean;
}) {
  const [open, setOpen] = useState(defaultOpen);

  if (rules.length === 0) return null;

  return (
    <div data-testid={`rule-section-${label.toLowerCase().replace(/\s+/g, '-')}`}>
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="flex w-full items-center gap-1.5 py-1 text-left text-xs font-medium uppercase tracking-wide text-muted-foreground hover:text-foreground transition-colors"
        data-testid={`rule-section-toggle-${label.toLowerCase().replace(/\s+/g, '-')}`}
      >
        {open ? (
          <ChevronDown className="h-3.5 w-3.5" />
        ) : (
          <ChevronRight className="h-3.5 w-3.5" />
        )}
        {label}
        <span className="ml-auto tabular-nums">{rules.length}</span>
      </button>
      {open && (
        <div className="mt-2 space-y-2">
          {rules.map((rule) => (
            <RuleRow key={rule.name} rule={rule} />
          ))}
        </div>
      )}
    </div>
  );
}

export function RuleSetPanel({ ruleSet, title = 'Rules' }: RuleSetPanelProps) {
  if (!hasRules(ruleSet)) {
    return (
      <Card data-testid="rule-set-panel">
        <CardHeader>
          <CardTitle className="text-sm">{title}</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground" data-testid="rules-empty">
            No rules defined
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card data-testid="rule-set-panel">
      <CardHeader>
        <CardTitle className="text-sm">{title}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <RuleSection label="Migration Rules" rules={ruleSet!.migrationRules ?? []} />
        <RuleSection label="Domain Rules" rules={ruleSet!.domainRules ?? []} />
        <RuleSection label="Encoding Rules" rules={ruleSet!.encodingRules ?? []} />
      </CardContent>
    </Card>
  );
}
