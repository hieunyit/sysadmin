"use client"

import { useState } from "react"
import useSWR, { mutate } from "swr"
import { DashboardLayout } from "@/components/dashboard-layout"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Spinner } from "@/components/ui/spinner"
import {
  RefreshCw,
  LogOut,
  Monitor,
  Globe,
  Clock,
  AlertTriangle,
  CheckCircle,
  XCircle,
  Activity,
} from "lucide-react"
import { formatDistanceToNow } from "date-fns"

const fetcher = (url: string) => fetch(url).then((res) => res.json())

interface Session {
  id: string
  username: string
  userId: string
  ipAddress: string
  start: number
  lastAccess: number
  clients: Record<string, string>
}

interface LoginEvent {
  time: number
  type: string
  userId?: string
  username?: string
  ipAddress?: string
  error?: string
  details?: Record<string, string>
}

export default function KeycloakSessionsPage() {
  const [isLoggingOut, setIsLoggingOut] = useState<string | null>(null)

  const { data: sessionsData, error: sessionsError, isLoading: isLoadingSessions } = useSWR(
    "/api/keycloak/sessions",
    fetcher,
    { refreshInterval: 10000 }
  )

  const sessions: Session[] = sessionsData?.sessions || []

  const handleLogoutSession = async (session: Session) => {
    setIsLoggingOut(session.id)
    try {
      await fetch("/api/keycloak/sessions", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action: "logoutSession", sessionId: session.id }),
      })
      mutate("/api/keycloak/sessions")
    } finally {
      setIsLoggingOut(null)
    }
  }

  const handleLogoutUser = async (userId: string, username: string) => {
    if (!confirm(`Logout all sessions for user "${username}"?`)) return
    
    await fetch("/api/keycloak/sessions", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action: "logoutUser", userId }),
    })
    mutate("/api/keycloak/sessions")
  }

  const handleLogoutAll = async () => {
    await fetch("/api/keycloak/sessions", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action: "logoutAllSessions" }),
    })
    mutate("/api/keycloak/sessions")
  }

  const getInitials = (username: string) => {
    return username.slice(0, 2).toUpperCase()
  }

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-slate-900">Sessions</h1>
            <p className="text-slate-500">Monitor active sessions and login events</p>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              onClick={() => mutate("/api/keycloak/sessions")}
            >
              <RefreshCw className="w-4 h-4 mr-2" />
              Refresh
            </Button>
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button variant="destructive">
                  <LogOut className="w-4 h-4 mr-2" />
                  Logout All
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Logout All Sessions?</AlertDialogTitle>
                  <AlertDialogDescription>
                    This will terminate all active sessions in the realm. All users will be logged out immediately.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction
                    onClick={handleLogoutAll}
                    className="bg-red-600 hover:bg-red-700"
                  >
                    Logout All
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>
        </div>

        {/* Stats Cards */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <Card>
            <CardHeader className="pb-2">
              <CardDescription>Active Sessions</CardDescription>
              <CardTitle className="text-3xl font-bold text-emerald-600">
                {sessions.length}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-center text-sm text-slate-500">
                <Activity className="w-4 h-4 mr-1" />
                Currently online
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardDescription>Unique Users</CardDescription>
              <CardTitle className="text-3xl font-bold">
                {new Set(sessions.map((s) => s.userId)).size}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-center text-sm text-slate-500">
                <Monitor className="w-4 h-4 mr-1" />
                Active users
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardDescription>Unique IPs</CardDescription>
              <CardTitle className="text-3xl font-bold">
                {new Set(sessions.map((s) => s.ipAddress)).size}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-center text-sm text-slate-500">
                <Globe className="w-4 h-4 mr-1" />
                Client locations
              </div>
            </CardContent>
          </Card>
        </div>

        <Tabs defaultValue="sessions" className="space-y-4">
          <TabsList>
            <TabsTrigger value="sessions">Active Sessions</TabsTrigger>
            <TabsTrigger value="events">Login Events</TabsTrigger>
          </TabsList>

          <TabsContent value="sessions">
            <Card>
              <CardContent className="pt-6">
                {isLoadingSessions ? (
                  <div className="flex items-center justify-center py-12">
                    <Spinner className="w-6 h-6 text-emerald-600" />
                  </div>
                ) : sessionsError ? (
                  <div className="text-center py-12 text-red-500">
                    Failed to load sessions. Please check your Keycloak connection.
                  </div>
                ) : sessions.length === 0 ? (
                  <div className="text-center py-12 text-slate-500">
                    <Monitor className="w-12 h-12 mx-auto mb-3 text-slate-300" />
                    <p>No active sessions</p>
                    <p className="text-sm">Sessions will appear here when users log in</p>
                  </div>
                ) : (
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>User</TableHead>
                        <TableHead>IP Address</TableHead>
                        <TableHead>Started</TableHead>
                        <TableHead>Last Access</TableHead>
                        <TableHead>Clients</TableHead>
                        <TableHead className="w-24"></TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {sessions.map((session) => (
                        <TableRow key={session.id}>
                          <TableCell>
                            <div className="flex items-center gap-3">
                              <Avatar className="w-8 h-8">
                                <AvatarFallback className="bg-emerald-100 text-emerald-700 text-xs">
                                  {getInitials(session.username)}
                                </AvatarFallback>
                              </Avatar>
                              <div>
                                <div className="font-medium text-slate-900">{session.username}</div>
                                <div className="text-xs text-slate-500 font-mono">{session.userId.slice(0, 8)}...</div>
                              </div>
                            </div>
                          </TableCell>
                          <TableCell>
                            <div className="flex items-center gap-1.5 text-slate-600 font-mono text-sm">
                              <Globe className="w-3.5 h-3.5" />
                              {session.ipAddress}
                            </div>
                          </TableCell>
                          <TableCell>
                            <div className="flex items-center gap-1.5 text-slate-500 text-sm">
                              <Clock className="w-3.5 h-3.5" />
                              {formatDistanceToNow(new Date(session.start), { addSuffix: true })}
                            </div>
                          </TableCell>
                          <TableCell className="text-slate-500 text-sm">
                            {formatDistanceToNow(new Date(session.lastAccess), { addSuffix: true })}
                          </TableCell>
                          <TableCell>
                            <div className="flex flex-wrap gap-1">
                              {Object.values(session.clients || {}).slice(0, 2).map((client, i) => (
                                <Badge key={i} variant="secondary" className="text-xs">
                                  {client}
                                </Badge>
                              ))}
                              {Object.keys(session.clients || {}).length > 2 && (
                                <Badge variant="secondary" className="text-xs">
                                  +{Object.keys(session.clients).length - 2}
                                </Badge>
                              )}
                            </div>
                          </TableCell>
                          <TableCell>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => handleLogoutSession(session)}
                              disabled={isLoggingOut === session.id}
                              className="text-red-600 hover:text-red-700 hover:bg-red-50"
                            >
                              {isLoggingOut === session.id ? (
                                <Spinner className="w-4 h-4" />
                              ) : (
                                <>
                                  <LogOut className="w-4 h-4 mr-1" />
                                  Logout
                                </>
                              )}
                            </Button>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                )}
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="events">
            <Card>
              <CardContent className="py-12">
                <div className="text-center text-slate-500">
                  <Activity className="w-12 h-12 mx-auto mb-3 text-slate-300" />
                  <p>Login event history</p>
                  <p className="text-sm">Configure event logging in Keycloak to view history</p>
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </DashboardLayout>
  )
}
