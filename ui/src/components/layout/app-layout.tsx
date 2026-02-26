import { SidebarProvider, SidebarInset } from '@/components/ui/sidebar';
import { AppSidebar } from './app-sidebar';
import { TopBar } from './top-bar';
import { StatusBar } from './status-bar';

export function AppLayout({ children }: { children: React.ReactNode }) {
  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <TopBar />
        <main className="flex-1 overflow-auto p-4 md:p-6">
          {children}
        </main>
        <StatusBar />
      </SidebarInset>
    </SidebarProvider>
  );
}
