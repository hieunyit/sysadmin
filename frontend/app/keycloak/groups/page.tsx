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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { FieldGroup, Field, FieldLabel } from "@/components/ui/field"
import { Spinner } from "@/components/ui/spinner"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Textarea } from "@/components/ui/textarea"
import {
  Plus,
  MoreHorizontal,
  Folder,
  FolderOpen,
  Users,
  Shield,
  Trash2,
  Edit,
  ChevronRight,
  ChevronDown,
  Search,
  RefreshCw,
  XCircle,
  UserPlus,
} from "lucide-react"

const fetcher = (url: string) => fetch(url).then((res) => res.json())

interface Group {
  id: string
  name: string
  path: string
  subGroups?: Group[]
}

interface User {
  id: string
  username: string
  email?: string
  firstName?: string
  lastName?: string
}

interface Role {
  id: string
  name: string
  description?: string
}

interface GroupItemProps {
  group: Group
  level: number
  onEdit: (group: Group) => void
  onDelete: (group: Group) => void
  onViewMembers: (group: Group) => void
  onManageRoles: (group: Group) => void
  onAddSubGroup: (group: Group) => void
}

function GroupItem({ group, level, onEdit, onDelete, onViewMembers, onManageRoles, onAddSubGroup }: GroupItemProps) {
  const [isExpanded, setIsExpanded] = useState(true)
  const hasSubGroups = group.subGroups && group.subGroups.length > 0

  return (
    <div>
      <div
        className="flex items-center gap-2 py-2 px-3 hover:bg-muted rounded-lg transition-colors"
        style={{ paddingLeft: `${level * 20 + 12}px` }}
      >
        {hasSubGroups ? (
          <button
            onClick={() => setIsExpanded(!isExpanded)}
            className="p-0.5 hover:bg-muted-foreground/20 rounded"
          >
            {isExpanded ? (
              <ChevronDown className="w-4 h-4 text-muted-foreground" />
            ) : (
              <ChevronRight className="w-4 h-4 text-muted-foreground" />
            )}
          </button>
        ) : (
          <div className="w-5" />
        )}
        
        {isExpanded && hasSubGroups ? (
          <FolderOpen className="w-5 h-5 text-amber-500" />
        ) : (
          <Folder className="w-5 h-5 text-muted-foreground" />
        )}
        
        <span className="flex-1 font-medium text-foreground">{group.name}</span>
        
        <Badge variant="secondary" className="text-xs">
          {group.path}
        </Badge>
        
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon" className="h-7 w-7">
              <MoreHorizontal className="w-4 h-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => onViewMembers(group)}>
              <Users className="w-4 h-4 mr-2" />
              View Members
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => onManageRoles(group)}>
              <Shield className="w-4 h-4 mr-2" />
              Manage Roles
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => onAddSubGroup(group)}>
              <Plus className="w-4 h-4 mr-2" />
              Add Sub-Group
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => onEdit(group)}>
              <Edit className="w-4 h-4 mr-2" />
              Edit Group
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={() => onDelete(group)} className="text-red-600">
              <Trash2 className="w-4 h-4 mr-2" />
              Delete Group
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      
      {isExpanded && hasSubGroups && (
        <div>
          {group.subGroups!.map((subGroup) => (
            <GroupItem
              key={subGroup.id}
              group={subGroup}
              level={level + 1}
              onEdit={onEdit}
              onDelete={onDelete}
              onViewMembers={onViewMembers}
              onManageRoles={onManageRoles}
              onAddSubGroup={onAddSubGroup}
            />
          ))}
        </div>
      )}
    </div>
  )
}

export default function KeycloakGroupsPage() {
  const [searchTerm, setSearchTerm] = useState("")
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [isEditOpen, setIsEditOpen] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [selectedGroup, setSelectedGroup] = useState<Group | null>(null)
  const [membersDialogOpen, setMembersDialogOpen] = useState(false)
  const [rolesDialogOpen, setRolesDialogOpen] = useState(false)
  const [addMemberDialogOpen, setAddMemberDialogOpen] = useState(false)
  const [parentGroup, setParentGroup] = useState<Group | null>(null)
  const [groupMembers, setGroupMembers] = useState<User[]>([])
  const [groupRoles, setGroupRoles] = useState<Role[]>([])
  const [availableRoles, setAvailableRoles] = useState<Role[]>([])
  const [availableUsers, setAvailableUsers] = useState<User[]>([])
  const [loadingData, setLoadingData] = useState(false)
  const [userSearch, setUserSearch] = useState("")

  const { data, error, isLoading: isLoadingGroups, mutate: refreshGroups } = useSWR(
    "/api/keycloak/groups",
    fetcher,
    { refreshInterval: 30000 }
  )

  const groups: Group[] = data?.groups || []

  // Filter groups based on search
  const filterGroups = (groups: Group[], term: string): Group[] => {
    if (!term) return groups
    return groups
      .map((group) => ({
        ...group,
        subGroups: group.subGroups ? filterGroups(group.subGroups, term) : [],
      }))
      .filter(
        (group) =>
          group.name.toLowerCase().includes(term.toLowerCase()) ||
          group.path.toLowerCase().includes(term.toLowerCase()) ||
          (group.subGroups && group.subGroups.length > 0)
      )
  }

  const filteredGroups = useMemo(() => filterGroups(groups, searchTerm), [groups, searchTerm])

  const handleCreateGroup = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setIsLoading(true)
    const formData = new FormData(e.currentTarget)
    
    try {
      const payload: Record<string, string | undefined> = {
        action: "create",
        name: formData.get("name") as string,
      }
      
      if (parentGroup) {
        payload.parentId = parentGroup.id
      }

      const res = await fetch("/api/keycloak/groups", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })
      
      if (res.ok) {
        setIsCreateOpen(false)
        setParentGroup(null)
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
      const res = await fetch("/api/keycloak/groups", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          action: "update",
          id: selectedGroup.id,
          name: formData.get("name"),
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

  const handleDeleteGroup = async (group: Group) => {
    if (!confirm(`Are you sure you want to delete group "${group.name}"? This will also delete all sub-groups.`)) return
    
    await fetch("/api/keycloak/groups", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action: "delete", id: group.id }),
    })
    refreshGroups()
  }

  const fetchGroupMembers = async (group: Group) => {
    setLoadingData(true)
    try {
      const res = await fetch("/api/keycloak/groups", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action: "getMembers", id: group.id }),
      })
      const data = await res.json()
      setGroupMembers(data.members || [])
    } finally {
      setLoadingData(false)
    }
  }

  const fetchGroupRoles = async (group: Group) => {
    setLoadingData(true)
    try {
      const [rolesRes, allRolesRes] = await Promise.all([
        fetch("/api/keycloak/groups", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ action: "getRoles", id: group.id }),
        }),
        fetch("/api/keycloak/roles"),
      ])
      const [rolesData, allRolesData] = await Promise.all([rolesRes.json(), allRolesRes.json()])
      setGroupRoles(rolesData.roles || [])
      setAvailableRoles(allRolesData.roles || [])
    } finally {
      setLoadingData(false)
    }
  }

  const fetchAvailableUsers = async () => {
    setLoadingData(true)
    try {
      const res = await fetch(`/api/keycloak/users?search=${userSearch}&max=20`)
      const data = await res.json()
      setAvailableUsers(data.users || [])
    } finally {
      setLoadingData(false)
    }
  }

  const handleViewMembers = async (group: Group) => {
    setSelectedGroup(group)
    setMembersDialogOpen(true)
    await fetchGroupMembers(group)
  }

  const handleManageRoles = async (group: Group) => {
    setSelectedGroup(group)
    setRolesDialogOpen(true)
    await fetchGroupRoles(group)
  }

  const handleAddSubGroup = (group: Group) => {
    setParentGroup(group)
    setIsCreateOpen(true)
  }

  const handleRemoveMember = async (user: User) => {
    if (!selectedGroup) return
    setIsLoading(true)
    try {
      await fetch("/api/keycloak/users", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          action: "removeFromGroup",
          userId: user.id,
          groupId: selectedGroup.id,
        }),
      })
      setGroupMembers(groupMembers.filter((m) => m.id !== user.id))
    } finally {
      setIsLoading(false)
    }
  }

  const handleAddMember = async (user: User) => {
    if (!selectedGroup) return
    setIsLoading(true)
    try {
      await fetch("/api/keycloak/users", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          action: "addToGroup",
          userId: user.id,
          groupId: selectedGroup.id,
        }),
      })
      setGroupMembers([...groupMembers, user])
      setAddMemberDialogOpen(false)
    } finally {
      setIsLoading(false)
    }
  }

  const handleAssignRole = async (role: Role) => {
    if (!selectedGroup) return
    setIsLoading(true)
    try {
      await fetch("/api/keycloak/groups", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          action: "assignRole",
          groupId: selectedGroup.id,
          roles: [{ id: role.id, name: role.name }],
        }),
      })
      setGroupRoles([...groupRoles, role])
    } finally {
      setIsLoading(false)
    }
  }

  const handleRemoveRole = async (role: Role) => {
    if (!selectedGroup) return
    setIsLoading(true)
    try {
      await fetch("/api/keycloak/groups", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          action: "removeRole",
          groupId: selectedGroup.id,
          roles: [{ id: role.id, name: role.name }],
        }),
      })
      setGroupRoles(groupRoles.filter((r) => r.id !== role.id))
    } finally {
      setIsLoading(false)
    }
  }

  const countGroups = (groups: Group[]): number => {
    return groups.reduce((acc, group) => {
      return acc + 1 + (group.subGroups ? countGroups(group.subGroups) : 0)
    }, 0)
  }

  const getInitials = (user: User) => {
    if (user.firstName && user.lastName) {
      return `${user.firstName[0]}${user.lastName[0]}`.toUpperCase()
    }
    return user.username?.slice(0, 2).toUpperCase() || "??"
  }

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-foreground">Groups</h1>
            <p className="text-muted-foreground">Manage Keycloak groups and hierarchy</p>
          </div>
          <Dialog open={isCreateOpen} onOpenChange={(open) => {
            setIsCreateOpen(open)
            if (!open) setParentGroup(null)
          }}>
            <DialogTrigger asChild>
              <Button className="bg-emerald-600 hover:bg-emerald-700">
                <Plus className="w-4 h-4 mr-2" />
                Add Group
              </Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-md">
              <DialogHeader>
                <DialogTitle>
                  {parentGroup ? `Create Sub-Group in ${parentGroup.name}` : "Create New Group"}
                </DialogTitle>
                <DialogDescription>
                  {parentGroup 
                    ? `Add a new sub-group under ${parentGroup.path}`
                    : "Add a new top-level group to the Keycloak realm"
                  }
                </DialogDescription>
              </DialogHeader>
              <form onSubmit={handleCreateGroup}>
                <FieldGroup className="space-y-4 py-4">
                  <Field>
                    <FieldLabel>Group Name</FieldLabel>
                    <Input name="name" required placeholder="e.g., administrators" />
                  </Field>
                </FieldGroup>
                <DialogFooter>
                  <Button type="button" variant="outline" onClick={() => {
                    setIsCreateOpen(false)
                    setParentGroup(null)
                  }}>
                    Cancel
                  </Button>
                  <Button type="submit" disabled={isLoading} className="bg-emerald-600 hover:bg-emerald-700">
                    {isLoading ? <Spinner className="w-4 h-4 mr-2" /> : null}
                    Create Group
                  </Button>
                </DialogFooter>
              </form>
            </DialogContent>
          </Dialog>
        </div>

        <Card>
          <CardHeader className="pb-3">
            <div className="flex flex-col sm:flex-row items-start sm:items-center gap-4">
              <div className="relative flex-1 max-w-sm">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                <Input
                  placeholder="Search groups..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pl-9"
                />
              </div>
              <div className="flex items-center gap-2">
                <Button variant="ghost" size="icon" onClick={() => refreshGroups()}>
                  <RefreshCw className="w-4 h-4" />
                </Button>
                <div className="text-sm text-muted-foreground">
                  {countGroups(groups)} groups total
                </div>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            {isLoadingGroups ? (
              <div className="flex items-center justify-center py-12">
                <Spinner className="w-6 h-6 text-emerald-600" />
              </div>
            ) : error ? (
              <div className="text-center py-12 text-red-500">
                Failed to load groups. Please check your Keycloak connection.
              </div>
            ) : filteredGroups.length === 0 ? (
              <div className="text-center py-12 text-muted-foreground">
                <Folder className="w-12 h-12 mx-auto mb-3 text-muted-foreground/50" />
                <p>No groups found</p>
                <p className="text-sm">Create your first group to get started</p>
              </div>
            ) : (
              <div className="space-y-1">
                {filteredGroups.map((group) => (
                  <GroupItem
                    key={group.id}
                    group={group}
                    level={0}
                    onEdit={(g) => {
                      setSelectedGroup(g)
                      setIsEditOpen(true)
                    }}
                    onDelete={handleDeleteGroup}
                    onViewMembers={handleViewMembers}
                    onManageRoles={handleManageRoles}
                    onAddSubGroup={handleAddSubGroup}
                  />
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Edit Group Dialog */}
        <Dialog open={isEditOpen} onOpenChange={setIsEditOpen}>
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <DialogTitle>Edit Group</DialogTitle>
              <DialogDescription>
                Update group name for {selectedGroup?.path}
              </DialogDescription>
            </DialogHeader>
            <form onSubmit={handleEditGroup}>
              <FieldGroup className="space-y-4 py-4">
                <Field>
                  <FieldLabel>Group Name</FieldLabel>
                  <Input name="name" required defaultValue={selectedGroup?.name || ""} />
                </Field>
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

        {/* Members Dialog */}
        <Dialog open={membersDialogOpen} onOpenChange={setMembersDialogOpen}>
          <DialogContent className="sm:max-w-2xl">
            <DialogHeader>
              <DialogTitle>Group Members</DialogTitle>
              <DialogDescription>
                Members of {selectedGroup?.name} ({selectedGroup?.path})
              </DialogDescription>
            </DialogHeader>
            <div className="py-4">
              {loadingData ? (
                <div className="flex items-center justify-center py-8">
                  <Spinner className="w-6 h-6 text-emerald-600" />
                </div>
              ) : (
                <div className="space-y-4">
                  <div className="flex justify-end">
                    <Button
                      size="sm"
                      onClick={() => {
                        setAddMemberDialogOpen(true)
                        fetchAvailableUsers()
                      }}
                    >
                      <UserPlus className="w-4 h-4 mr-2" />
                      Add Member
                    </Button>
                  </div>
                  {groupMembers.length > 0 ? (
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>User</TableHead>
                          <TableHead>Email</TableHead>
                          <TableHead className="w-12"></TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {groupMembers.map((member) => (
                          <TableRow key={member.id}>
                            <TableCell>
                              <div className="flex items-center gap-3">
                                <Avatar className="w-8 h-8">
                                  <AvatarFallback className="bg-emerald-100 text-emerald-700 text-xs">
                                    {getInitials(member)}
                                  </AvatarFallback>
                                </Avatar>
                                <div>
                                  <div className="font-medium">{member.username}</div>
                                  {(member.firstName || member.lastName) && (
                                    <div className="text-sm text-muted-foreground">
                                      {[member.firstName, member.lastName].filter(Boolean).join(" ")}
                                    </div>
                                  )}
                                </div>
                              </div>
                            </TableCell>
                            <TableCell className="text-muted-foreground">
                              {member.email || "-"}
                            </TableCell>
                            <TableCell>
                              <Button
                                size="sm"
                                variant="ghost"
                                onClick={() => handleRemoveMember(member)}
                                disabled={isLoading}
                              >
                                <XCircle className="w-4 h-4 text-red-500" />
                              </Button>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  ) : (
                    <div className="text-center py-8 text-muted-foreground">
                      <Users className="w-12 h-12 mx-auto mb-3 text-muted-foreground/50" />
                      <p>No members in this group</p>
                    </div>
                  )}
                </div>
              )}
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setMembersDialogOpen(false)}>
                Close
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* Add Member Dialog */}
        <Dialog open={addMemberDialogOpen} onOpenChange={setAddMemberDialogOpen}>
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <DialogTitle>Add Member</DialogTitle>
              <DialogDescription>
                Add a user to {selectedGroup?.name}
              </DialogDescription>
            </DialogHeader>
            <div className="py-4 space-y-4">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                <Input
                  placeholder="Search users..."
                  value={userSearch}
                  onChange={(e) => setUserSearch(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      e.preventDefault()
                      fetchAvailableUsers()
                    }
                  }}
                  className="pl-9"
                />
              </div>
              <ScrollArea className="h-64 border rounded-lg">
                {loadingData ? (
                  <div className="flex items-center justify-center py-8">
                    <Spinner className="w-6 h-6 text-emerald-600" />
                  </div>
                ) : (
                  <div className="p-2 space-y-2">
                    {availableUsers
                      .filter((user) => !groupMembers.find((m) => m.id === user.id))
                      .map((user) => (
                        <div
                          key={user.id}
                          className="flex items-center justify-between p-2 hover:bg-muted rounded-md"
                        >
                          <div className="flex items-center gap-3">
                            <Avatar className="w-8 h-8">
                              <AvatarFallback className="bg-emerald-100 text-emerald-700 text-xs">
                                {getInitials(user)}
                              </AvatarFallback>
                            </Avatar>
                            <div>
                              <div className="text-sm font-medium">{user.username}</div>
                              {user.email && (
                                <div className="text-xs text-muted-foreground">{user.email}</div>
                              )}
                            </div>
                          </div>
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => handleAddMember(user)}
                            disabled={isLoading}
                          >
                            Add
                          </Button>
                        </div>
                      ))}
                    {availableUsers.filter((user) => !groupMembers.find((m) => m.id === user.id)).length === 0 && (
                      <p className="text-sm text-muted-foreground text-center py-8">
                        No users found or all users are already members
                      </p>
                    )}
                  </div>
                )}
              </ScrollArea>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setAddMemberDialogOpen(false)}>
                Close
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* Manage Roles Dialog */}
        <Dialog open={rolesDialogOpen} onOpenChange={setRolesDialogOpen}>
          <DialogContent className="sm:max-w-lg">
            <DialogHeader>
              <DialogTitle>Group Role Mappings</DialogTitle>
              <DialogDescription>
                Manage realm role mappings for {selectedGroup?.name}
              </DialogDescription>
            </DialogHeader>
            <div className="py-4">
              {loadingData ? (
                <div className="flex items-center justify-center py-8">
                  <Spinner className="w-6 h-6 text-emerald-600" />
                </div>
              ) : (
                <div className="space-y-4">
                  <div>
                    <h4 className="text-sm font-medium mb-2">Assigned Roles</h4>
                    {groupRoles.length > 0 ? (
                      <div className="flex flex-wrap gap-2">
                        {groupRoles.map((role) => (
                          <Badge key={role.id} variant="secondary" className="gap-1">
                            {role.name}
                            <button
                              className="ml-1 hover:text-red-500"
                              onClick={() => handleRemoveRole(role)}
                              disabled={isLoading}
                            >
                              <XCircle className="w-3 h-3" />
                            </button>
                          </Badge>
                        ))}
                      </div>
                    ) : (
                      <p className="text-sm text-muted-foreground">No roles assigned</p>
                    )}
                  </div>
                  <div>
                    <h4 className="text-sm font-medium mb-2">Available Roles</h4>
                    <ScrollArea className="h-48 border rounded-lg p-2">
                      <div className="space-y-2">
                        {availableRoles
                          .filter((role) => !groupRoles.find((r) => r.id === role.id))
                          .map((role) => (
                            <div
                              key={role.id}
                              className="flex items-center justify-between p-2 hover:bg-muted rounded-md"
                            >
                              <div>
                                <p className="text-sm font-medium">{role.name}</p>
                                {role.description && (
                                  <p className="text-xs text-muted-foreground">{role.description}</p>
                                )}
                              </div>
                              <Button
                                size="sm"
                                variant="outline"
                                onClick={() => handleAssignRole(role)}
                                disabled={isLoading}
                              >
                                <Plus className="w-3 h-3 mr-1" />
                                Add
                              </Button>
                            </div>
                          ))}
                        {availableRoles.filter((role) => !groupRoles.find((r) => r.id === role.id)).length === 0 && (
                          <p className="text-sm text-muted-foreground text-center py-4">
                            All available roles have been assigned
                          </p>
                        )}
                      </div>
                    </ScrollArea>
                  </div>
                </div>
              )}
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setRolesDialogOpen(false)}>
                Close
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
    </DashboardLayout>
  )
}
