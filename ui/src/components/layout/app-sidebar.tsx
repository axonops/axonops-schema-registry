import { useLocation, useNavigate } from '@tanstack/react-router';
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarHeader,
  SidebarFooter,
} from '@/components/ui/sidebar';
import {
  BookOpen,
  Search,
  Settings,
  ToggleLeft,
  Users,
  KeyRound,
  Upload,
  Info,
  User,
  Database,
  CircleCheck,
  SearchCheck,
  ArrowRightLeft,
  ShieldCheck,
  Layers,
  FileText,
} from 'lucide-react';
import { useAuth } from '@/context/auth-context';

interface NavItem {
  title: string;
  url: string;
  icon: React.ComponentType<{ className?: string }>;
  testId: string;
}

interface NavGroup {
  label: string;
  items: NavItem[];
  minRole?: string[];
}

const navGroups: NavGroup[] = [
  {
    label: 'SCHEMAS',
    items: [
      { title: 'Subjects', url: '/ui/subjects', icon: BookOpen, testId: 'nav-sidebar-subjects-link' },
      { title: 'Schema Browser', url: '/ui/schemas', icon: Search, testId: 'nav-sidebar-schemas-link' },
    ],
  },
  {
    label: 'CONFIGURATION',
    items: [
      { title: 'Compatibility', url: '/ui/config', icon: Settings, testId: 'nav-sidebar-config-link' },
      { title: 'Modes', url: '/ui/modes', icon: ToggleLeft, testId: 'nav-sidebar-modes-link' },
      { title: 'Exporters', url: '/ui/exporters', icon: ArrowRightLeft, testId: 'nav-sidebar-exporters-link' },
    ],
    minRole: ['super_admin', 'admin'],
  },
  {
    label: 'TOOLS',
    items: [
      { title: 'Compatibility Check', url: '/ui/tools/compatibility', icon: CircleCheck, testId: 'nav-sidebar-compat-check-link' },
      { title: 'Schema Lookup', url: '/ui/tools/lookup', icon: SearchCheck, testId: 'nav-sidebar-schema-lookup-link' },
    ],
  },
  {
    label: 'SECURITY',
    items: [
      { title: 'Encryption Keys', url: '/ui/encryption', icon: ShieldCheck, testId: 'nav-sidebar-encryption-link' },
    ],
    minRole: ['super_admin', 'admin'],
  },
  {
    label: 'ADMINISTRATION',
    items: [
      { title: 'Users', url: '/ui/admin/users', icon: Users, testId: 'nav-sidebar-users-link' },
      { title: 'API Keys', url: '/ui/admin/apikeys', icon: KeyRound, testId: 'nav-sidebar-apikeys-link' },
      { title: 'Import', url: '/ui/import', icon: Upload, testId: 'nav-sidebar-import-link' },
    ],
    minRole: ['super_admin', 'admin'],
  },
  {
    label: 'ACCOUNT',
    items: [
      { title: 'My Profile', url: '/ui/account', icon: User, testId: 'nav-sidebar-profile-link' },
      { title: 'My API Keys', url: '/ui/account/apikeys', icon: KeyRound, testId: 'nav-sidebar-my-apikeys-link' },
    ],
  },
  {
    label: 'SYSTEM',
    items: [
      { title: 'API Docs', url: '/ui/api-docs', icon: FileText, testId: 'nav-sidebar-api-docs-link' },
      { title: 'Contexts', url: '/ui/contexts', icon: Layers, testId: 'nav-sidebar-contexts-link' },
      { title: 'About', url: '/ui/about', icon: Info, testId: 'nav-sidebar-about-link' },
    ],
  },
];

export function AppSidebar() {
  const { user } = useAuth();
  const location = useLocation();
  const navigate = useNavigate();

  const isActive = (url: string) => {
    return location.pathname === url || location.pathname.startsWith(url + '/');
  };

  const hasRole = (group: NavGroup) => {
    if (!group.minRole) return true;
    if (!user) return false;
    return group.minRole.includes(user.role);
  };

  return (
    <Sidebar data-testid="app-sidebar">
      <SidebarHeader className="border-b px-4 py-3">
        <div className="flex items-center gap-2">
          <Database className="h-5 w-5" />
          <span className="font-semibold text-sm">Schema Registry</span>
        </div>
      </SidebarHeader>
      <SidebarContent>
        {navGroups.filter(hasRole).map((group) => (
          <SidebarGroup key={group.label}>
            <SidebarGroupLabel>{group.label}</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {group.items.map((item) => (
                  <SidebarMenuItem key={item.url}>
                    <SidebarMenuButton
                      isActive={isActive(item.url)}
                      onClick={() => navigate({ to: item.url })}
                      data-testid={item.testId}
                    >
                      <item.icon className="h-4 w-4" />
                      <span>{item.title}</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        ))}
      </SidebarContent>
      <SidebarFooter className="border-t px-4 py-2">
        <div className="text-xs text-muted-foreground">
          AxonOps Schema Registry
        </div>
      </SidebarFooter>
    </Sidebar>
  );
}
