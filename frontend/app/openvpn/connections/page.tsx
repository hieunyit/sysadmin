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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Spinner } from "@/components/ui/spinner"
import { Progress } from "@/components/ui/progress"
import {
  RefreshCw,
  Unplug,
  Globe,
  Clock,
  ArrowDownToLine,
  ArrowUpFromLine,
  Activity,
  Wifi,
  WifiOff,
} from "lucide-react"
import { formatDistanceToNow } from "date-fns"

const fetcher = (url: string) => fetch(url).then((res) => res.json())

interface VPNConnection {
  id: string
  username: string
  realAddress: string
  virtualAddress: string
  bytesReceived: number
  bytesSent: number
  connectedSince: string
  clientId: string
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B"
  const k = 1024
  const sizes = ["B", "KB", "MB", "GB", "TB"]
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i]
}

export default function OpenVPNConnectionsPage() {
  const [isDisconnecting, setIsDisconnecting] = useState<string | null>(null)

  const { data, error, isLoading } = useSWR(
    "/api/openvpn/connections",
    fetcher,
    { refreshInterval: 5000 }
  )

  const connections: VPNConnection[] = data?.connections || []

  const handleDisconnect = async (connection: VPNConnection) => {
    if (!confirm(`Disconnect user "${connection.username}"?`)) return
    
    setIsDisconnecting(connection.id)
    try {
      await fetch("/api/openvpn/connections", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action: "disconnect", clientId: connection.clientId }),
      })
      mutate("/api/openvpn/connections")
    } finally {
      setIsDisconnecting(null)
    }
  }

  const totalBytesReceived = connections.reduce((acc, c) => acc + c.bytesReceived, 0)
  const totalBytesSent = connections.reduce((acc, c) => acc + c.bytesSent, 0)

  const getInitials = (username: string) => {
    return username.slice(0, 2).toUpperCase()
  }

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-slate-900">Connections</h1>
            <p className="text-slate-500">Monitor active VPN connections in real-time</p>
          </div>
          <Button
            variant="outline"
            onClick={() => mutate("/api/openvpn/connections")}
          >
            <RefreshCw className="w-4 h-4 mr-2" />
            Refresh
          </Button>
        </div>

        {/* Stats Cards */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <Card>
            <CardHeader className="pb-2">
              <CardDescription>Active Connections</CardDescription>
              <CardTitle className="text-3xl font-bold text-teal-600">
                {connections.length}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-center text-sm text-slate-500">
                <Wifi className="w-4 h-4 mr-1 text-teal-500" />
                Currently connected
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardDescription>Unique Users</CardDescription>
              <CardTitle className="text-3xl font-bold">
                {new Set(connections.map((c) => c.username)).size}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-center text-sm text-slate-500">
                <Activity className="w-4 h-4 mr-1" />
                Connected users
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardDescription>Total Downloaded</CardDescription>
              <CardTitle className="text-3xl font-bold text-blue-600">
                {formatBytes(totalBytesReceived)}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-center text-sm text-slate-500">
                <ArrowDownToLine className="w-4 h-4 mr-1 text-blue-500" />
                Data received
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardDescription>Total Uploaded</CardDescription>
              <CardTitle className="text-3xl font-bold text-orange-600">
                {formatBytes(totalBytesSent)}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-center text-sm text-slate-500">
                <ArrowUpFromLine className="w-4 h-4 mr-1 text-orange-500" />
                Data sent
              </div>
            </CardContent>
          </Card>
        </div>

        <Tabs defaultValue="active" className="space-y-4">
          <TabsList>
            <TabsTrigger value="active">Active Connections</TabsTrigger>
            <TabsTrigger value="history">Connection History</TabsTrigger>
          </TabsList>

          <TabsContent value="active">
            <Card>
              <CardContent className="pt-6">
                {isLoading ? (
                  <div className="flex items-center justify-center py-12">
                    <Spinner className="w-6 h-6 text-teal-600" />
                  </div>
                ) : error ? (
                  <div className="text-center py-12 text-red-500">
                    Failed to load connections. Please check your OpenVPN connection.
                  </div>
                ) : connections.length === 0 ? (
                  <div className="text-center py-12 text-slate-500">
                    <WifiOff className="w-12 h-12 mx-auto mb-3 text-slate-300" />
                    <p>No active connections</p>
                    <p className="text-sm">Connections will appear here when users connect</p>
                  </div>
                ) : (
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>User</TableHead>
                        <TableHead>Real IP</TableHead>
                        <TableHead>Virtual IP</TableHead>
                        <TableHead>Connected</TableHead>
                        <TableHead>Data Transfer</TableHead>
                        <TableHead className="w-24"></TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {connections.map((connection) => {
                        const totalBytes = connection.bytesReceived + connection.bytesSent
                        const downloadPercent = totalBytes > 0 
                          ? (connection.bytesReceived / totalBytes) * 100 
                          : 50

                        return (
                          <TableRow key={connection.id}>
                            <TableCell>
                              <div className="flex items-center gap-3">
                                <Avatar className="w-8 h-8">
                                  <AvatarFallback className="bg-teal-100 text-teal-700 text-xs">
                                    {getInitials(connection.username)}
                                  </AvatarFallback>
                                </Avatar>
                                <div>
                                  <div className="font-medium text-slate-900">{connection.username}</div>
                                  <div className="text-xs text-slate-500 font-mono">
                                    {connection.clientId.slice(0, 8)}...
                                  </div>
                                </div>
                              </div>
                            </TableCell>
                            <TableCell>
                              <div className="flex items-center gap-1.5 text-slate-600 font-mono text-sm">
                                <Globe className="w-3.5 h-3.5 text-slate-400" />
                                {connection.realAddress}
                              </div>
                            </TableCell>
                            <TableCell>
                              <Badge variant="outline" className="font-mono">
                                {connection.virtualAddress}
                              </Badge>
                            </TableCell>
                            <TableCell>
                              <div className="flex items-center gap-1.5 text-slate-500 text-sm">
                                <Clock className="w-3.5 h-3.5" />
                                {formatDistanceToNow(new Date(connection.connectedSince), { addSuffix: true })}
                              </div>
                            </TableCell>
                            <TableCell>
                              <div className="space-y-1 min-w-[140px]">
                                <div className="flex justify-between text-xs text-slate-500">
                                  <span className="flex items-center gap-1">
                                    <ArrowDownToLine className="w-3 h-3 text-blue-500" />
                                    {formatBytes(connection.bytesReceived)}
                                  </span>
                                  <span className="flex items-center gap-1">
                                    <ArrowUpFromLine className="w-3 h-3 text-orange-500" />
                                    {formatBytes(connection.bytesSent)}
                                  </span>
                                </div>
                                <Progress value={downloadPercent} className="h-1.5" />
                              </div>
                            </TableCell>
                            <TableCell>
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => handleDisconnect(connection)}
                                disabled={isDisconnecting === connection.id}
                                className="text-red-600 hover:text-red-700 hover:bg-red-50"
                              >
                                {isDisconnecting === connection.id ? (
                                  <Spinner className="w-4 h-4" />
                                ) : (
                                  <>
                                    <Unplug className="w-4 h-4 mr-1" />
                                    Disconnect
                                  </>
                                )}
                              </Button>
                            </TableCell>
                          </TableRow>
                        )
                      })}
                    </TableBody>
                  </Table>
                )}
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="history">
            <Card>
              <CardContent className="py-12">
                <div className="text-center text-slate-500">
                  <Activity className="w-12 h-12 mx-auto mb-3 text-slate-300" />
                  <p>Connection history</p>
                  <p className="text-sm">Historical connection data will appear here</p>
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </DashboardLayout>
  )
}
