"use client"

import useSWR from "swr"
import Link from "next/link"
import { DashboardLayout } from "@/components/dashboard-layout"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Spinner } from "@/components/ui/spinner"
import {
  Users,
  Shield,
  Wifi,
  Activity,
  ArrowRight,
  Key,
  Network,
  Clock,
  TrendingUp,
  Globe,
  CheckCircle,
} from "lucide-react"

const fetcher = (url: string) => fetch(url).then((res) => res.json()).catch(() => null)

interface StatCardProps {
  title: string
  value: string | number
  description: string
  icon: React.ElementType
  trend?: string
  color: string
  href?: string
  isLoading?: boolean
}

function StatCard({ title, value, description, icon: Icon, trend, color, href, isLoading }: StatCardProps) {
  const content = (
    <Card className="hover:shadow-md transition-shadow">
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm font-medium text-slate-600">{title}</CardTitle>
        <div className={`p-2 rounded-lg ${color}`}>
          <Icon className="w-4 h-4" />
        </div>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Spinner className="w-6 h-6 text-slate-400" />
        ) : (
          <>
            <div className="text-2xl font-bold text-slate-900">{value}</div>
            <p className="text-xs text-slate-500 mt-1">{description}</p>
            {trend && (
              <div className="flex items-center gap-1 mt-2">
                <TrendingUp className="w-3 h-3 text-emerald-500" />
                <span className="text-xs text-emerald-600">{trend}</span>
              </div>
            )}
          </>
        )}
      </CardContent>
    </Card>
  )

  if (href) {
    return <Link href={href}>{content}</Link>
  }
  return content
}

interface RecentActivity {
  id: string
  type: "login" | "vpn_connect" | "user_created" | "role_assigned"
  user: string
  description: string
  timestamp: string
}

export default function OverviewPage() {
  const { data: usersData, isLoading: isLoadingUsers } = useSWR("/api/keycloak/users?max=1", fetcher)
  const { data: sessionsData, isLoading: isLoadingSessions } = useSWR("/api/keycloak/sessions", fetcher)
  const { data: vpnConnectionsData, isLoading: isLoadingVPN } = useSWR("/api/openvpn/connections", fetcher)
  const { data: vpnUsersData, isLoading: isLoadingVPNUsers } = useSWR("/api/openvpn/users", fetcher)

  const totalUsers = usersData?.total || 0
  const activeSessions = sessionsData?.sessions?.length || 0
  const vpnConnections = vpnConnectionsData?.connections?.length || 0
  const totalVPNUsers = vpnUsersData?.users?.length || 0

  const recentActivity: RecentActivity[] = [
    { id: "1", type: "login", user: "john.doe", description: "Signed in from 192.168.1.100", timestamp: "2 min ago" },
    { id: "2", type: "vpn_connect", user: "jane.smith", description: "Connected to VPN", timestamp: "5 min ago" },
    { id: "3", type: "user_created", user: "admin", description: "Created new user: mike.wilson", timestamp: "15 min ago" },
    { id: "4", type: "role_assigned", user: "admin", description: "Assigned admin role to john.doe", timestamp: "1 hour ago" },
  ]

  const getActivityIcon = (type: string) => {
    switch (type) {
      case "login": return <Key className="w-4 h-4 text-emerald-500" />
      case "vpn_connect": return <Wifi className="w-4 h-4 text-teal-500" />
      case "user_created": return <Users className="w-4 h-4 text-blue-500" />
      case "role_assigned": return <Shield className="w-4 h-4 text-purple-500" />
      default: return <Activity className="w-4 h-4 text-slate-500" />
    }
  }

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-semibold text-slate-900">Dashboard Overview</h1>
          <p className="text-slate-500">Monitor your Keycloak and OpenVPN systems</p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <StatCard
            title="Total Users"
            value={totalUsers}
            description="Keycloak realm users"
            icon={Users}
            color="bg-emerald-100 text-emerald-600"
            href="/keycloak/users"
            isLoading={isLoadingUsers}
          />
          <StatCard
            title="Active Sessions"
            value={activeSessions}
            description="Currently logged in"
            icon={Activity}
            color="bg-blue-100 text-blue-600"
            href="/keycloak/sessions"
            isLoading={isLoadingSessions}
          />
          <StatCard
            title="VPN Connections"
            value={vpnConnections}
            description="Active VPN clients"
            icon={Wifi}
            color="bg-teal-100 text-teal-600"
            href="/openvpn/connections"
            isLoading={isLoadingVPN}
          />
          <StatCard
            title="VPN Users"
            value={totalVPNUsers}
            description="Total VPN accounts"
            icon={Network}
            color="bg-orange-100 text-orange-600"
            href="/openvpn/users"
            isLoading={isLoadingVPNUsers}
          />
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle className="text-lg">Recent Activity</CardTitle>
              <CardDescription>Latest events from your systems</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {recentActivity.map((activity) => (
                  <div key={activity.id} className="flex items-start gap-3 p-3 rounded-lg hover:bg-slate-50 transition-colors">
                    <div className="mt-0.5">{getActivityIcon(activity.type)}</div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-slate-900">{activity.user}</span>
                        <span className="text-slate-500 text-sm truncate">{activity.description}</span>
                      </div>
                      <div className="flex items-center gap-1 mt-1 text-xs text-slate-400">
                        <Clock className="w-3 h-3" />
                        {activity.timestamp}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
              <div className="mt-4 pt-4 border-t">
                <Link href="/keycloak/sessions">
                  <Button variant="ghost" className="w-full text-emerald-600 hover:text-emerald-700 hover:bg-emerald-50">
                    View All Activity
                    <ArrowRight className="w-4 h-4 ml-2" />
                  </Button>
                </Link>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-lg">System Status</CardTitle>
              <CardDescription>Service health overview</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between p-3 rounded-lg bg-slate-50">
                <div className="flex items-center gap-3">
                  <div className="p-2 rounded-lg bg-emerald-100">
                    <Key className="w-4 h-4 text-emerald-600" />
                  </div>
                  <div>
                    <p className="font-medium text-slate-900">Keycloak</p>
                    <p className="text-xs text-slate-500">Identity Provider</p>
                  </div>
                </div>
                <Badge className="bg-emerald-100 text-emerald-700 hover:bg-emerald-100">
                  <CheckCircle className="w-3 h-3 mr-1" />
                  Healthy
                </Badge>
              </div>

              <div className="flex items-center justify-between p-3 rounded-lg bg-slate-50">
                <div className="flex items-center gap-3">
                  <div className="p-2 rounded-lg bg-teal-100">
                    <Network className="w-4 h-4 text-teal-600" />
                  </div>
                  <div>
                    <p className="font-medium text-slate-900">OpenVPN</p>
                    <p className="text-xs text-slate-500">VPN Server</p>
                  </div>
                </div>
                <Badge className="bg-emerald-100 text-emerald-700 hover:bg-emerald-100">
                  <CheckCircle className="w-3 h-3 mr-1" />
                  Healthy
                </Badge>
              </div>

              <div className="flex items-center justify-between p-3 rounded-lg bg-slate-50">
                <div className="flex items-center gap-3">
                  <div className="p-2 rounded-lg bg-blue-100">
                    <Globe className="w-4 h-4 text-blue-600" />
                  </div>
                  <div>
                    <p className="font-medium text-slate-900">API Gateway</p>
                    <p className="text-xs text-slate-500">Admin Interface</p>
                  </div>
                </div>
                <Badge className="bg-emerald-100 text-emerald-700 hover:bg-emerald-100">
                  <CheckCircle className="w-3 h-3 mr-1" />
                  Healthy
                </Badge>
              </div>
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Quick Actions</CardTitle>
            <CardDescription>Common administrative tasks</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <Link href="/keycloak/users">
                <Button variant="outline" className="w-full h-auto py-4 flex-col gap-2">
                  <Users className="w-5 h-5 text-emerald-600" />
                  <span>Add User</span>
                </Button>
              </Link>
              <Link href="/keycloak/groups">
                <Button variant="outline" className="w-full h-auto py-4 flex-col gap-2">
                  <Shield className="w-5 h-5 text-emerald-600" />
                  <span>Manage Groups</span>
                </Button>
              </Link>
              <Link href="/openvpn/configs">
                <Button variant="outline" className="w-full h-auto py-4 flex-col gap-2">
                  <Wifi className="w-5 h-5 text-teal-600" />
                  <span>Generate VPN Config</span>
                </Button>
              </Link>
              <Link href="/keycloak/sessions">
                <Button variant="outline" className="w-full h-auto py-4 flex-col gap-2">
                  <Activity className="w-5 h-5 text-blue-600" />
                  <span>View Sessions</span>
                </Button>
              </Link>
            </div>
          </CardContent>
        </Card>
      </div>
    </DashboardLayout>
  )
}
