"use client"

import { useState, useMemo } from "react"
import { DashboardLayout } from "@/components/dashboard-layout"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardHeader } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  Search,
  Download,
  RefreshCw,
  Filter,
  ClipboardList,
  User,
  Folder,
  Shield,
  Key,
  LogIn,
  LogOut,
  Trash2,
  Edit,
  Plus,
  AlertTriangle,
  CheckCircle,
  XCircle,
  ChevronLeft,
  ChevronRight,
  Eye,
  Network,
} from "lucide-react"
import { formatDistanceToNow, format } from "date-fns"

interface AuditLog {
  id: string
  timestamp: string
  actor: string
  actorType: "admin" | "system" | "user"
  action: string
  actionType: "create" | "update" | "delete" | "login" | "logout" | "assign" | "revoke" | "enable" | "disable" | "error"
  target: string
  targetType: "user" | "group" | "role" | "session" | "vpn" | "system"
  source: "keycloak" | "openvpn" | "system"
  status: "success" | "failure" | "warning"
  ipAddress?: string
  details?: string
}

const mockLogs: AuditLog[] = [
  {
    id: "log1",
    timestamp: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
    actor: "admin",
    actorType: "admin",
    action: "User Created",
    actionType: "create",
    target: "john.doe",
    targetType: "user",
    source: "keycloak",
    status: "success",
    ipAddress: "192.168.1.100",
    details: "New user account created with temporary password. Email: john.doe@company.com",
  },
  {
    id: "log2",
    timestamp: new Date(Date.now() - 12 * 60 * 1000).toISOString(),
    actor: "admin",
    actorType: "admin",
    action: "Password Reset",
    actionType: "update",
    target: "jane.smith",
    targetType: "user",
    source: "keycloak",
    status: "success",
    ipAddress: "192.168.1.100",
    details: "Admin reset password for user. Temporary password flag set to true.",
  },
  {
    id: "log3",
    timestamp: new Date(Date.now() - 25 * 60 * 1000).toISOString(),
    actor: "system",
    actorType: "system",
    action: "VPN Session Started",
    actionType: "login",
    target: "mike.wilson",
    targetType: "vpn",
    source: "openvpn",
    status: "success",
    ipAddress: "203.0.113.45",
    details: "VPN connection established. Assigned IP: 10.8.0.4",
  },
  {
    id: "log4",
    timestamp: new Date(Date.now() - 40 * 60 * 1000).toISOString(),
    actor: "admin",
    actorType: "admin",
    action: "User Added to Group",
    actionType: "assign",
    target: "sarah.johnson",
    targetType: "group",
    source: "keycloak",
    status: "success",
    ipAddress: "192.168.1.100",
    details: "User sarah.johnson added to group: developers",
  },
  {
    id: "log5",
    timestamp: new Date(Date.now() - 55 * 60 * 1000).toISOString(),
    actor: "admin",
    actorType: "admin",
    action: "User Disabled",
    actionType: "disable",
    target: "robert.taylor",
    targetType: "user",
    source: "keycloak",
    status: "success",
    ipAddress: "192.168.1.100",
    details: "User account disabled by administrator.",
  },
  {
    id: "log6",
    timestamp: new Date(Date.now() - 70 * 60 * 1000).toISOString(),
    actor: "system",
    actorType: "system",
    action: "Login Failed",
    actionType: "error",
    target: "unknown",
    targetType: "user",
    source: "keycloak",
    status: "failure",
    ipAddress: "198.51.100.22",
    details: "Invalid credentials for username: unknown. 3 consecutive failed attempts.",
  },
  {
    id: "log7",
    timestamp: new Date(Date.now() - 90 * 60 * 1000).toISOString(),
    actor: "admin",
    actorType: "admin",
    action: "Group Created",
    actionType: "create",
    target: "devops-team",
    targetType: "group",
    source: "keycloak",
    status: "success",
    ipAddress: "192.168.1.100",
    details: "New group created: devops-team",
  },
  {
    id: "log8",
    timestamp: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
    actor: "admin",
    actorType: "admin",
    action: "VPN User Created",
    actionType: "create",
    target: "lisa.anderson",
    targetType: "vpn",
    source: "openvpn",
    status: "success",
    ipAddress: "192.168.1.100",
    details: "New VPN user created. Group: support. Auto-login: disabled.",
  },
  {
    id: "log9",
    timestamp: new Date(Date.now() - 3 * 60 * 60 * 1000).toISOString(),
    actor: "system",
    actorType: "system",
    action: "VPN Session Ended",
    actionType: "logout",
    target: "david.brown",
    targetType: "vpn",
    source: "openvpn",
    status: "success",
    ipAddress: "10.8.0.6",
    details: "VPN connection terminated. Duration: 2h 15m. Bytes received: 145MB.",
  },
  {
    id: "log10",
    timestamp: new Date(Date.now() - 4 * 60 * 60 * 1000).toISOString(),
    actor: "admin",
    actorType: "admin",
    action: "Role Assigned",
    actionType: "assign",
    target: "jane.smith",
    targetType: "role",
    source: "keycloak",
    status: "success",
    ipAddress: "192.168.1.100",
    details: "Role 'developer' assigned to user jane.smith",
  },
  {
    id: "log11",
    timestamp: new Date(Date.now() - 5 * 60 * 60 * 1000).toISOString(),
    actor: "admin",
    actorType: "admin",
    action: "User Deleted",
    actionType: "delete",
    target: "temp.user",
    targetType: "user",
    source: "keycloak",
    status: "success",
    ipAddress: "192.168.1.100",
    details: "User account permanently deleted.",
  },
  {
    id: "log12",
    timestamp: new Date(Date.now() - 6 * 60 * 60 * 1000).toISOString(),
    actor: "system",
    actorType: "system",
    action: "Certificate Expired",
    actionType: "error",
    target: "old.vpn.user",
    targetType: "vpn",
    source: "openvpn",
    status: "warning",
    details: "VPN certificate expired for user old.vpn.user. Auto-revoked.",
  },
  {
    id: "log13",
    timestamp: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
    actor: "admin",
    actorType: "admin",
    action: "VPN Group Created",
    actionType: "create",
    target: "contractors",
    targetType: "group",
    source: "openvpn",
    status: "success",
    ipAddress: "192.168.1.100",
    details: "New VPN group created: contractors",
  },
  {
    id: "log14",
    timestamp: new Date(Date.now() - 30 * 60 * 60 * 1000).toISOString(),
    actor: "admin",
    actorType: "admin",
    action: "User Sessions Revoked",
    actionType: "revoke",
    target: "mike.wilson",
    targetType: "session",
    source: "keycloak",
    status: "success",
    ipAddress: "192.168.1.100",
    details: "All active sessions revoked for user mike.wilson",
  },
  {
    id: "log15",
    timestamp: new Date(Date.now() - 48 * 60 * 60 * 1000).toISOString(),
    actor: "system",
    actorType: "system",
    action: "Sync Completed",
    actionType: "update",
    target: "system",
    targetType: "system",
    source: "system",
    status: "success",
    details: "Scheduled Keycloak-VPN sync completed. 42 users synced.",
  },
]

const ACTION_TYPE_ICONS: Record<AuditLog["actionType"], React.ElementType> = {
  create: Plus,
  update: Edit,
  delete: Trash2,
  login: LogIn,
  logout: LogOut,
  assign: Shield,
  revoke: XCircle,
  enable: CheckCircle,
  disable: XCircle,
  error: AlertTriangle,
}

const ACTION_TYPE_COLORS: Record<AuditLog["actionType"], string> = {
  create: "text-emerald-600 bg-emerald-50",
  update: "text-blue-600 bg-blue-50",
  delete: "text-red-600 bg-red-50",
  login: "text-teal-600 bg-teal-50",
  logout: "text-slate-600 bg-slate-50",
  assign: "text-purple-600 bg-purple-50",
  revoke: "text-orange-600 bg-orange-50",
  enable: "text-emerald-600 bg-emerald-50",
  disable: "text-amber-600 bg-amber-50",
  error: "text-red-600 bg-red-50",
}

const TARGET_ICONS: Record<AuditLog["targetType"], React.ElementType> = {
  user: User,
  group: Folder,
  role: Shield,
  session: Key,
  vpn: Network,
  system: ClipboardList,
}

export default function AuditLogsPage() {
  const [search, setSearch] = useState("")
  const [sourceFilter, setSourceFilter] = useState("all")
  const [statusFilter, setStatusFilter] = useState("all")
  const [actionTypeFilter, setActionTypeFilter] = useState("all")
  const [page, setPage] = useState(0)
  const [selectedLog, setSelectedLog] = useState<AuditLog | null>(null)
  const pageSize = 10

  const filteredLogs = useMemo(() => {
    let logs = [...mockLogs]

    if (search) {
      const q = search.toLowerCase()
      logs = logs.filter(
        (l) =>
          l.actor.toLowerCase().includes(q) ||
          l.action.toLowerCase().includes(q) ||
          l.target.toLowerCase().includes(q) ||
          l.ipAddress?.includes(q) ||
          l.details?.toLowerCase().includes(q)
      )
    }
    if (sourceFilter !== "all") logs = logs.filter((l) => l.source === sourceFilter)
    if (statusFilter !== "all") logs = logs.filter((l) => l.status === statusFilter)
    if (actionTypeFilter !== "all") logs = logs.filter((l) => l.actionType === actionTypeFilter)

    return logs.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
  }, [search, sourceFilter, statusFilter, actionTypeFilter])

  const totalPages = Math.ceil(filteredLogs.length / pageSize)
  const paginatedLogs = filteredLogs.slice(page * pageSize, (page + 1) * pageSize)

  const getStatusBadge = (status: AuditLog["status"]) => {
    switch (status) {
      case "success":
        return <Badge className="bg-emerald-100 text-emerald-700 hover:bg-emerald-100">Success</Badge>
      case "failure":
        return <Badge className="bg-red-100 text-red-700 hover:bg-red-100">Failure</Badge>
      case "warning":
        return <Badge className="bg-amber-100 text-amber-700 hover:bg-amber-100">Warning</Badge>
    }
  }

  const getSourceBadge = (source: AuditLog["source"]) => {
    switch (source) {
      case "keycloak":
        return <Badge variant="outline" className="text-xs text-emerald-700 border-emerald-200">Keycloak</Badge>
      case "openvpn":
        return <Badge variant="outline" className="text-xs text-teal-700 border-teal-200">OpenVPN</Badge>
      case "system":
        return <Badge variant="outline" className="text-xs text-slate-600 border-slate-200">System</Badge>
    }
  }

  const exportLogs = () => {
    const csv = [
      ["Timestamp", "Actor", "Action", "Target", "Source", "Status", "IP Address", "Details"],
      ...filteredLogs.map((l) => [
        format(new Date(l.timestamp), "yyyy-MM-dd HH:mm:ss"),
        l.actor,
        l.action,
        l.target,
        l.source,
        l.status,
        l.ipAddress || "",
        l.details || "",
      ]),
    ]
      .map((row) => row.map((cell) => `"${cell}"`).join(","))
      .join("\n")

    const blob = new Blob([csv], { type: "text/csv" })
    const url = URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = url
    a.download = `audit-logs-${format(new Date(), "yyyy-MM-dd")}.csv`
    a.click()
  }

  // Stats
  const successCount = filteredLogs.filter((l) => l.status === "success").length
  const failureCount = filteredLogs.filter((l) => l.status === "failure").length
  const warningCount = filteredLogs.filter((l) => l.status === "warning").length

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-foreground">Audit Logs</h1>
            <p className="text-muted-foreground">Track all administrative actions across Keycloak and OpenVPN</p>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={exportLogs}>
              <Download className="w-4 h-4 mr-2" />
              Export CSV
            </Button>
            <Button variant="outline" size="sm" onClick={() => setPage(0)}>
              <RefreshCw className="w-4 h-4 mr-2" />
              Refresh
            </Button>
          </div>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
          <Card>
            <CardContent className="pt-5">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Total Events</p>
                  <p className="text-2xl font-bold">{filteredLogs.length}</p>
                </div>
                <div className="p-2.5 bg-slate-100 rounded-lg">
                  <ClipboardList className="w-5 h-5 text-slate-600" />
                </div>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-5">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Success</p>
                  <p className="text-2xl font-bold text-emerald-600">{successCount}</p>
                </div>
                <div className="p-2.5 bg-emerald-50 rounded-lg">
                  <CheckCircle className="w-5 h-5 text-emerald-600" />
                </div>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-5">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Failures</p>
                  <p className="text-2xl font-bold text-red-600">{failureCount}</p>
                </div>
                <div className="p-2.5 bg-red-50 rounded-lg">
                  <XCircle className="w-5 h-5 text-red-600" />
                </div>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-5">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Warnings</p>
                  <p className="text-2xl font-bold text-amber-600">{warningCount}</p>
                </div>
                <div className="p-2.5 bg-amber-50 rounded-lg">
                  <AlertTriangle className="w-5 h-5 text-amber-600" />
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader className="pb-4">
            <div className="flex flex-col sm:flex-row items-start sm:items-center gap-4 flex-wrap">
              <div className="relative flex-1 min-w-[200px] max-w-sm">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                <Input
                  placeholder="Search actor, action, target, IP..."
                  value={search}
                  onChange={(e) => { setSearch(e.target.value); setPage(0) }}
                  className="pl-9"
                />
              </div>
              <div className="flex items-center gap-2 flex-wrap">
                <Filter className="w-4 h-4 text-muted-foreground" />
                <Select value={sourceFilter} onValueChange={(v) => { setSourceFilter(v); setPage(0) }}>
                  <SelectTrigger className="w-[130px]">
                    <SelectValue placeholder="Source" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Sources</SelectItem>
                    <SelectItem value="keycloak">Keycloak</SelectItem>
                    <SelectItem value="openvpn">OpenVPN</SelectItem>
                    <SelectItem value="system">System</SelectItem>
                  </SelectContent>
                </Select>
                <Select value={statusFilter} onValueChange={(v) => { setStatusFilter(v); setPage(0) }}>
                  <SelectTrigger className="w-[120px]">
                    <SelectValue placeholder="Status" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Status</SelectItem>
                    <SelectItem value="success">Success</SelectItem>
                    <SelectItem value="failure">Failure</SelectItem>
                    <SelectItem value="warning">Warning</SelectItem>
                  </SelectContent>
                </Select>
                <Select value={actionTypeFilter} onValueChange={(v) => { setActionTypeFilter(v); setPage(0) }}>
                  <SelectTrigger className="w-[130px]">
                    <SelectValue placeholder="Action Type" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Actions</SelectItem>
                    <SelectItem value="create">Create</SelectItem>
                    <SelectItem value="update">Update</SelectItem>
                    <SelectItem value="delete">Delete</SelectItem>
                    <SelectItem value="login">Login</SelectItem>
                    <SelectItem value="logout">Logout</SelectItem>
                    <SelectItem value="assign">Assign</SelectItem>
                    <SelectItem value="revoke">Revoke</SelectItem>
                    <SelectItem value="enable">Enable</SelectItem>
                    <SelectItem value="disable">Disable</SelectItem>
                    <SelectItem value="error">Error</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="text-sm text-muted-foreground ml-auto">
                {filteredLogs.length} events
              </div>
            </div>
          </CardHeader>
          <CardContent>
            {paginatedLogs.length === 0 ? (
              <div className="text-center py-12 text-muted-foreground">
                <ClipboardList className="w-12 h-12 mx-auto mb-3 text-muted-foreground/40" />
                <p>No audit events found</p>
                <p className="text-sm">Try adjusting your filters</p>
              </div>
            ) : (
              <>
                <div className="overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Timestamp</TableHead>
                        <TableHead>Actor</TableHead>
                        <TableHead>Action</TableHead>
                        <TableHead>Target</TableHead>
                        <TableHead>Source</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead>IP Address</TableHead>
                        <TableHead className="w-12"></TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {paginatedLogs.map((log) => {
                        const ActionIcon = ACTION_TYPE_ICONS[log.actionType]
                        const TargetIcon = TARGET_ICONS[log.targetType]
                        return (
                          <TableRow key={log.id} className="cursor-pointer hover:bg-muted/50" onClick={() => setSelectedLog(log)}>
                            <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
                              <div>{format(new Date(log.timestamp), "MMM d, HH:mm:ss")}</div>
                              <div className="text-xs text-muted-foreground/60">
                                {formatDistanceToNow(new Date(log.timestamp), { addSuffix: true })}
                              </div>
                            </TableCell>
                            <TableCell>
                              <div className="flex items-center gap-2">
                                <div className={`p-1 rounded ${log.actorType === "admin" ? "bg-emerald-50" : "bg-slate-100"}`}>
                                  <User className={`w-3 h-3 ${log.actorType === "admin" ? "text-emerald-600" : "text-slate-500"}`} />
                                </div>
                                <span className="text-sm font-medium">{log.actor}</span>
                              </div>
                            </TableCell>
                            <TableCell>
                              <div className="flex items-center gap-2">
                                <div className={`p-1 rounded ${ACTION_TYPE_COLORS[log.actionType]}`}>
                                  <ActionIcon className="w-3 h-3" />
                                </div>
                                <span className="text-sm">{log.action}</span>
                              </div>
                            </TableCell>
                            <TableCell>
                              <div className="flex items-center gap-2">
                                <TargetIcon className="w-3.5 h-3.5 text-muted-foreground" />
                                <span className="text-sm font-mono text-foreground">{log.target}</span>
                              </div>
                            </TableCell>
                            <TableCell>{getSourceBadge(log.source)}</TableCell>
                            <TableCell>{getStatusBadge(log.status)}</TableCell>
                            <TableCell className="text-sm font-mono text-muted-foreground">
                              {log.ipAddress || "-"}
                            </TableCell>
                            <TableCell>
                              <Button variant="ghost" size="icon" className="h-8 w-8">
                                <Eye className="w-4 h-4" />
                              </Button>
                            </TableCell>
                          </TableRow>
                        )
                      })}
                    </TableBody>
                  </Table>
                </div>

                {totalPages > 1 && (
                  <div className="flex items-center justify-between mt-4 pt-4 border-t">
                    <div className="text-sm text-muted-foreground">
                      Showing {page * pageSize + 1}–{Math.min((page + 1) * pageSize, filteredLogs.length)} of {filteredLogs.length}
                    </div>
                    <div className="flex items-center gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setPage((p) => Math.max(0, p - 1))}
                        disabled={page === 0}
                      >
                        <ChevronLeft className="w-4 h-4 mr-1" />
                        Previous
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
                        disabled={page >= totalPages - 1}
                      >
                        Next
                        <ChevronRight className="w-4 h-4 ml-1" />
                      </Button>
                    </div>
                  </div>
                )}
              </>
            )}
          </CardContent>
        </Card>

        {/* Log Detail Dialog */}
        <Dialog open={!!selectedLog} onOpenChange={() => setSelectedLog(null)}>
          <DialogContent className="sm:max-w-lg">
            <DialogHeader>
              <DialogTitle>Event Details</DialogTitle>
            </DialogHeader>
            {selectedLog && (
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">Timestamp</p>
                    <p className="text-sm font-medium">{format(new Date(selectedLog.timestamp), "PPpp")}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">Status</p>
                    <div>{getStatusBadge(selectedLog.status)}</div>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">Actor</p>
                    <p className="text-sm font-medium">{selectedLog.actor}</p>
                    <p className="text-xs text-muted-foreground capitalize">{selectedLog.actorType}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">Source</p>
                    <div>{getSourceBadge(selectedLog.source)}</div>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">Action</p>
                    <p className="text-sm font-medium">{selectedLog.action}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">Target</p>
                    <p className="text-sm font-mono font-medium">{selectedLog.target}</p>
                    <p className="text-xs text-muted-foreground capitalize">{selectedLog.targetType}</p>
                  </div>
                  {selectedLog.ipAddress && (
                    <div>
                      <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">IP Address</p>
                      <p className="text-sm font-mono">{selectedLog.ipAddress}</p>
                    </div>
                  )}
                </div>
                {selectedLog.details && (
                  <div>
                    <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">Details</p>
                    <div className="p-3 bg-muted rounded-lg text-sm text-foreground">
                      {selectedLog.details}
                    </div>
                  </div>
                )}
              </div>
            )}
          </DialogContent>
        </Dialog>
      </div>
    </DashboardLayout>
  )
}
