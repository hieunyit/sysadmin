"use client"

import { useState } from "react"
import useSWR, { mutate } from "swr"
import { DashboardLayout } from "@/components/dashboard-layout"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Search,
  Plus,
  MoreHorizontal,
  Shield,
  Users,
  Layers,
  Trash2,
  Edit,
} from "lucide-react"

const fetcher = (url: string) => fetch(url).then((res) => res.json())

interface Role {
  id: string
  name: string
  description?: string
  composite: boolean
  clientRole: boolean
  containerId: string
}

export default function KeycloakRolesPage() {
  const [search, setSearch] = useState("")
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [isLoading, setIsLoading] = useState(false)

  const { data, error, isLoading: isLoadingRoles } = useSWR(
    "/api/keycloak/roles",
    fetcher,
    { refreshInterval: 30000 }
  )

  const allRoles: Role[] = data?.roles || []
  const realmRoles = allRoles.filter((r) => !r.clientRole)
  const filteredRoles = realmRoles.filter(
    (role) =>
      role.name.toLowerCase().includes(search.toLowerCase()) ||
      role.description?.toLowerCase().includes(search.toLowerCase())
  )

  // Default Keycloak roles that shouldn't be deleted
  const systemRoles = ["offline_access", "uma_authorization", "default-roles-"]

  const isSystemRole = (roleName: string) => {
    return systemRoles.some((sr) => roleName.startsWith(sr) || roleName === sr)
  }

  const handleCreateRole = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setIsLoading(true)
    const formData = new FormData(e.currentTarget)
    
    try {
      const res = await fetch("/api/keycloak/roles", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          action: "create",
          name: formData.get("name"),
          description: formData.get("description"),
        }),
      })
      
      if (res.ok) {
        setIsCreateOpen(false)
        mutate("/api/keycloak/roles")
      }
    } finally {
      setIsLoading(false)
    }
  }

  const handleDeleteRole = async (role: Role) => {
    if (isSystemRole(role.name)) {
      alert("Cannot delete system roles")
      return
    }
    if (!confirm(`Are you sure you want to delete role "${role.name}"?`)) return
    
    await fetch("/api/keycloak/roles", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action: "delete", name: role.name }),
    })
    mutate("/api/keycloak/roles")
  }

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-slate-900">Roles</h1>
            <p className="text-slate-500">Manage Keycloak roles and permissions</p>
          </div>
          <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
            <DialogTrigger asChild>
              <Button className="bg-emerald-600 hover:bg-emerald-700">
                <Plus className="w-4 h-4 mr-2" />
                Add Role
              </Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-md">
              <DialogHeader>
                <DialogTitle>Create New Role</DialogTitle>
                <DialogDescription>
                  Add a new realm role to Keycloak
                </DialogDescription>
              </DialogHeader>
              <form onSubmit={handleCreateRole}>
                <FieldGroup className="space-y-4 py-4">
                  <Field>
                    <FieldLabel>Role Name</FieldLabel>
                    <Input name="name" required placeholder="e.g., admin" />
                  </Field>
                  <Field>
                    <FieldLabel>Description</FieldLabel>
                    <Textarea 
                      name="description" 
                      placeholder="Optional description for this role"
                      rows={3}
                    />
                  </Field>
                </FieldGroup>
                <DialogFooter>
                  <Button type="button" variant="outline" onClick={() => setIsCreateOpen(false)}>
                    Cancel
                  </Button>
                  <Button type="submit" disabled={isLoading} className="bg-emerald-600 hover:bg-emerald-700">
                    {isLoading ? <Spinner className="w-4 h-4 mr-2" /> : null}
                    Create Role
                  </Button>
                </DialogFooter>
              </form>
            </DialogContent>
          </Dialog>
        </div>

        <Tabs defaultValue="realm" className="space-y-4">
          <TabsList>
            <TabsTrigger value="realm">Realm Roles</TabsTrigger>
            <TabsTrigger value="client">Client Roles</TabsTrigger>
          </TabsList>

          <TabsContent value="realm">
            <Card>
              <CardHeader className="pb-4">
                <div className="flex items-center gap-4">
                  <div className="relative flex-1 max-w-sm">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
                    <Input
                      placeholder="Search roles..."
                      value={search}
                      onChange={(e) => setSearch(e.target.value)}
                      className="pl-9"
                    />
                  </div>
                  <div className="text-sm text-slate-500">
                    {filteredRoles.length} roles
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                {isLoadingRoles ? (
                  <div className="flex items-center justify-center py-12">
                    <Spinner className="w-6 h-6 text-emerald-600" />
                  </div>
                ) : error ? (
                  <div className="text-center py-12 text-red-500">
                    Failed to load roles. Please check your Keycloak connection.
                  </div>
                ) : filteredRoles.length === 0 ? (
                  <div className="text-center py-12 text-slate-500">
                    <Shield className="w-12 h-12 mx-auto mb-3 text-slate-300" />
                    <p>No roles found</p>
                  </div>
                ) : (
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Role Name</TableHead>
                        <TableHead>Description</TableHead>
                        <TableHead>Type</TableHead>
                        <TableHead className="w-12"></TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {filteredRoles.map((role) => (
                        <TableRow key={role.id}>
                          <TableCell>
                            <div className="flex items-center gap-2">
                              <Shield className="w-4 h-4 text-emerald-500" />
                              <span className="font-medium text-slate-900">{role.name}</span>
                              {isSystemRole(role.name) && (
                                <Badge variant="secondary" className="text-xs">System</Badge>
                              )}
                            </div>
                          </TableCell>
                          <TableCell className="text-slate-500 max-w-md truncate">
                            {role.description || "-"}
                          </TableCell>
                          <TableCell>
                            {role.composite ? (
                              <Badge variant="outline" className="gap-1">
                                <Layers className="w-3 h-3" />
                                Composite
                              </Badge>
                            ) : (
                              <Badge variant="secondary">Simple</Badge>
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
                                <DropdownMenuItem>
                                  <Users className="w-4 h-4 mr-2" />
                                  View Users
                                </DropdownMenuItem>
                                <DropdownMenuItem>
                                  <Layers className="w-4 h-4 mr-2" />
                                  Composite Roles
                                </DropdownMenuItem>
                                {!isSystemRole(role.name) && (
                                  <>
                                    <DropdownMenuItem>
                                      <Edit className="w-4 h-4 mr-2" />
                                      Edit Role
                                    </DropdownMenuItem>
                                    <DropdownMenuSeparator />
                                    <DropdownMenuItem
                                      onClick={() => handleDeleteRole(role)}
                                      className="text-red-600"
                                    >
                                      <Trash2 className="w-4 h-4 mr-2" />
                                      Delete Role
                                    </DropdownMenuItem>
                                  </>
                                )}
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
          </TabsContent>

          <TabsContent value="client">
            <Card>
              <CardContent className="py-12">
                <div className="text-center text-slate-500">
                  <Shield className="w-12 h-12 mx-auto mb-3 text-slate-300" />
                  <p>Client role management</p>
                  <p className="text-sm">Select a client to view its roles</p>
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </DashboardLayout>
  )
}
