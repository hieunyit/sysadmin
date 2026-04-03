"use client"

import { useState, useMemo } from "react"
import useSWR from "swr"
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
import { FieldGroup, Field, FieldLabel } from "@/components/ui/field"
import { Switch } from "@/components/ui/switch"
import { Spinner } from "@/components/ui/spinner"
import { Textarea } from "@/components/ui/textarea"
import { ScrollArea } from "@/components/ui/scroll-area"
import {
  Search,
  Plus,
  MoreHorizontal,
  Folder,
  Users,
  Trash2,
  Edit,
  RefreshCw,
  Settings,
  Network,
  XCircle,
} from "lucide-react"

const fetcher = (url: string) => fetch(url).then((res) => res.json())

interface VPNGroup {
  id: string
  name: string
  description?: string
  userCount?: number
  prop_autologin?: boolean
  prop_deny?: boolean
  group_subnets?: string[]
  access_from?: string[]
  access_to?: string[]
}

interface VPNUser {
  id: string
  username: string
  email?: string
  enabled: boolean
  status: string
}

export default function OpenVPNGroupsPage() {
  const [search, setSearch] = useState("")
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [isEditOpen, setIsEditOpen] = useState(false)
  const [isPropsOpen, setIsPropsOpen] = useState(false)
  const [isMembersOpen, setIsMembersOpen] = useState(false)
  const [selectedGroup, setSelectedGroup] = useState<VPNGroup | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [loadingData, setLoadingData] = useState(false)
  const [groupProps, setGroupProps] = useState<Partial<VPNGroup>>({})
  const [groupMembers, setGroupMembers] = useState<VPNUser[]>([])

  const { data, error, isLoading: isLoadingGroups, mutate: refreshGroups } = useSWR(
    "/api/openvpn/groups",
    fetcher,
    { refreshInterval: 30000 }
  )

  const groups: VPNGroup[] = data?.groups || []

  const filteredGroups = useMemo(() => {
    if (!search) return groups
    const searchLower = search.toLowerCase()
    return groups.filter(
      (g) =>
        g.name.toLowerCase().includes(searchLower) ||
        g.description?.toLowerCase().includes(searchLower)
    )
  }, [groups, search])

  const handleCreateGroup = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setIsLoading(true)
    const formData = new FormData(e.currentTarget)

    try {
      const res = await fetch("/api/openvpn/groups", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          action: "create",
          name: formData.get("name"),
          description: formData.get("description"),
          prop_autologin: formData.get("autologin") === "on",
        }),
      })

      if (res.ok) {
        setIsCreateOpen(false)
        refreshGroups()
      }
    } finally {
      setIsLoading(false)
    }
  }

  const handleEditGroup = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (!selectedGroup) return
    setIsLoading(true)
    const formData = new FormData(e.currentTarget)

    try {
      const res = await fetch("/api/openvpn/groups", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          action: "update",
          groupname: selectedGroup.name,
          description: formData.get("description"),
        }),
      })

      if (res.ok) {
        setIsEditOpen(false)
        setSelectedGroup(null)
        refreshGroups()
      }
    } finally {
      setIsLoading(false)
    }
  }

  const handleDeleteGroup = async (group: VPNGroup) => {
    if (!confirm(`Are you sure you want to delete group "${group.name}"? Users in this group will not be deleted but will no longer be assigned to this group.`)) return

    await fetch("/api/openvpn/groups", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action: "delete", groupname: group.name }),
    })
    refreshGroups()
  }

  const fetchGroupProps = async (group: VPNGroup) => {
    setLoadingData(true)
    try {
      const res = await fetch("/api/openvpn/groups", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action: "getProps", groupname: group.name }),
      })
      const data = await res.json()
      setGroupProps(data.props || {})
    } finally {
      setLoadingData(false)
    }
  }

  const fetchGroupMembers = async (group: VPNGroup) => {
    setLoadingData(true)
    try {
      const res = await fetch("/api/openvpn/groups", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action: "getMembers", groupname: group.name }),
      })
      const data = await res.json()
      setGroupMembers(data.members || [])
    } finally {
      setLoadingData(false)
    }
  }

  const handleUpdateProps = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (!selectedGroup) return
    setIsLoading(true)
    const formData = new FormData(e.currentTarget)

    try {
      await fetch("/api/openvpn/groups", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          action: "setProps",
          groupname: selectedGroup.name,
          props: {
            prop_autologin: formData.get("autologin") === "on",
            prop_deny: formData.get("deny") === "on",
          },
        }),
      })
      setIsPropsOpen(false)
      refreshGroups()
    } finally {
      setIsLoading(false)
    }
  }

  const handleRemoveUserFromGroup = async (user: VPNUser) => {
    if (!selectedGroup) return
    setIsLoading(true)
    try {
      await fetch("/api/openvpn/users", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          action: "setProps",
          username: user.username,
          props: { group: null },
        }),
      })
      setGroupMembers(groupMembers.filter((m) => m.id !== user.id))
      refreshGroups()
    } finally {
      setIsLoading(false)
    }
  }

  const getInitials = (username: string) => {
    return username.slice(0, 2).toUpperCase()
  }

  const totalUsers = groups.reduce((acc, g) => acc + (g.userCount || 0), 0)

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-foreground">VPN Groups</h1>
            <p className="text-muted-foreground">Manage OpenVPN Access Server user groups</p>
          </div>
          <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
            <DialogTrigger asChild>
              <Button className="bg-teal-600 hover:bg-teal-700">
                <Plus className="w-4 h-4 mr-2" />
                Add Group
              </Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-md">
              <DialogHeader>
                <DialogTitle>Create VPN Group</DialogTitle>
                <DialogDescription>
                  Add a new group to OpenVPN Access Server
                </DialogDescription>
              </DialogHeader>
              <form onSubmit={handleCreateGroup}>
                <FieldGroup className="space-y-4 py-4">
                  <Field>
                    <FieldLabel>Group Name</FieldLabel>
                    <Input name="name" required placeholder="e.g., developers" />
                  </Field>
                  <Field>
                    <FieldLabel>Description (optional)</FieldLabel>
                    <Textarea name="description" placeholder="Group description..." rows={2} />
                  </Field>
                  <div className="flex items-center justify-between">
                    <div>
                      <FieldLabel>Allow Auto-login</FieldLabel>
                      <p className="text-xs text-muted-foreground">Members can download auto-login profiles</p>
                    </div>
                    <Switch name="autologin" />
                  </div>
                </FieldGroup>
                <DialogFooter>
                  <Button type="button" variant="outline" onClick={() => setIsCreateOpen(false)}>
                    Cancel
                  </Button>
                  <Button type="submit" disabled={isLoading} className="bg-teal-600 hover:bg-teal-700">
                    {isLoading ? <Spinner className="w-4 h-4 mr-2" /> : null}
                    Create Group
                  </Button>
                </DialogFooter>
              </form>
            </DialogContent>
          </Dialog>
        </div>

        {/* Stats Cards */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Total Groups</p>
                  <p className="text-2xl font-bold">{groups.length}</p>
                </div>
                <div className="p-3 bg-teal-100 rounded-lg">
                  <Folder className="w-5 h-5 text-teal-600" />
                </div>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Total Members</p>
                  <p className="text-2xl font-bold">{totalUsers}</p>
                </div>
                <div className="p-3 bg-emerald-100 rounded-lg">
                  <Users className="w-5 h-5 text-emerald-600" />
                </div>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Auto-login Enabled</p>
                  <p className="text-2xl font-bold">
                    {groups.filter((g) => g.prop_autologin).length}
                  </p>
                </div>
                <div className="p-3 bg-amber-100 rounded-lg">
                  <Network className="w-5 h-5 text-amber-600" />
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
                  placeholder="Search groups..."
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  className="pl-9"
                />
              </div>
              <div className="flex items-center gap-2">
                <Button variant="ghost" size="icon" onClick={() => refreshGroups()}>
                  <RefreshCw className="w-4 h-4" />
                </Button>
                <div className="text-sm text-muted-foreground">
                  {filteredGroups.length} groups
                </div>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            {isLoadingGroups ? (
              <div className="flex items-center justify-center py-12">
                <Spinner className="w-6 h-6 text-teal-600" />
              </div>
            ) : error ? (
              <div className="text-center py-12 text-red-500">
                Failed to load VPN groups. Please check your OpenVPN connection.
              </div>
            ) : filteredGroups.length === 0 ? (
              <div className="text-center py-12 text-muted-foreground">
                <Folder className="w-12 h-12 mx-auto mb-3 text-muted-foreground/50" />
                <p>No VPN groups found</p>
                <p className="text-sm">Create your first group to organize users</p>
              </div>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Group</TableHead>
                    <TableHead>Members</TableHead>
                    <TableHead>Auto-login</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead className="w-12"></TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredGroups.map((group) => (
                    <TableRow key={group.id}>
                      <TableCell>
                        <div className="flex items-center gap-3">
                          <div className="p-2 bg-amber-100 rounded-lg">
                            <Folder className="w-5 h-5 text-amber-600" />
                          </div>
                          <div>
                            <div className="font-medium text-foreground">{group.name}</div>
                            {group.description && (
                              <div className="text-sm text-muted-foreground truncate max-w-[200px]">
                                {group.description}
                              </div>
                            )}
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge variant="secondary">
                          <Users className="w-3 h-3 mr-1" />
                          {group.userCount || 0} users
                        </Badge>
                      </TableCell>
                      <TableCell>
                        {group.prop_autologin ? (
                          <Badge className="bg-emerald-100 text-emerald-700">Enabled</Badge>
                        ) : (
                          <Badge variant="secondary">Disabled</Badge>
                        )}
                      </TableCell>
                      <TableCell>
                        {group.prop_deny ? (
                          <Badge className="bg-red-100 text-red-700">Blocked</Badge>
                        ) : (
                          <Badge className="bg-emerald-100 text-emerald-700">Active</Badge>
                        )}
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
                                setSelectedGroup(group)
                                fetchGroupMembers(group)
                                setIsMembersOpen(true)
                              }}
                            >
                              <Users className="w-4 h-4 mr-2" />
                              View Members
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              onClick={() => {
                                setSelectedGroup(group)
                                fetchGroupProps(group)
                                setIsPropsOpen(true)
                              }}
                            >
                              <Settings className="w-4 h-4 mr-2" />
                              Group Properties
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              onClick={() => {
                                setSelectedGroup(group)
                                setIsEditOpen(true)
                              }}
                            >
                              <Edit className="w-4 h-4 mr-2" />
                              Edit Group
                            </DropdownMenuItem>
                            <DropdownMenuSeparator />
                            <DropdownMenuItem
                              onClick={() => handleDeleteGroup(group)}
                              className="text-red-600"
                            >
                              <Trash2 className="w-4 h-4 mr-2" />
                              Delete Group
                            </DropdownMenuItem>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>

        {/* Edit Group Dialog */}
        <Dialog open={isEditOpen} onOpenChange={setIsEditOpen}>
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <DialogTitle>Edit Group</DialogTitle>
              <DialogDescription>
                Update group: {selectedGroup?.name}
              </DialogDescription>
            </DialogHeader>
            <form onSubmit={handleEditGroup}>
              <FieldGroup className="space-y-4 py-4">
                <Field>
                  <FieldLabel>Group Name</FieldLabel>
                  <Input value={selectedGroup?.name || ""} disabled className="bg-muted" />
                </Field>
                <Field>
                  <FieldLabel>Description</FieldLabel>
                  <Textarea
                    name="description"
                    defaultValue={selectedGroup?.description || ""}
                    placeholder="Group description..."
                    rows={2}
                  />
                </Field>
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

        {/* Group Properties Dialog */}
        <Dialog open={isPropsOpen} onOpenChange={setIsPropsOpen}>
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <DialogTitle>Group Properties</DialogTitle>
              <DialogDescription>
                Configure VPN properties for {selectedGroup?.name}
              </DialogDescription>
            </DialogHeader>
            {loadingData ? (
              <div className="flex items-center justify-center py-8">
                <Spinner className="w-6 h-6 text-teal-600" />
              </div>
            ) : (
              <form onSubmit={handleUpdateProps}>
                <FieldGroup className="space-y-4 py-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <FieldLabel>Auto-login</FieldLabel>
                      <p className="text-xs text-muted-foreground">Allow auto-login profile generation for members</p>
                    </div>
                    <Switch name="autologin" defaultChecked={groupProps.prop_autologin} />
                  </div>
                  <div className="flex items-center justify-between">
                    <div>
                      <FieldLabel>Deny Access</FieldLabel>
                      <p className="text-xs text-muted-foreground">Block VPN connections for all members</p>
                    </div>
                    <Switch name="deny" defaultChecked={groupProps.prop_deny} />
                  </div>
                </FieldGroup>
                <DialogFooter>
                  <Button type="button" variant="outline" onClick={() => setIsPropsOpen(false)}>
                    Cancel
                  </Button>
                  <Button type="submit" disabled={isLoading} className="bg-teal-600 hover:bg-teal-700">
                    {isLoading ? <Spinner className="w-4 h-4 mr-2" /> : null}
                    Save Properties
                  </Button>
                </DialogFooter>
              </form>
            )}
          </DialogContent>
        </Dialog>

        {/* Members Dialog */}
        <Dialog open={isMembersOpen} onOpenChange={setIsMembersOpen}>
          <DialogContent className="sm:max-w-2xl">
            <DialogHeader>
              <DialogTitle>Group Members</DialogTitle>
              <DialogDescription>
                Members of {selectedGroup?.name}
              </DialogDescription>
            </DialogHeader>
            <div className="py-4">
              {loadingData ? (
                <div className="flex items-center justify-center py-8">
                  <Spinner className="w-6 h-6 text-teal-600" />
                </div>
              ) : groupMembers.length === 0 ? (
                <div className="text-center py-8 text-muted-foreground">
                  <Users className="w-12 h-12 mx-auto mb-3 text-muted-foreground/50" />
                  <p>No members in this group</p>
                  <p className="text-sm">Assign users to this group from the Users page</p>
                </div>
              ) : (
                <ScrollArea className="h-64">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>User</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead className="w-12"></TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {groupMembers.map((member) => (
                        <TableRow key={member.id}>
                          <TableCell>
                            <div className="flex items-center gap-3">
                              <Avatar className="w-8 h-8">
                                <AvatarFallback className="bg-teal-100 text-teal-700 text-xs">
                                  {getInitials(member.username)}
                                </AvatarFallback>
                              </Avatar>
                              <div>
                                <div className="font-medium">{member.username}</div>
                                {member.email && (
                                  <div className="text-sm text-muted-foreground">{member.email}</div>
                                )}
                              </div>
                            </div>
                          </TableCell>
                          <TableCell>
                            <Badge className={member.enabled ? "bg-emerald-100 text-emerald-700" : "bg-muted text-muted-foreground"}>
                              {member.enabled ? "Enabled" : "Disabled"}
                            </Badge>
                          </TableCell>
                          <TableCell>
                            <Button
                              size="sm"
                              variant="ghost"
                              onClick={() => handleRemoveUserFromGroup(member)}
                              disabled={isLoading}
                            >
                              <XCircle className="w-4 h-4 text-red-500" />
                            </Button>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </ScrollArea>
              )}
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setIsMembersOpen(false)}>
                Close
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
    </DashboardLayout>
  )
}
