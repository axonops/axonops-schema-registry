import {
  createRouter,
  createRoute,
  createRootRoute,
  redirect,
  Outlet,
  useNavigate,
} from '@tanstack/react-router';
import { useEffect } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Toaster } from '@/components/ui/sonner';
import { TooltipProvider } from '@/components/ui/tooltip';
import { AuthProvider, useAuth } from '@/context/auth-context';
import { RegistryContextProvider } from '@/context/registry-context';
import { ThemeProvider } from '@/context/theme-context';
import { AppLayout } from '@/components/layout/app-layout';
import { LoginPage } from '@/features/auth/login-page';
import { SubjectsListPage } from '@/features/subjects/subjects-list-page';
import { SubjectDetailPage } from '@/features/subjects/subject-detail-page';
import { SchemaVersionPage } from '@/features/schemas/schema-version-page';
import { SchemaBrowserPage } from '@/features/schemas/schema-browser-page';
import { SchemaByIdPage } from '@/features/schemas/schema-by-id-page';
import { AboutPage } from '@/features/about/about-page';
import { RegisterSchemaPage } from '@/features/authoring/register-schema-page';
import { RegisterNewPage } from '@/features/authoring/register-new-page';
import { GlobalConfigPage } from '@/features/config/global-config-page';
import { GlobalModesPage } from '@/features/config/global-modes-page';
import { ImportPage } from '@/features/admin/import-page';
import { UsersPage } from '@/features/admin/users-page';
import { ProfilePage } from '@/features/account/profile-page';
import { CompatibilityCheckPage } from '@/features/tools/compatibility-check-page';
import { SchemaLookupPage } from '@/features/tools/schema-lookup-page';
import { ExportersPage } from '@/features/exporters/exporters-page';
import { ExporterDetailPage } from '@/features/exporters/exporter-detail-page';
import { KEKsPage } from '@/features/encryption/keks-page';
import { KEKDetailPage } from '@/features/encryption/kek-detail-page';
import { DEKDetailPage } from '@/features/encryption/dek-detail-page';
import { ContextsPage } from '@/features/contexts/contexts-page';
import { ApiDocsPage } from '@/features/docs/api-docs-page';
import { DashboardPage } from '@/features/dashboard/dashboard-page';
import { SearchPage } from '@/features/search/search-page';

// ── Query Client ──
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
    },
  },
});

// ── Auth Guard ──
function AuthGuard({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      navigate({ to: '/ui/login', search: { redirect: window.location.pathname } });
    }
  }, [isAuthenticated, isLoading, navigate]);

  if (isLoading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="text-muted-foreground">Loading...</div>
      </div>
    );
  }

  if (!isAuthenticated) return null;

  return <>{children}</>;
}

// ── Root Route ──
const rootRoute = createRootRoute({
  component: () => (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <AuthProvider>
          <RegistryContextProvider>
            <TooltipProvider>
              <Outlet />
              <Toaster position="bottom-right" />
            </TooltipProvider>
          </RegistryContextProvider>
        </AuthProvider>
      </ThemeProvider>
    </QueryClientProvider>
  ),
});

// ── Login Route ──
const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/ui/login',
  component: function LoginRoute() {
    const { isAuthenticated } = useAuth();
    const navigate = useNavigate();
    const searchParams = new URLSearchParams(window.location.search);
    const redirectTo = searchParams.get('redirect') || '/ui/dashboard';

    useEffect(() => {
      if (isAuthenticated) {
        navigate({ to: redirectTo });
      }
    }, [isAuthenticated, navigate, redirectTo]);

    return <LoginPage onSuccess={() => navigate({ to: redirectTo })} />;
  },
});

// ── Authenticated Layout Route ──
const authenticatedRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: 'authenticated',
  component: () => (
    <AuthGuard>
      <AppLayout>
        <Outlet />
      </AppLayout>
    </AuthGuard>
  ),
});

// ── Page Routes ──
const dashboardRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/ui/dashboard',
  component: DashboardPage,
});

const subjectsRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/ui/subjects',
  component: SubjectsListPage,
});

const subjectDetailRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/ui/subjects/$subject',
  component: SubjectDetailPage,
});

const schemaVersionRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/ui/subjects/$subject/versions/$version',
  component: SchemaVersionPage,
});

const registerSchemaRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/ui/subjects/$subject/register',
  component: RegisterSchemaPage,
});

const registerNewRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/ui/register',
  component: RegisterNewPage,
});

const schemasRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/ui/schemas',
  component: SchemaBrowserPage,
});

const searchRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/ui/search',
  component: SearchPage,
});

const schemaByIdRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/ui/schemas/$id',
  component: SchemaByIdPage,
});

const configRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/ui/config',
  component: GlobalConfigPage,
});

const modesRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/ui/modes',
  component: GlobalModesPage,
});

const importRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/ui/import',
  component: ImportPage,
});

const usersRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/ui/admin/users',
  component: UsersPage,
});

const profileRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/ui/account',
  component: ProfilePage,
});

const compatibilityCheckRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/ui/tools/compatibility',
  component: CompatibilityCheckPage,
});

const schemaLookupRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/ui/tools/lookup',
  component: SchemaLookupPage,
});

const exportersRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/ui/exporters',
  component: ExportersPage,
});

const exporterDetailRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/ui/exporters/$name',
  component: ExporterDetailPage,
});

const encryptionRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/ui/encryption',
  component: KEKsPage,
});

const kekDetailRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/ui/encryption/$name',
  component: KEKDetailPage,
});

const dekDetailRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/ui/encryption/$name/deks/$subject',
  component: DEKDetailPage,
});

const contextsRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/ui/contexts',
  component: ContextsPage,
});

const apiDocsRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/ui/api-docs',
  component: ApiDocsPage,
});

const aboutRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/ui/about',
  component: AboutPage,
});

// ── Redirect / → /ui/dashboard ──
const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  beforeLoad: () => {
    throw redirect({ to: '/ui/dashboard' });
  },
});

const uiIndexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/ui',
  beforeLoad: () => {
    throw redirect({ to: '/ui/dashboard' });
  },
});

// ── Route Tree ──
const routeTree = rootRoute.addChildren([
  loginRoute,
  indexRoute,
  uiIndexRoute,
  authenticatedRoute.addChildren([
    dashboardRoute,
    subjectsRoute,
    subjectDetailRoute,
    registerSchemaRoute,
    registerNewRoute,
    schemaVersionRoute,
    schemasRoute,
    searchRoute,
    schemaByIdRoute,
    configRoute,
    modesRoute,
    importRoute,
    usersRoute,
    compatibilityCheckRoute,
    schemaLookupRoute,
    exportersRoute,
    exporterDetailRoute,
    encryptionRoute,
    kekDetailRoute,
    dekDetailRoute,
    profileRoute,
    contextsRoute,
    apiDocsRoute,
    aboutRoute,
  ]),
]);

// ── Router ──
export const router = createRouter({
  routeTree,
  defaultPreload: 'intent',
});

// ── Type Registration ──
declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}
