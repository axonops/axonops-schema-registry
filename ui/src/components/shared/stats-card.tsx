import { Card, CardContent } from '@/components/ui/card';
import { cn } from '@/lib/utils';

interface StatsCardProps {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  value: string | number;
  subtitle?: string;
  className?: string;
}

export function StatsCard({ icon: Icon, label, value, subtitle, className }: StatsCardProps) {
  return (
    <Card className={cn('py-4', className)} data-testid="stats-card">
      <CardContent className="flex flex-col gap-1 px-4">
        <Icon className="h-4 w-4 text-muted-foreground" />
        <p className="text-xs text-muted-foreground">{label}</p>
        <p className="text-2xl font-bold tabular-nums leading-none">{value}</p>
        {subtitle && (
          <p className="text-xs text-muted-foreground">{subtitle}</p>
        )}
      </CardContent>
    </Card>
  );
}
