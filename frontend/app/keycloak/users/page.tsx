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
  UserCog,
  Key,
  Trash2,
  Shield,
  Users as UsersIcon,
  Mail,
  CheckCircle,
  XCircle,
  ChevronLeft,
  ChevronRight,
  Filter,
  Download,
  RefreshCw,
  Edit,
} from "lucide-react"

const fetcher = (url: string) => fetch(url).then((res) => res.json())

interface User {
  id: string
  username: string
  email?: string
  firstName?: string
  lastName?: string
  enabled: boolean
  emailVerified: boolean
  createdTimestamp: number
  groups?: string[]
  roles?: string[]
}

export default function KeycloakUsersPage() {
  const [search, setSearch] = useState("")
  const [page, setPage] = useState(0)
  const [statusFilter, setStatusFilter] = useState<string>("all")
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [isEditOpen, setIsEditOpen] = useState(false)
  const [isPasswordOpen, setIsPasswordOpen] = useState(false)
  const [isRolesOpen, setIsRolesOpen] = useState(false)
  const [selectedUser, setSelectedUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const pageSize = 10

  const { data, error, isLoading: isLoadingUsers, mutate: refreshUsers } = useSWR(
    `/api/keycloak/users?search=${search}&first=${page * pageSize}&max=${pageSize}`,
    fetcher,
    { refreshInterval: 30000 }
  )

  const allUsers: User[] = data?.users || []
  
  // Apply client-side status filter
  const users = useMemo(() => {
    if (statusFilter === "all") return allUsers
    if (statusFilter === "active") return allUsers.filter((u) => u.enabled)
    if (statusFilter === "disabled") return allUsers.filter((u) => !u.enabled)
    if (statusFilter === "verified") return allUsers.filter((u) => u.emailVerified)
    if (statusFilter === "unverified") return allUsers.filter((u) => !u.emailVerified)
    return allUsers
  }, [allUsers, statusFilter])

  const total = data?.total || 0
  const totalPages = Math.ceil(total / pageSize)

  const handleCreateUser = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setIsLoading(true)
    const formData = new FormData(e.currentTarget)
    
    try {
      const res = await fetch("/api/keycloak/users", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          action: "create",
          username: formData.get("username"),
          email: formData.get("email"),
          firstName: formData.get("firstName"),
          lastName: formData.get("lastName"),
          password: formData.get("password"),
          enabled: formData.get("enabled") === "on",
          temporaryPassword: formData.get("temporaryPassword") === "on",
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
      const res = await fetch("/api/keycloak/users", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          action: "update",
          id: selectedUser.id,
          email: formData.get("email"),
          firstName: formData.get("firstName"),
          lastName: formData.get("lastName"),
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

  const handleResetPassword = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (!selectedUser) return
    setIsLoading(true)
    const formData = new FormData(e.currentTarget)
    
    try {
      const res = await fetch("/api/keycloak/users", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          action: "resetPassword",
          id: selectedUser.id,
          password: formData.get("password"),
          temporary: formData.get("temporary") === "on",
        }),
      })
      
      if (res.ok) {
        setIsPasswordOpen(false)
        setSelectedUser(null)
      }
    } finally {
      setIsLoading(false)
    }
  }

  const handleToggleEnabled = async (user: User) => {
    await fetch("/api/keycloak/users", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        action: "update",
        id: user.id,
        enabled: !user.enabled,
      }),
    })
    refreshUsers()
  }

  const handleDeleteUser = async (user: User) => {
    if (!confirm(`Are you sure you want to delete user "${user.username}"?`)) return
    
    await fetch("/api/keycloak/users", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action: "delete", id: user.id }),
    })
    refreshUsers()
  }

  const getInitials = (user: User) => {
    if (user.firstName && user.lastName) {
      return `${user.firstName[0]}${user.lastName[0]}`.toUpperCase()
    }
    return user.username?.slice(0, 2).toUpperCase() || "??"
  }

  const exportUsers = () => {
    const csv = [
      ["Username", "Email", "First Name", "Last Name", "Status", "Email Verified", "Created"],
      ...users.map((u) => [
        u.username,
        u.email || "",
        u.firstName || "",
        u.lastName || "",
        u.enabled ? "Active" : "Disabled",
        u.emailVerified ? "Yes" : "No",
        new Date(u.createdTimestamp).toLocaleDateString(),
      ]),
    ]
      .map((row) => row.join(","))
      .join("\n")

    const blob = new Blob([csv], { type: "text/csv" })
    const url = URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = url
    a.download = "keycloak-users.csv"
    a.click()
  }

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-foreground">Users</h1>
            <p className="text-muted-foreground">Manage Keycloak user accounts</p>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={exportUsers}>
              <Download className="w-4 h-4 mr-2" />
              Export
            </Button>
            <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
              <DialogTrigger asChild>
                <Button className="bg-emerald-600 hover:bg-emerald-700">
                  <Plus className="w-4 h-4 mr-2" />
                  Add User
                </Button>
              </DialogTrigger>
              <DialogContent className="sm:max-w-md">
                <DialogHeader>
                  <DialogTitle>Create New User</DialogTitle>
                  <DialogDescription>
                    Add a new user to the Keycloak realm
                  </DialogDescription>
                </DialogHeader>
                <form onSubmit={handleCreateUser}>
                  <FieldGroup className="space-y-4 py-4">
                    <Field>
                      <FieldLabel>Username</FieldLabel>
                      <Input name="username" required placeholder="johndoe" />
                    </Field>
                    <Field>
                      <FieldLabel>Email</FieldLabel>
                      <Input name="email" type="email" placeholder="john@example.com" />
                    </Field>
                    <div className="grid grid-cols-2 gap-4">
                      <Field>
                        <FieldLabel>First Name</FieldLabel>
                        <Input name="firstName" placeholder="John" />
                      </Field>
                      <Field>
                        <FieldLabel>Last Name</FieldLabel>
                        <Input name="lastName" placeholder="Doe" />
                      </Field>
                    </div>
                    <Field>
                      <FieldLabel>Initial Password</FieldLabel>
                      <Input name="password" type="password" placeholder="Enter password" />
                    </Field>
                    <div className="flex items-center justify-between">
                      <FieldLabel>Enabled</FieldLabel>
                      <Switch name="enabled" defaultChecked />
                    </div>
                    <div className="flex items-center justify-between">
                      <FieldLabel>Temporary Password</FieldLabel>
                      <Switch name="temporaryPassword" defaultChecked />
                    </div>
                  </FieldGroup>
                  <DialogFooter>
                    <Button type="button" variant="outline" onClick={() => setIsCreateOpen(false)}>
                      Cancel
                    </Button>
                    <Button type="submit" disabled={isLoading} className="bg-emerald-600 hover:bg-emerald-700">
                      {isLoading ? <Spinner className="w-4 h-4 mr-2" /> : null}
                      Create User
                    </Button>
                  </DialogFooter>
                </form>
              </DialogContent>
            </Dialog>
          </div>
        </div>

        <Card>
          <CardHeader className="pb-4">
            <div className="flex flex-col sm:flex-row items-start sm:items-center gap-4">
              <div className="relative flex-1 max-w-sm">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                <Input
                  placeholder="Search users..."
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
                <Select value={statusFilter} onValueChange={setStatusFilter}>
                  <SelectTrigger className="w-[140px]">
                    <SelectValue placeholder="Filter status" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Users</SelectItem>
                    <SelectItem value="active">Active</SelectItem>
                    <SelectItem value="disabled">Disabled</SelectItem>
                    <SelectItem value="verified">Email Verified</SelectItem>
                    <SelectItem value="unverified">Not Verified</SelectItem>
                  </SelectContent>
                </Select>
                <Button variant="ghost" size="icon" onClick={() => refreshUsers()}>
                  <RefreshCw className="w-4 h-4" />
                </Button>
              </div>
              <div className="text-sm text-muted-foreground">
                {total} users total
              </div>
            </div>
          </CardHeader>
          <CardContent>
            {isLoadingUsers ? (
              <div className="flex items-center justify-center py-12">
                <Spinner className="w-6 h-6 text-emerald-600" />
              </div>
            ) : error ? (
              <div className="text-center py-12 text-red-500">
                Failed to load users. Please check your Keycloak connection.
              </div>
            ) : users.length === 0 ? (
              <div className="text-center py-12 text-muted-foreground">
                <UsersIcon className="w-12 h-12 mx-auto mb-3 text-muted-foreground/50" />
                <p>No users found</p>
                <p className="text-sm">Try adjusting your search or filters</p>
              </div>
            ) : (
              <>
                <div className="overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>User</TableHead>
                        <TableHead>Email</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead>Email Verified</TableHead>
                        <TableHead>Groups</TableHead>
                        <TableHead>Created</TableHead>
                        <TableHead className="w-12"></TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {users.map((user) => (
                        <TableRow key={user.id}>
                          <TableCell>
                            <div className="flex items-center gap-3">
                              <Avatar className="w-9 h-9">
                                <AvatarFallback className="bg-emerald-100 text-emerald-700 text-xs font-medium">
                                  {getInitials(user)}
                                </AvatarFallback>
                              </Avatar>
                              <div>
                                <div className="font-medium text-foreground">{user.username}</div>
                                {(user.firstName || user.lastName) && (
                                  <div className="text-sm text-muted-foreground">
                                    {[user.firstName, user.lastName].filter(Boolean).join(" ")}
                                  </div>
                                )}
                              </div>
                            </div>
                          </TableCell>
                          <TableCell>
                            {user.email ? (
                              <div className="flex items-center gap-1 text-muted-foreground">
                                <Mail className="w-3.5 h-3.5" />
                                <span className="truncate max-w-[180px]">{user.email}</span>
                              </div>
                            ) : (
                              <span className="text-muted-foreground/50">-</span>
                            )}
                          </TableCell>
                          <TableCell>
                            <Badge
                              variant={user.enabled ? "default" : "secondary"}
                              className={user.enabled ? "bg-emerald-100 text-emerald-700 hover:bg-emerald-100" : ""}
                            >
                              {user.enabled ? "Active" : "Disabled"}
                            </Badge>
                          </TableCell>
                          <TableCell>
                            {user.emailVerified ? (
                              <div className="flex items-center gap-1 text-emerald-600">
                                <CheckCircle className="w-4 h-4" />
                                <span className="text-sm">Verified</span>
                              </div>
                            ) : (
                              <div className="flex items-center gap-1 text-muted-foreground">
                                <XCircle className="w-4 h-4" />
                                <span className="text-sm">Pending</span>
                              </div>
                            )}
                          </TableCell>
                          <TableCell>
                            {user.groups && user.groups.length > 0 ? (
                              <div className="flex flex-wrap gap-1">
                                {user.groups.slice(0, 2).map((group) => (
                                  <Badge key={group} variant="outline" className="text-xs">
                                    {group}
                                  </Badge>
                                ))}
                                {user.groups.length > 2 && (
                                  <Badge variant="outline" className="text-xs">
                                    +{user.groups.length - 2}
                                  </Badge>
                                )}
                              </div>
                            ) : (
                              <span className="text-muted-foreground/50">-</span>
                            )}
                          </TableCell>
                          <TableCell className="text-muted-foreground text-sm">
                            {new Date(user.createdTimestamp).toLocaleDateString()}
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
                                <DropdownMenuItem
                                  onClick={() => {
                                    setSelectedUser(user)
                                    setIsPasswordOpen(true)
                                  }}
                                >
                                  <Key className="w-4 h-4 mr-2" />
                                  Reset Password
                                </DropdownMenuItem>
                                <DropdownMenuItem
                                  onClick={() => {
                                    setSelectedUser(user)
                                    setIsRolesOpen(true)
                                  }}
                                >
                                  <Shield className="w-4 h-4 mr-2" />
                                  Manage Roles
                                </DropdownMenuItem>
                                <DropdownMenuItem>
                                  <UsersIcon className="w-4 h-4 mr-2" />
                                  Manage Groups
                                </DropdownMenuItem>
                                <DropdownMenuSeparator />
                                <DropdownMenuItem onClick={() => handleToggleEnabled(user)}>
                                  {user.enabled ? (
                                    <>
                                      <XCircle className="w-4 h-4 mr-2" />
                                      Disable User
                                    </>
                                  ) : (
                                    <>
                                      <CheckCircle className="w-4 h-4 mr-2" />
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
                      Showing {page * pageSize + 1} - {Math.min((page + 1) * pageSize, total)} of {total}
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
                      <div className="flex items-center gap-1">
                        {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                          let pageNum = i
                          if (totalPages > 5) {
                            if (page < 3) pageNum = i
                            else if (page > totalPages - 4) pageNum = totalPages - 5 + i
                            else pageNum = page - 2 + i
                          }
                          return (
                            <Button
                              key={pageNum}
                              variant={page === pageNum ? "default" : "outline"}
                              size="sm"
                              className={`w-8 h-8 p-0 ${page === pageNum ? "bg-emerald-600 hover:bg-emerald-700" : ""}`}
                              onClick={() => setPage(pageNum)}
                            >
                              {pageNum + 1}
                            </Button>
                          )
                        })}
                      </div>
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
              <DialogTitle>Edit User</DialogTitle>
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
                <div className="grid grid-cols-2 gap-4">
                  <Field>
                    <FieldLabel>First Name</FieldLabel>
                    <Input name="firstName" defaultValue={selectedUser?.firstName || ""} />
                  </Field>
                  <Field>
                    <FieldLabel>Last Name</FieldLabel>
                    <Input name="lastName" defaultValue={selectedUser?.lastName || ""} />
                  </Field>
                </div>
                <div className="flex items-center justify-between">
                  <FieldLabel>Enabled</FieldLabel>
                  <Switch name="enabled" defaultChecked={selectedUser?.enabled} />
                </div>
              </FieldGroup>
              <DialogFooter>
                <Button type="button" variant="outline" onClick={() => setIsEditOpen(false)}>
                  Cancel
                </Button>
                <Button type="submit" disabled={isLoading} className="bg-emerald-600 hover:bg-emerald-700">
                  {isLoading ? <Spinner className="w-4 h-4 mr-2" /> : null}
                  Save Changes
                </Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>

        {/* Reset Password Dialog */}
        <Dialog open={isPasswordOpen} onOpenChange={setIsPasswordOpen}>
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <DialogTitle>Reset Password</DialogTitle>
              <DialogDescription>
                Set a new password for {selectedUser?.username}
              </DialogDescription>
            </DialogHeader>
            <form onSubmit={handleResetPassword}>
              <FieldGroup className="space-y-4 py-4">
                <Field>
                  <FieldLabel>New Password</FieldLabel>
                  <Input name="password" type="password" required placeholder="Enter new password" />
                </Field>
                <Field>
                  <FieldLabel>Confirm Password</FieldLabel>
                  <Input name="confirmPassword" type="password" required placeholder="Confirm password" />
                </Field>
                <div className="flex items-center justify-between">
                  <div>
                    <FieldLabel>Temporary Password</FieldLabel>
                    <p className="text-xs text-muted-foreground">User must change on next login</p>
                  </div>
                  <Switch name="temporary" defaultChecked />
                </div>
              </FieldGroup>
              <DialogFooter>
                <Button type="button" variant="outline" onClick={() => setIsPasswordOpen(false)}>
                  Cancel
                </Button>
                <Button type="submit" disabled={isLoading} className="bg-emerald-600 hover:bg-emerald-700">
                  {isLoading ? <Spinner className="w-4 h-4 mr-2" /> : null}
                  Reset Password
                </Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>

        {/* Manage Roles Dialog */}
        <Dialog open={isRolesOpen} onOpenChange={setIsRolesOpen}>
          <DialogContent className="sm:max-w-lg">
            <DialogHeader>
              <DialogTitle>Manage Roles</DialogTitle>
              <DialogDescription>
                Assign or remove roles for {selectedUser?.username}
              </DialogDescription>
            </DialogHeader>
            <div className="py-4">
              <div className="space-y-4">
                <div>
                  <h4 className="text-sm font-medium mb-2">Current Roles</h4>
                  <div className="flex flex-wrap gap-2">
                    {selectedUser?.roles?.map((role) => (
                      <Badge key={role} variant="secondary" className="gap-1">
                        {role}
                        <button
                          className="ml-1 hover:text-red-500"
                          onClick={() => {
                            // Remove role logic
                          }}
                        >
                          <XCircle className="w-3 h-3" />
                        </button>
                      </Badge>
                    )) || <span className="text-sm text-muted-foreground">No roles assigned</span>}
                  </div>
                </div>
                <div>
                  <h4 className="text-sm font-medium mb-2">Available Roles</h4>
                  <div className="flex flex-wrap gap-2">
                    {["admin", "developer", "user", "support", "tester", "viewer"]
                      .filter((r) => !selectedUser?.roles?.includes(r))
                      .map((role) => (
                        <Badge
                          key={role}
                          variant="outline"
                          className="gap-1 cursor-pointer hover:bg-emerald-50 hover:border-emerald-300"
                          onClick={() => {
                            // Add role logic
                          }}
                        >
                          <Plus className="w-3 h-3" />
                          {role}
                        </Badge>
                      ))}
                  </div>
                </div>
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setIsRolesOpen(false)}>
                Close
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
    </DashboardLayout>
  )
}
