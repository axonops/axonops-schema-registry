import { useNavigate } from '@tanstack/react-router';
import {
  Breadcrumb,
  BreadcrumbItem as BreadcrumbItemUI,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb';
import { Fragment } from 'react';

export interface BreadcrumbEntry {
  label: string;
  href?: string;
}

interface PageBreadcrumbsProps {
  items: BreadcrumbEntry[];
}

export function PageBreadcrumbs({ items }: PageBreadcrumbsProps) {
  const navigate = useNavigate();

  return (
    <Breadcrumb className="mb-4" data-testid="breadcrumbs">
      <BreadcrumbList>
        {items.map((item, index) => {
          const isLast = index === items.length - 1;
          return (
            <Fragment key={item.label}>
              {index > 0 && <BreadcrumbSeparator />}
              <BreadcrumbItemUI>
                {isLast || !item.href ? (
                  <BreadcrumbPage>{item.label}</BreadcrumbPage>
                ) : (
                  <BreadcrumbLink
                    className="cursor-pointer"
                    onClick={() => navigate({ to: item.href! })}
                  >
                    {item.label}
                  </BreadcrumbLink>
                )}
              </BreadcrumbItemUI>
            </Fragment>
          );
        })}
      </BreadcrumbList>
    </Breadcrumb>
  );
}
