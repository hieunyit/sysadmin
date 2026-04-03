"use client"

import { useState } from "react"
import useSWR, { mutate } from "swr"
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
import { Spinner } from "@/components/ui/spinner"
import {
  Search,
  Plus,
  MoreHorizontal,
  Download,
  FileKey,
  Trash2,
  QrCode,
  Ban,
  Clock,
  User,
} from "lucide-react"
import { formatDistanceToNow } from "date-fns"

const fetcher = (url: string) => fetch(url).then((res) => res.json())

interface VPNConfig {
  id: string
  name: string
  username: string
  createdAt: string
  expiresAt?: string
  downloadCount: number
  status?: "active" | "revoked" | "expired"
}

interface VPNUser {
  id: string
  username: string
}

export default function OpenVPNConfigsPage() {
  const [search, setSearch] = useState("")
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [isQROpen, setIsQROpen] = useState(false)
  const [selectedConfig, setSelectedConfig] = useState<VPNConfig | null>(null)
  const [qrCode, setQRCode] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)

  const { data: configsData, error: configsError, isLoading: isLoadingConfigs } = useSWR(
    "/api/openvpn/configs",
    fetcher,
    { refreshInterval: 30000 }
  )

  const { data: usersData } = useSWR("/api/openvpn/users", fetcher)

  const configs: VPNConfig[] = configsData?.configs || []
  const users: VPNUser[] = usersData?.users || []
  
  const filteredConfigs = configs.filter(
    (config) =>
      config.name.toLowerCase().includes(search.toLowerCase()) ||
      config.username.toLowerCase().includes(search.toLowerCase())
  )

  const handleCreateConfig = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setIsLoading(true)
    const formData = new FormData(e.currentTarget)
    
    try {
      const res = await fetch("/api/openvpn/configs", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          action: "create",
          username: formData.get("username"),
          name: formData.get("name"),
        }),
      })
      
      if (res.ok) {
        setIsCreateOpen(false)
        mutate("/api/openvpn/configs")
      }
    } finally {
      setIsLoading(false)
    }
  }

  const handleDownload = async (config: VPNConfig) => {
    try {
      const res = await fetch("/api/openvpn/configs", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          action: "download",
          configId: config.id,
          filename: `${config.name}.ovpn`,
        }),
      })
      
      if (res.ok) {
        const blob = await res.blob()
        const url = window.URL.createObjectURL(blob)
        const a = document.createElement("a")
        a.href = url
        a.download = `${config.name}.ovpn`
        document.body.appendChild(a)
        a.click()
        window.URL.revokeObjectURL(url)
        document.body.removeChild(a)
        mutate("/api/openvpn/configs")
      }
    } catch (error) {
      console.error("Download failed:", error)
    }
  }

  const handleShowQR = async (config: VPNConfig) => {
    setSelectedConfig(config)
    setIsQROpen(true)
    
    try {
      const res = await fetch("/api/openvpn/configs", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action: "qrcode", configId: config.id }),
      })
      
      if (res.ok) {
        const data = await res.json()
        setQRCode(data.qrCode)
      }
    } catch (error) {
      console.error("Failed to get QR code:", error)
    }
  }

  const handleRevoke = async (config: VPNConfig) => {
    if (!confirm(`Revoke config "${config.name}"? This cannot be undone.`)) return
    
    await fetch("/api/openvpn/configs", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action: "revoke", configId: config.id }),
    })
    mutate("/api/openvpn/configs")
  }

  const handleDelete = async (config: VPNConfig) => {
    if (!confirm(`Delete config "${config.name}"?`)) return
    
    await fetch("/api/openvpn/configs", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action: "delete", configId: config.id }),
    })
    mutate("/api/openvpn/configs")
  }

  const getStatusBadge = (config: VPNConfig) => {
    if (config.status === "revoked") {
      return <Badge variant="destructive">Revoked</Badge>
    }
    if (config.expiresAt && new Date(config.expiresAt) < new Date()) {
      return <Badge variant="secondary">Expired</Badge>
    }
    return <Badge className="bg-emerald-100 text-emerald-700 hover:bg-emerald-100">Active</Badge>
  }

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-slate-900">Configurations</h1>
            <p className="text-slate-500">Manage OpenVPN connection profiles</p>
          </div>
          <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
            <DialogTrigger asChild>
              <Button className="bg-teal-600 hover:bg-teal-700">
                <Plus className="w-4 h-4 mr-2" />
                Generate Config
              </Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-md">
              <DialogHeader>
                <DialogTitle>Generate VPN Configuration</DialogTitle>
                <DialogDescription>
                  Create a new .ovpn configuration file for a user
                </DialogDescription>
              </DialogHeader>
              <form onSubmit={handleCreateConfig}>
                <FieldGroup className="space-y-4 py-4">
                  <Field>
                    <FieldLabel>User</FieldLabel>
                    <Select name="username" required>
                      <SelectTrigger>
                        <SelectValue placeholder="Select a user" />
                      </SelectTrigger>
                      <SelectContent>
                        {users.map((user) => (
                          <SelectItem key={user.id} value={user.username}>
                            {user.username}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </Field>
                  <Field>
                    <FieldLabel>Config Name</FieldLabel>
                    <Input name="name" required placeholder="e.g., laptop-work" />
                  </Field>
                </FieldGroup>
                <DialogFooter>
                  <Button type="button" variant="outline" onClick={() => setIsCreateOpen(false)}>
                    Cancel
                  </Button>
                  <Button type="submit" disabled={isLoading} className="bg-teal-600 hover:bg-teal-700">
                    {isLoading ? <Spinner className="w-4 h-4 mr-2" /> : null}
                    Generate
                  </Button>
                </DialogFooter>
              </form>
            </DialogContent>
          </Dialog>
        </div>

        <Card>
          <CardHeader className="pb-4">
            <div className="flex items-center gap-4">
              <div className="relative flex-1 max-w-sm">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
                <Input
                  placeholder="Search configurations..."
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  className="pl-9"
                />
              </div>
              <div className="text-sm text-slate-500">
                {filteredConfigs.length} configurations
              </div>
            </div>
          </CardHeader>
          <CardContent>
            {isLoadingConfigs ? (
              <div className="flex items-center justify-center py-12">
                <Spinner className="w-6 h-6 text-teal-600" />
              </div>
            ) : configsError ? (
              <div className="text-center py-12 text-red-500">
                Failed to load configurations. Please check your OpenVPN connection.
              </div>
            ) : filteredConfigs.length === 0 ? (
              <div className="text-center py-12 text-slate-500">
                <FileKey className="w-12 h-12 mx-auto mb-3 text-slate-300" />
                <p>No configurations found</p>
                <p className="text-sm">Generate your first config to get started</p>
              </div>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>User</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Created</TableHead>
                    <TableHead>Downloads</TableHead>
                    <TableHead className="w-12"></TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredConfigs.map((config) => (
                    <TableRow key={config.id}>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <FileKey className="w-4 h-4 text-teal-500" />
                          <span className="font-medium text-slate-900">{config.name}</span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1.5 text-slate-600">
                          <User className="w-3.5 h-3.5" />
                          {config.username}
                        </div>
                      </TableCell>
                      <TableCell>{getStatusBadge(config)}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1.5 text-slate-500 text-sm">
                          <Clock className="w-3.5 h-3.5" />
                          {formatDistanceToNow(new Date(config.createdAt), { addSuffix: true })}
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge variant="secondary">{config.downloadCount}</Badge>
                      </TableCell>
                      <TableCell>
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="icon" className="h-8 w-8">
                              <MoreHorizontal className="w-4 h-4" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            <DropdownMenuItem onClick={() => handleDownload(config)}>
                              <Download className="w-4 h-4 mr-2" />
                              Download .ovpn
                            </DropdownMenuItem>
                            <DropdownMenuItem onClick={() => handleShowQR(config)}>
                              <QrCode className="w-4 h-4 mr-2" />
                              Show QR Code
                            </DropdownMenuItem>
                            <DropdownMenuSeparator />
                            <DropdownMenuItem
                              onClick={() => handleRevoke(config)}
                              className="text-orange-600"
                            >
                              <Ban className="w-4 h-4 mr-2" />
                              Revoke
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              onClick={() => handleDelete(config)}
                              className="text-red-600"
                            >
                              <Trash2 className="w-4 h-4 mr-2" />
                              Delete
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

        {/* QR Code Dialog */}
        <Dialog open={isQROpen} onOpenChange={(open) => {
          setIsQROpen(open)
          if (!open) {
            setQRCode(null)
            setSelectedConfig(null)
          }
        }}>
          <DialogContent className="sm:max-w-sm">
            <DialogHeader>
              <DialogTitle>Mobile Setup</DialogTitle>
              <DialogDescription>
                Scan this QR code with OpenVPN Connect app
              </DialogDescription>
            </DialogHeader>
            <div className="flex items-center justify-center py-6">
              {qrCode ? (
                <div className="p-4 bg-white rounded-lg border">
                  <img src={qrCode} alt="QR Code" className="w-48 h-48" />
                </div>
              ) : (
                <Spinner className="w-8 h-8 text-teal-600" />
              )}
            </div>
            <p className="text-xs text-center text-slate-500">
              Config: {selectedConfig?.name}
            </p>
          </DialogContent>
        </Dialog>
      </div>
    </DashboardLayout>
  )
}
