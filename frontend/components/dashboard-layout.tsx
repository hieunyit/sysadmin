"use client"

import type React from "react"
import { useState } from "react"
import Link from "next/link"
import { usePathname } from "next/navigation"
import { useSession, signOut } from "next-auth/react"
import {
  Search,
  Bell,
  Home,
  Settings,
  Users,
  Shield,
  Network,
  Key,
  Folder,
  Activity,
  Wifi,
  FileKey,
  ChevronDown,
  ChevronRight,
  LogOut,
  User,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
  DropdownMenuLabel,
} from "@/components/ui/dropdown-menu"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import { cn } from "@/lib/utils"

interface NavItem {
  name: string
  href: string
  icon: React.ElementType
}

interface NavGroup {
  name: string
  icon: React.ElementType
  items: NavItem[]
  color: string
}

const navigation: (NavItem | NavGroup)[] = [
  { name: "Overview", href: "/", icon: Home },
  {
    name: "Keycloak",
    icon: Key,
    color: "text-emerald-600",
    items: [
      { name: "Users", href: "/keycloak/users", icon: Users },
      { name: "Groups", href: "/keycloak/groups", icon: Folder },
      { name: "Roles", href: "/keycloak/roles", icon: Shield },
      { name: "Sessions", href: "/keycloak/sessions", icon: Activity },
    ],
  },
  {
    name: "OpenVPN",
    icon: Network,
    color: "text-teal-600",
    items: [
      { name: "VPN Users", href: "/openvpn/users", icon: Users },
      { name: "Connections", href: "/openvpn/connections", icon: Wifi },
      { name: "Configs", href: "/openvpn/configs", icon: FileKey },
    ],
  },
  { name: "Settings", href: "/settings", icon: Settings },
]

function isNavGroup(item: NavItem | NavGroup): item is NavGroup {
  return "items" in item
}

interface DashboardLayoutProps {
  children: React.ReactNode
}

export function DashboardLayout({ children }: DashboardLayoutProps) {
  const pathname = usePathname()
  const { data: session } = useSession()
  const [openGroups, setOpenGroups] = useState<string[]>(["Keycloak", "OpenVPN"])

  const toggleGroup = (name: string) => {
    setOpenGroups((prev) =>
      prev.includes(name) ? prev.filter((g) => g !== name) : [...prev, name]
    )
  }

  const isActiveLink = (href: string) => {
    if (href === "/") return pathname === "/"
    return pathname.startsWith(href)
  }

  const isGroupActive = (group: NavGroup) => {
    return group.items.some((item) => isActiveLink(item.href))
  }

  const getBreadcrumb = () => {
    if (pathname === "/") return "Overview"
    const parts = pathname.split("/").filter(Boolean)
    return parts.map((p) => p.charAt(0).toUpperCase() + p.slice(1)).join(" / ")
  }

  const getUserInitials = () => {
    if (session?.user?.name) {
      return session.user.name
        .split(" ")
        .map((n) => n[0])
        .join("")
        .toUpperCase()
        .slice(0, 2)
    }
    return session?.user?.email?.slice(0, 2).toUpperCase() || "AD"
  }

  return (
    <div className="min-h-screen bg-slate-50">
      {/* Header */}
      <header className="h-16 border-b border-slate-200 bg-white px-6 flex items-center justify-between sticky top-0 z-50">
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <div className="w-9 h-9 bg-gradient-to-br from-emerald-500 to-teal-600 rounded-xl flex items-center justify-center shadow-sm">
              <Shield className="w-5 h-5 text-white" />
            </div>
            <span className="font-semibold text-slate-900">Admin Portal</span>
          </div>
          <div className="text-sm text-slate-500 hidden md:block">
            <span className="mx-2 text-slate-300">/</span>
            <span>{getBreadcrumb()}</span>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <div className="relative hidden lg:block">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-slate-400 w-4 h-4" />
            <Input
              placeholder="Search..."
              className="pl-10 w-64 bg-slate-50 border-slate-200 focus:bg-white"
            />
          </div>
          <Button variant="ghost" size="icon" className="relative text-slate-600">
            <Bell className="w-5 h-5" />
            <span className="absolute top-1.5 right-1.5 w-2 h-2 bg-red-500 rounded-full"></span>
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="flex items-center gap-2 px-2">
                <Avatar className="w-8 h-8">
                  <AvatarImage src={session?.user?.image || undefined} />
                  <AvatarFallback className="bg-emerald-100 text-emerald-700 text-sm">
                    {getUserInitials()}
                  </AvatarFallback>
                </Avatar>
                <span className="text-sm font-medium text-slate-700 hidden md:inline">
                  {session?.user?.name || session?.user?.email || "Admin"}
                </span>
                <ChevronDown className="w-4 h-4 text-slate-400" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-56">
              <DropdownMenuLabel className="font-normal">
                <div className="flex flex-col space-y-1">
                  <p className="text-sm font-medium">{session?.user?.name || "Admin"}</p>
                  <p className="text-xs text-slate-500">{session?.user?.email}</p>
                </div>
              </DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem>
                <User className="w-4 h-4 mr-2" />
                Profile
              </DropdownMenuItem>
              <DropdownMenuItem asChild>
                <Link href="/settings">
                  <Settings className="w-4 h-4 mr-2" />
                  Settings
                </Link>
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => signOut({ callbackUrl: "/auth/signin" })} className="text-red-600">
                <LogOut className="w-4 h-4 mr-2" />
                Sign out
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </header>

      <div className="flex">
        {/* Sidebar */}
        <aside className="w-64 border-r border-slate-200 bg-white h-[calc(100vh-4rem)] overflow-y-auto sticky top-16">
          <div className="p-4">
            <nav className="space-y-1">
              {navigation.map((item) => {
                if (isNavGroup(item)) {
                  const isOpen = openGroups.includes(item.name)
                  const isActive = isGroupActive(item)

                  return (
                    <Collapsible
                      key={item.name}
                      open={isOpen}
                      onOpenChange={() => toggleGroup(item.name)}
                    >
                      <CollapsibleTrigger asChild>
                        <button
                          className={cn(
                            "flex items-center w-full justify-between px-3 py-2 rounded-lg text-sm font-medium transition-colors",
                            isActive
                              ? "bg-slate-100 text-slate-900"
                              : "text-slate-600 hover:bg-slate-50"
                          )}
                        >
                          <div className="flex items-center">
                            <item.icon className={cn("w-4 h-4 mr-3", item.color)} />
                            {item.name}
                          </div>
                          {isOpen ? (
                            <ChevronDown className="w-4 h-4 text-slate-400" />
                          ) : (
                            <ChevronRight className="w-4 h-4 text-slate-400" />
                          )}
                        </button>
                      </CollapsibleTrigger>
                      <CollapsibleContent className="pl-4 mt-1 space-y-1">
                        {item.items.map((subItem) => {
                          const isSubActive = isActiveLink(subItem.href)
                          return (
                            <Link
                              key={subItem.name}
                              href={subItem.href}
                              className={cn(
                                "flex items-center px-3 py-2 rounded-lg text-sm transition-colors",
                                isSubActive
                                  ? "bg-emerald-50 text-emerald-700 font-medium"
                                  : "text-slate-500 hover:bg-slate-50 hover:text-slate-700"
                              )}
                            >
                              <subItem.icon className="w-4 h-4 mr-3" />
                              {subItem.name}
                            </Link>
                          )
                        })}
                      </CollapsibleContent>
                    </Collapsible>
                  )
                }

                const isActive = isActiveLink(item.href)
                return (
                  <Link
                    key={item.name}
                    href={item.href}
                    className={cn(
                      "flex items-center px-3 py-2 rounded-lg text-sm font-medium transition-colors",
                      isActive
                        ? "bg-emerald-50 text-emerald-700"
                        : "text-slate-600 hover:bg-slate-50"
                    )}
                  >
                    <item.icon className="w-4 h-4 mr-3" />
                    {item.name}
                  </Link>
                )
              })}
            </nav>
          </div>

          {/* Sidebar Footer */}
          <div className="absolute bottom-0 left-0 right-0 p-4 border-t border-slate-100 bg-white">
            <div className="flex items-center gap-3 px-3 py-2 rounded-lg bg-slate-50">
              <div className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></div>
              <div className="text-xs">
                <p className="font-medium text-slate-700">System Status</p>
                <p className="text-slate-500">All services operational</p>
              </div>
            </div>
          </div>
        </aside>

        {/* Main Content */}
        <main className="flex-1 p-6 min-h-[calc(100vh-4rem)]">{children}</main>
      </div>
    </div>
  )
}
