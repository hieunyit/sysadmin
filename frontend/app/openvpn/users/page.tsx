"use client"

import { useState, useMemo } from "react"
import useSWR, { mutate } from "swr"
import { DashboardLayout } from "@/components/dashboard-layout"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardHeader } from "@/components/ui/card"
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
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { FieldGroup, Field, FieldLabel } from "@/components/ui/field"
import { Switch } from "@/components/ui/switch"
import { Spinner } from "@/components/ui/spinner"
import {
  Search,
  Plus,
  MoreHorizontal,
  Network,
  FileKey,
  Trash2,
  Power,
  PowerOff,
  Clock,
  Globe,
  ChevronLeft,
  ChevronRight,
  Filter,
  Download,
  RefreshCw,
  Edit,
} from "lucide-react"
import { formatDistanceToNow } from "date-fns"

const fetcher = (url: string) => fetch(url).then((res) => res.json())

interface VPNUser {
  id: string
  username: string
  email?: string
  enabled: boolean
  createdAt: string
  lastLogin?: string
  status: "active" | "inactive" | "suspended"
  assignedIP?: string
}

export default function OpenVPNUsersPage() {
  const [search, setSearch] = useState("")
  const [statusFilter, setStatusFilter] = useState<string>("all")
  const [page, setPage] = useState(0)
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [isEditOpen, setIsEditOpen] = useState(false)
  const [selectedUser, setSelectedUser] = useState<VPNUser | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const pageSize = 10

  const { data, error, isLoading: isLoadingUsers, mutate: refreshUsers } = useSWR(
    "/api/openvpn/users",
    fetcher,
    { refreshInterval: 30000 }
  )

  const allUsers: VPNUser[] = data?.users || []
  
  const filteredUsers = useMemo(() => {
    let users = [...allUsers]
    
    // Apply search filter
    if (search) {
      const searchLower = search.toLowerCase()
      users = users.filter(
        (u) =>
          u.username.toLowerCase().includes(searchLower) ||
          u.email?.toLowerCase().includes(searchLower) ||
          u.assignedIP?.includes(search)
      )
    }
    
    // Apply status filter
    if (statusFilter !== "all") {
      users = users.filter((u) => {
        if (statusFilter === "enabled") return u.enabled
        if (statusFilter === "disabled") return !u.enabled
        return u.status === statusFilter
      })
    }
    
    return users
  }, [allUsers, search, statusFilter])

  const totalPages = Math.ceil(filteredUsers.length / pageSize)
  const paginatedUsers = filteredUsers.slice(page * pageSize, (page + 1) * pageSize)

  const handleCreateUser = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setIsLoading(true)
    const formData = new FormData(e.currentTarget)
    
    try {
      const res = await fetch("/api/openvpn/users", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          action: "create",
          username: formData.get("username"),
          email: formData.get("email"),
          enabled: formData.get("enabled") === "on",
        }),
      })
      
      if (res.ok) {
        setIsCreateOpen(false)
        refreshUsers()
      }
    } finally {
      setIsLoading(false)
    }
  }

  const handleEditUser = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (!selectedUser) return
    setIsLoading(true)
    const formData = new FormData(e.currentTarget)
    
    try {
      const res = await fetch("/api/openvpn/users", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          action: "update",
          id: selectedUser.id,
          email: formData.get("email"),
          enabled: formData.get("enabled") === "on",
        }),
      })
      
      if (res.ok) {
        setIsEditOpen(false)
        setSelectedUser(null)
        refreshUsers()
      }
    } finally {
      setIsLoading(false)
    }
  }

  const handleToggleEnabled = async (user: VPNUser) => {
    const action = user.enabled ? "disable" : "enable"
    await fetch("/api/openvpn/users", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action, id: user.id }),
    })
    refreshUsers()
  }

  const handleDeleteUser = async (user: VPNUser) => {
    if (!confirm(`Are you sure you want to delete VPN user "${user.username}"?`)) return
    
    await fetch("/api/openvpn/users", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action: "delete", id: user.id }),
    })
    refreshUsers()
  }

  const getStatusColor = (user: VPNUser) => {
    if (!user.enabled) return "bg-slate-100 text-slate-600"
    switch (user.status) {
      case "active":
        return "bg-emerald-100 text-emerald-700"
      case "inactive":
        return "bg-amber-100 text-amber-700"
      case "suspended":
        return "bg-red-100 text-red-700"
      default:
        return "bg-slate-100 text-slate-600"
    }
  }

  const getStatusLabel = (user: VPNUser) => {
    if (!user.enabled) return "Disabled"
    return user.status.charAt(0).toUpperCase() + user.status.slice(1)
  }

  const getInitials = (username: string) => {
    return username.slice(0, 2).toUpperCase()
  }

  const exportUsers = () => {
    const csv = [
      ["Username", "Email", "Status", "Assigned IP", "Last Login", "Created"],
      ...filteredUsers.map((u) => [
        u.username,
        u.email || "",
        getStatusLabel(u),
        u.assignedIP || "",
        u.lastLogin ? new Date(u.lastLogin).toLocaleString() : "Never",
        new Date(u.createdAt).toLocaleDateString(),
      ]),
    ]
      .map((row) => row.join(","))
      .join("\n")

    const blob = new Blob([csv], { type: "text/csv" })
    const url = URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = url
    a.download = "openvpn-users.csv"
    a.click()
  }

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-foreground">VPN Users</h1>
            <p className="text-muted-foreground">Manage OpenVPN user accounts</p>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={exportUsers}>
              <Download className="w-4 h-4 mr-2" />
              Export
            </Button>
            <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
              <DialogTrigger asChild>
                <Button className="bg-teal-600 hover:bg-teal-700">
                  <Plus className="w-4 h-4 mr-2" />
                  Add VPN User
                </Button>
              </DialogTrigger>
              <DialogContent className="sm:max-w-md">
                <DialogHeader>
                  <DialogTitle>Create VPN User</DialogTitle>
                  <DialogDescription>
                    Add a new user to OpenVPN Access Server
                  </DialogDescription>
                </DialogHeader>
                <form onSubmit={handleCreateUser}>
                  <FieldGroup className="space-y-4 py-4">
                    <Field>
                      <FieldLabel>Username</FieldLabel>
                      <Input name="username" required placeholder="vpnuser" />
                    </Field>
                    <Field>
                      <FieldLabel>Email (optional)</FieldLabel>
                      <Input name="email" type="email" placeholder="user@example.com" />
                    </Field>
                    <div className="flex items-center justify-between">
                      <FieldLabel>Enabled</FieldLabel>
                      <Switch name="enabled" defaultChecked />
                    </div>
                  </FieldGroup>
                  <DialogFooter>
                    <Button type="button" variant="outline" onClick={() => setIsCreateOpen(false)}>
                      Cancel
                    </Button>
                    <Button type="submit" disabled={isLoading} className="bg-teal-600 hover:bg-teal-700">
                      {isLoading ? <Spinner className="w-4 h-4 mr-2" /> : null}
                      Create User
                    </Button>
                  </DialogFooter>
                </form>
              </DialogContent>
            </Dialog>
          </div>
        </div>

        {/* Stats Cards */}
        <div className="grid grid-cols-1 sm:grid-cols-4 gap-4">
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Total Users</p>
                  <p className="text-2xl font-bold">{allUsers.length}</p>
                </div>
                <div className="p-3 bg-teal-100 rounded-lg">
                  <Network className="w-5 h-5 text-teal-600" />
                </div>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Active</p>
                  <p className="text-2xl font-bold text-emerald-600">
                    {allUsers.filter((u) => u.enabled && u.status === "active").length}
                  </p>
                </div>
                <div className="p-3 bg-emerald-100 rounded-lg">
                  <Power className="w-5 h-5 text-emerald-600" />
                </div>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Inactive</p>
                  <p className="text-2xl font-bold text-amber-600">
                    {allUsers.filter((u) => u.enabled && u.status === "inactive").length}
                  </p>
                </div>
                <div className="p-3 bg-amber-100 rounded-lg">
                  <Clock className="w-5 h-5 text-amber-600" />
                </div>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Disabled</p>
                  <p className="text-2xl font-bold text-slate-600">
                    {allUsers.filter((u) => !u.enabled).length}
                  </p>
                </div>
                <div className="p-3 bg-slate-100 rounded-lg">
                  <PowerOff className="w-5 h-5 text-slate-600" />
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader className="pb-4">
            <div className="flex flex-col sm:flex-row items-start sm:items-center gap-4">
              <div className="relative flex-1 max-w-sm">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                <Input
                  placeholder="Search users or IP..."
                  value={search}
                  onChange={(e) => {
                    setSearch(e.target.value)
                    setPage(0)
                  }}
                  className="pl-9"
                />
              </div>
              <div className="flex items-center gap-2">
                <Filter className="w-4 h-4 text-muted-foreground" />
                <Select value={statusFilter} onValueChange={(v) => { setStatusFilter(v); setPage(0) }}>
                  <SelectTrigger className="w-[140px]">
                    <SelectValue placeholder="Filter status" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Users</SelectItem>
                    <SelectItem value="enabled">Enabled</SelectItem>
                    <SelectItem value="disabled">Disabled</SelectItem>
                    <SelectItem value="active">Active</SelectItem>
                    <SelectItem value="inactive">Inactive</SelectItem>
                    <SelectItem value="suspended">Suspended</SelectItem>
                  </SelectContent>
                </Select>
                <Button variant="ghost" size="icon" onClick={() => refreshUsers()}>
                  <RefreshCw className="w-4 h-4" />
                </Button>
              </div>
              <div className="text-sm text-muted-foreground">
                {filteredUsers.length} users
              </div>
            </div>
          </CardHeader>
          <CardContent>
            {isLoadingUsers ? (
              <div className="flex items-center justify-center py-12">
                <Spinner className="w-6 h-6 text-teal-600" />
              </div>
            ) : error ? (
              <div className="text-center py-12 text-red-500">
                Failed to load VPN users. Please check your OpenVPN connection.
              </div>
            ) : paginatedUsers.length === 0 ? (
              <div className="text-center py-12 text-muted-foreground">
                <Network className="w-12 h-12 mx-auto mb-3 text-muted-foreground/50" />
                <p>No VPN users found</p>
                <p className="text-sm">Create your first VPN user to get started</p>
              </div>
            ) : (
              <>
                <div className="overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>User</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead>Assigned IP</TableHead>
                        <TableHead>Last Login</TableHead>
                        <TableHead>Created</TableHead>
                        <TableHead className="w-12"></TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {paginatedUsers.map((user) => (
                        <TableRow key={user.id}>
                          <TableCell>
                            <div className="flex items-center gap-3">
                              <Avatar className="w-9 h-9">
                                <AvatarFallback className="bg-teal-100 text-teal-700 text-xs font-medium">
                                  {getInitials(user.username)}
                                </AvatarFallback>
                              </Avatar>
                              <div>
                                <div className="font-medium text-foreground">{user.username}</div>
                                {user.email && (
                                  <div className="text-sm text-muted-foreground">{user.email}</div>
                                )}
                              </div>
                            </div>
                          </TableCell>
                          <TableCell>
                            <Badge className={getStatusColor(user)}>
                              {getStatusLabel(user)}
                            </Badge>
                          </TableCell>
                          <TableCell>
                            {user.assignedIP ? (
                              <div className="flex items-center gap-1.5 text-foreground font-mono text-sm">
                                <Globe className="w-3.5 h-3.5 text-muted-foreground" />
                                {user.assignedIP}
                              </div>
                            ) : (
                              <span className="text-muted-foreground/50">-</span>
                            )}
                          </TableCell>
                          <TableCell>
                            {user.lastLogin ? (
                              <div className="flex items-center gap-1.5 text-muted-foreground text-sm">
                                <Clock className="w-3.5 h-3.5" />
                                {formatDistanceToNow(new Date(user.lastLogin), { addSuffix: true })}
                              </div>
                            ) : (
                              <span className="text-muted-foreground/50">Never</span>
                            )}
                          </TableCell>
                          <TableCell className="text-muted-foreground text-sm">
                            {new Date(user.createdAt).toLocaleDateString()}
                          </TableCell>
                          <TableCell>
                            <DropdownMenu>
                              <DropdownMenuTrigger asChild>
                                <Button variant="ghost" size="icon" className="h-8 w-8">
                                  <MoreHorizontal className="w-4 h-4" />
                                </Button>
                              </DropdownMenuTrigger>
                              <DropdownMenuContent align="end">
                                <DropdownMenuItem
                                  onClick={() => {
                                    setSelectedUser(user)
                                    setIsEditOpen(true)
                                  }}
                                >
                                  <Edit className="w-4 h-4 mr-2" />
                                  Edit User
                                </DropdownMenuItem>
                                <DropdownMenuItem>
                                  <FileKey className="w-4 h-4 mr-2" />
                                  Generate Config
                                </DropdownMenuItem>
                                <DropdownMenuSeparator />
                                <DropdownMenuItem onClick={() => handleToggleEnabled(user)}>
                                  {user.enabled ? (
                                    <>
                                      <PowerOff className="w-4 h-4 mr-2" />
                                      Disable User
                                    </>
                                  ) : (
                                    <>
                                      <Power className="w-4 h-4 mr-2" />
                                      Enable User
                                    </>
                                  )}
                                </DropdownMenuItem>
                                <DropdownMenuItem
                                  onClick={() => handleDeleteUser(user)}
                                  className="text-red-600"
                                >
                                  <Trash2 className="w-4 h-4 mr-2" />
                                  Delete User
                                </DropdownMenuItem>
                              </DropdownMenuContent>
                            </DropdownMenu>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>

                {totalPages > 1 && (
                  <div className="flex items-center justify-between mt-4 pt-4 border-t">
                    <div className="text-sm text-muted-foreground">
                      Showing {page * pageSize + 1} - {Math.min((page + 1) * pageSize, filteredUsers.length)} of {filteredUsers.length}
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

        {/* Edit User Dialog */}
        <Dialog open={isEditOpen} onOpenChange={setIsEditOpen}>
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <DialogTitle>Edit VPN User</DialogTitle>
              <DialogDescription>
                Update information for {selectedUser?.username}
              </DialogDescription>
            </DialogHeader>
            <form onSubmit={handleEditUser}>
              <FieldGroup className="space-y-4 py-4">
                <Field>
                  <FieldLabel>Username</FieldLabel>
                  <Input value={selectedUser?.username || ""} disabled className="bg-muted" />
                </Field>
                <Field>
                  <FieldLabel>Email</FieldLabel>
                  <Input name="email" type="email" defaultValue={selectedUser?.email || ""} />
                </Field>
                <Field>
                  <FieldLabel>Assigned IP</FieldLabel>
                  <Input value={selectedUser?.assignedIP || "Auto-assigned"} disabled className="bg-muted" />
                </Field>
                <div className="flex items-center justify-between">
                  <FieldLabel>Enabled</FieldLabel>
                  <Switch name="enabled" defaultChecked={selectedUser?.enabled} />
                </div>
              </FieldGroup>
              <DialogFooter>
                <Button type="button" variant="outline" onClick={() => setIsEditOpen(false)}>
                  Cancel
                </Button>
                <Button type="submit" disabled={isLoading} className="bg-teal-600 hover:bg-teal-700">
                  {isLoading ? <Spinner className="w-4 h-4 mr-2" /> : null}
                  Save Changes
                </Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </div>
    </DashboardLayout>
  )
}
