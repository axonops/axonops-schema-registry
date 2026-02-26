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
          <TooltipProvider>
            <Outlet />
            <Toaster position="bottom-right" />
          </TooltipProvider>
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
    const redirectTo = searchParams.get('redirect') || '/ui/subjects';

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

const schemasRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/ui/schemas',
  component: SchemaBrowserPage,
});

const schemaByIdRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/ui/schemas/$id',
  component: SchemaByIdPage,
});

const aboutRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/ui/about',
  component: AboutPage,
});

// ── Redirect / → /ui/subjects ──
const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  beforeLoad: () => {
    throw redirect({ to: '/ui/subjects' });
  },
});

const uiIndexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/ui',
  beforeLoad: () => {
    throw redirect({ to: '/ui/subjects' });
  },
});

// ── Route Tree ──
const routeTree = rootRoute.addChildren([
  loginRoute,
  indexRoute,
  uiIndexRoute,
  authenticatedRoute.addChildren([
    subjectsRoute,
    subjectDetailRoute,
    registerSchemaRoute,
    schemaVersionRoute,
    schemasRoute,
    schemaByIdRoute,
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
