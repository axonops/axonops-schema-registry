import { useAuth } from '@/context/auth-context';
import { useTheme } from '@/context/theme-context';
import { SidebarTrigger } from '@/components/ui/sidebar';
import { Separator } from '@/components/ui/separator';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Button } from '@/components/ui/button';
import { Avatar, AvatarFallback } from '@/components/ui/avatar';
import { Moon, Sun, Monitor, LogOut, User } from 'lucide-react';
import { useNavigate } from '@tanstack/react-router';

export function TopBar() {
  const { user, logout } = useAuth();
  const { theme, setTheme } = useTheme();
  const navigate = useNavigate();

  const handleLogout = async () => {
    await logout();
    navigate({ to: '/ui/login' });
  };

  const initials = user?.username
    ? user.username.slice(0, 2).toUpperCase()
    : '??';

  return (
    <header className="flex h-14 items-center gap-2 border-b px-4" data-testid="top-bar">
      <SidebarTrigger data-testid="sidebar-toggle" />
      <Separator orientation="vertical" className="h-6" />

      <div className="flex-1" />

      {/* Theme toggle */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="icon" data-testid="theme-toggle">
            {theme === 'dark' ? <Moon className="h-4 w-4" /> :
             theme === 'light' ? <Sun className="h-4 w-4" /> :
             <Monitor className="h-4 w-4" />}
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem onClick={() => setTheme('light')}>
            <Sun className="mr-2 h-4 w-4" /> Light
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => setTheme('dark')}>
            <Moon className="mr-2 h-4 w-4" /> Dark
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => setTheme('system')}>
            <Monitor className="mr-2 h-4 w-4" /> System
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      {/* User menu */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" className="flex items-center gap-2" data-testid="user-menu-trigger">
            <Avatar className="h-7 w-7">
              <AvatarFallback className="text-xs">{initials}</AvatarFallback>
            </Avatar>
            <span className="text-sm hidden sm:inline" data-testid="user-menu-username">
              {user?.username}
            </span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-56" data-testid="user-menu">
          <DropdownMenuLabel>
            <div className="flex flex-col">
              <span>{user?.username}</span>
              <span className="text-xs font-normal text-muted-foreground">{user?.role}</span>
            </div>
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={() => navigate({ to: '/ui/about' })}>
            <User className="mr-2 h-4 w-4" /> My Profile
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={handleLogout} data-testid="user-menu-signout">
            <LogOut className="mr-2 h-4 w-4" /> Sign Out
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </header>
  );
}
