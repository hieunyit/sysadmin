"use client"

import { useState } from "react"
import useSWR, { mutate } from "swr"
import { DashboardLayout } from "@/components/dashboard-layout"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardHeader } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
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
import { Spinner } from "@/components/ui/spinner"
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
} from "lucide-react"

const fetcher = (url: string) => fetch(url).then((res) => res.json())

interface Group {
  id: string
  name: string
  path: string
  subGroups?: Group[]
}

interface GroupItemProps {
  group: Group
  level: number
  onEdit: (group: Group) => void
  onDelete: (group: Group) => void
  onViewMembers: (group: Group) => void
}

function GroupItem({ group, level, onEdit, onDelete, onViewMembers }: GroupItemProps) {
  const [isExpanded, setIsExpanded] = useState(true)
  const hasSubGroups = group.subGroups && group.subGroups.length > 0

  return (
    <div>
      <div
        className="flex items-center gap-2 py-2 px-3 hover:bg-slate-50 rounded-lg transition-colors"
        style={{ paddingLeft: `${level * 20 + 12}px` }}
      >
        {hasSubGroups ? (
          <button
            onClick={() => setIsExpanded(!isExpanded)}
            className="p-0.5 hover:bg-slate-200 rounded"
          >
            {isExpanded ? (
              <ChevronDown className="w-4 h-4 text-slate-400" />
            ) : (
              <ChevronRight className="w-4 h-4 text-slate-400" />
            )}
          </button>
        ) : (
          <div className="w-5" />
        )}
        
        {isExpanded && hasSubGroups ? (
          <FolderOpen className="w-5 h-5 text-amber-500" />
        ) : (
          <Folder className="w-5 h-5 text-slate-400" />
        )}
        
        <span className="flex-1 font-medium text-slate-700">{group.name}</span>
        
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
            <DropdownMenuItem>
              <Shield className="w-4 h-4 mr-2" />
              Manage Roles
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
            />
          ))}
        </div>
      )}
    </div>
  )
}

export default function KeycloakGroupsPage() {
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [selectedGroup, setSelectedGroup] = useState<Group | null>(null)
  const [membersDialogOpen, setMembersDialogOpen] = useState(false)

  const { data, error, isLoading: isLoadingGroups } = useSWR(
    "/api/keycloak/groups",
    fetcher,
    { refreshInterval: 30000 }
  )

  const groups: Group[] = data?.groups || []

  const handleCreateGroup = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setIsLoading(true)
    const formData = new FormData(e.currentTarget)
    
    try {
      const res = await fetch("/api/keycloak/groups", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          action: "create",
          name: formData.get("name"),
        }),
      })
      
      if (res.ok) {
        setIsCreateOpen(false)
        mutate("/api/keycloak/groups")
      }
    } finally {
      setIsLoading(false)
    }
  }

  const handleDeleteGroup = async (group: Group) => {
    if (!confirm(`Are you sure you want to delete group "${group.name}"?`)) return
    
    await fetch("/api/keycloak/groups", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action: "delete", id: group.id }),
    })
    mutate("/api/keycloak/groups")
  }

  const handleViewMembers = async (group: Group) => {
    setSelectedGroup(group)
    setMembersDialogOpen(true)
  }

  const countGroups = (groups: Group[]): number => {
    return groups.reduce((acc, group) => {
      return acc + 1 + (group.subGroups ? countGroups(group.subGroups) : 0)
    }, 0)
  }

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-slate-900">Groups</h1>
            <p className="text-slate-500">Manage Keycloak groups and hierarchy</p>
          </div>
          <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
            <DialogTrigger asChild>
              <Button className="bg-emerald-600 hover:bg-emerald-700">
                <Plus className="w-4 h-4 mr-2" />
                Add Group
              </Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-md">
              <DialogHeader>
                <DialogTitle>Create New Group</DialogTitle>
                <DialogDescription>
                  Add a new group to the Keycloak realm
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
                  <Button type="button" variant="outline" onClick={() => setIsCreateOpen(false)}>
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
            <div className="flex items-center justify-between">
              <div className="text-sm text-slate-500">
                {countGroups(groups)} groups total
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
            ) : groups.length === 0 ? (
              <div className="text-center py-12 text-slate-500">
                <Folder className="w-12 h-12 mx-auto mb-3 text-slate-300" />
                <p>No groups found</p>
                <p className="text-sm">Create your first group to get started</p>
              </div>
            ) : (
              <div className="space-y-1">
                {groups.map((group) => (
                  <GroupItem
                    key={group.id}
                    group={group}
                    level={0}
                    onEdit={(g) => console.log("Edit", g)}
                    onDelete={handleDeleteGroup}
                    onViewMembers={handleViewMembers}
                  />
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Members Dialog */}
        <Dialog open={membersDialogOpen} onOpenChange={setMembersDialogOpen}>
          <DialogContent className="sm:max-w-lg">
            <DialogHeader>
              <DialogTitle>Group Members</DialogTitle>
              <DialogDescription>
                Members of {selectedGroup?.name}
              </DialogDescription>
            </DialogHeader>
            <div className="py-4">
              <p className="text-sm text-slate-500 text-center">
                Member management coming soon
              </p>
            </div>
          </DialogContent>
        </Dialog>
      </div>
    </DashboardLayout>
  )
}
