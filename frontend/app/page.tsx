"use client"

import { useState } from "react"
import {
  Search,
  Bell,
  Home,
  Workflow,
  BarChart3,
  Settings,
  AlertTriangle,
  RefreshCw,
  MoreHorizontal,
  Filter,
  TrendingUp,
  TrendingDown,
  Clock,
  CheckCircle,
  XCircle,
  ChevronDown,
  Plus,
  ArrowRight,
  Users,
  Eye,
  Database,
} from "lucide-react"
import { AreaChart, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Area } from "recharts"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
  DropdownMenuLabel,
} from "@/components/ui/dropdown-menu"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Progress } from "@/components/ui/progress"
import Link from "next/link"

// Sample data
const metricsData = [
  { label: "Total Workflows", value: "237", change: "+12%", trend: "up", icon: Workflow },
  { label: "Success Rate", value: "98.7%", change: "+0.3%", trend: "up", icon: CheckCircle },
  { label: "Avg Response", value: "38s", change: "-2.1s", trend: "up", icon: Clock },
  { label: "Active Users", value: "1,423", change: "+8.2%", trend: "up", icon: Users },
]

const workflowData = [
  {
    id: 6734,
    name: "Product Catalog Sync",
    started: "22 Jun 2025, 10:48",
    duration: "45.2s",
    status: "running",
    error: null,
  },
  {
    id: 6733,
    name: "Customer Webhook Listener",
    started: "22 Jun 2025, 10:12",
    duration: "30s",
    status: "success",
    error: null,
  },
  {
    id: 6732,
    name: "Data Enrichment Pipeline",
    started: "22 Jun 2025, 09:45",
    duration: "2m 15s",
    status: "success",
    error: null,
  },
  {
    id: 6731,
    name: "Analytics Refresh",
    started: "22 Jun 2025, 09:30",
    duration: "1m 8s",
    status: "success",
    error: null,
  },
  {
    id: 6730,
    name: "Billing Reconciliation",
    started: "22 Jun 2025, 09:15",
    duration: "3m 22s",
    status: "success",
    error: null,
  },
  {
    id: 6729,
    name: "Inventory Level Sync",
    started: "22 Jun 2025, 08:58",
    duration: "45s",
    status: "failed",
    error: "HTTP Error 404: Not Found",
  },
  {
    id: 6728,
    name: "KYC Data Update",
    started: "22 Jun 2025, 08:45",
    duration: "1m 12s",
    status: "success",
    error: null,
  },
  {
    id: 6727,
    name: "Monthly Log Archiver",
    started: "22 Jun 2025, 08:30",
    duration: "4m 33s",
    status: "success",
    error: null,
  },
]

const chartData = [
  { name: "Jan", sales: 4000, views: 2400, workflows: 240 },
  { name: "Feb", sales: 3000, views: 1398, workflows: 221 },
  { name: "Mar", sales: 2000, views: 9800, workflows: 229 },
  { name: "Apr", sales: 2780, views: 3908, workflows: 200 },
  { name: "May", sales: 1890, views: 4800, workflows: 218 },
  { name: "Jun", sales: 2390, views: 3800, workflows: 250 },
  { name: "Jul", sales: 3490, views: 4300, workflows: 210 },
]

const teamMembers = [
  {
    name: "Clara Blackwood",
    role: "Engineer",
    status: "online",
    avatar: "/placeholder.svg?height=32&width=32",
    availability: "On-call",
  },
  {
    name: "Michael Whitmore",
    role: "Owner",
    status: "online",
    avatar: "/placeholder.svg?height=32&width=32",
    availability: "Available",
  },
  {
    name: "Dennis Brightwood",
    role: "Engineer",
    status: "away",
    avatar: "/placeholder.svg?height=32&width=32",
    availability: "Available in 2hrs",
  },
  {
    name: "Sarah Chen",
    role: "Designer",
    status: "online",
    avatar: "/placeholder.svg?height=32&width=32",
    availability: "In meeting",
  },
]

const recentActivity = [
  { workflow: "Product Catalog Sync", time: "2 minutes ago", status: "success", duration: "45s" },
  { workflow: "Customer Webhook", time: "5 minutes ago", status: "success", duration: "30s" },
  { workflow: "Data Enrichment", time: "12 minutes ago", status: "success", duration: "2m 15s" },
  { workflow: "Analytics Refresh", time: "18 minutes ago", status: "success", duration: "1m 8s" },
  { workflow: "Inventory Sync", time: "32 minutes ago", status: "failed", duration: "45s" },
]

export default function Dashboard() {
  const [selectedPeriod, setSelectedPeriod] = useState("Last 30 days")

  return (
    <div className="min-h-screen bg-white">
      {/* Header */}
      <header className="h-16 border-b border-gray-200 bg-white px-6 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 bg-gradient-to-br from-purple-600 to-blue-600 rounded-lg flex items-center justify-center">
              <Workflow className="w-4 h-4 text-white" />
            </div>
            <span className="font-semibold text-gray-900">Emma</span>
          </div>
          <div className="text-sm text-gray-500">
            <span>Dashboard</span> <span className="mx-1">/</span> <span>Overview</span>
          </div>
        </div>

        <div className="flex items-center gap-4">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-4 h-4" />
            <Input
              placeholder="Search workflows, logs..."
              className="pl-10 w-80 bg-gray-50 border-gray-200 focus:bg-white"
            />
          </div>
          <Button variant="ghost" size="icon" className="relative">
            <Bell className="w-4 h-4" />
            <span className="absolute -top-1 -right-1 w-2 h-2 bg-red-500 rounded-full"></span>
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon">
                <Avatar className="w-8 h-8">
                  <AvatarImage src="/placeholder.svg?height=32&width=32" />
                  <AvatarFallback>AE</AvatarFallback>
                </Avatar>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-56">
              <DropdownMenuLabel>Alex Evans</DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem>Profile</DropdownMenuItem>
              <DropdownMenuItem>Settings</DropdownMenuItem>
              <DropdownMenuItem>Support</DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem>Sign out</DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </header>

      <div className="flex">
        {/* Sidebar */}
        <aside className="w-60 border-r border-gray-200 bg-white h-[calc(100vh-4rem)] overflow-y-auto">
          <div className="p-4">
            <div className="relative mb-6">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-4 h-4" />
              <Input placeholder="Search anything..." className="pl-10 bg-gray-50 border-gray-200 text-sm" />
              <Button
                size="icon"
                variant="ghost"
                className="absolute right-1 top-1/2 transform -translate-y-1/2 w-6 h-6"
              >
                <ArrowRight className="w-3 h-3" />
              </Button>
            </div>

            <nav className="space-y-1">
              <Link
                href="/"
                className="flex items-center w-full justify-start bg-purple-50 text-purple-700 hover:bg-purple-100 px-3 py-2 rounded-md text-sm font-medium"
              >
                <Home className="w-4 h-4 mr-3" />
                Overview
              </Link>
              <Link
                href="/workflows"
                className="flex items-center w-full justify-start text-gray-600 hover:bg-gray-50 px-3 py-2 rounded-md text-sm font-medium"
              >
                <Workflow className="w-4 h-4 mr-3" />
                Workflows
              </Link>
              <Link
                href="/analytics"
                className="flex items-center w-full justify-start text-gray-600 hover:bg-gray-50 px-3 py-2 rounded-md text-sm font-medium"
              >
                <BarChart3 className="w-4 h-4 mr-3" />
                Analytics
              </Link>
              <Link
                href="/templates"
                className="flex items-center w-full justify-start text-gray-600 hover:bg-gray-50 px-3 py-2 rounded-md text-sm font-medium"
              >
                <Database className="w-4 h-4 mr-3" />
                Templates
              </Link>
              <Link
                href="/team"
                className="flex items-center w-full justify-start text-gray-600 hover:bg-gray-50 px-3 py-2 rounded-md text-sm font-medium"
              >
                <Users className="w-4 h-4 mr-3" />
                Team
              </Link>
              <Link
                href="/settings"
                className="flex items-center w-full justify-start text-gray-600 hover:bg-gray-50 px-3 py-2 rounded-md text-sm font-medium"
              >
                <Settings className="w-4 h-4 mr-3" />
                Settings
              </Link>
            </nav>
          </div>
        </aside>

        {/* Main Content */}
        <main className="flex-1 p-8 bg-gray-50">
          {/* Quick Actions Bar */}
          <div className="mb-8">
            <div className="flex items-center justify-between mb-6">
              <div>
                <h1 className="text-2xl font-semibold text-gray-900">Dashboard Overview</h1>
                <p className="text-gray-600 mt-1">Monitor your workflows and system performance</p>
              </div>
              <div className="flex items-center gap-3">
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="outline" className="gap-2 bg-transparent">
                      {selectedPeriod} <ChevronDown className="w-4 h-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent>
                    <DropdownMenuItem onClick={() => setSelectedPeriod("Last 7 days")}>Last 7 days</DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setSelectedPeriod("Last 30 days")}>Last 30 days</DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setSelectedPeriod("Last 90 days")}>Last 90 days</DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
                <Button className="bg-purple-600 hover:bg-purple-700">
                  <Plus className="w-4 h-4 mr-2" />
                  New Workflow
                </Button>
              </div>
            </div>

            {/* Quick Action Cards */}
            <div className="grid grid-cols-3 gap-4 mb-8">
              <Card className="p-6 hover:shadow-md transition-shadow cursor-pointer border-gray-200">
                <div className="flex items-center gap-4">
                  <div className="w-12 h-12 bg-green-100 rounded-lg flex items-center justify-center">
                    <Plus className="w-6 h-6 text-green-600" />
                  </div>
                  <div>
                    <h3 className="font-medium text-gray-900">New workflow</h3>
                    <p className="text-sm text-gray-600">Create a new automation</p>
                  </div>
                </div>
              </Card>

              <Card className="p-6 hover:shadow-md transition-shadow cursor-pointer border-gray-200">
                <div className="flex items-center gap-4">
                  <div className="w-12 h-12 bg-red-100 rounded-lg flex items-center justify-center">
                    <AlertTriangle className="w-6 h-6 text-red-600" />
                  </div>
                  <div>
                    <h3 className="font-medium text-gray-900">View breaches</h3>
                    <p className="text-sm text-gray-600">Check failed workflows</p>
                  </div>
                </div>
              </Card>

              <Card className="p-6 hover:shadow-md transition-shadow cursor-pointer border-gray-200">
                <div className="flex items-center gap-4">
                  <div className="w-12 h-12 bg-blue-100 rounded-lg flex items-center justify-center">
                    <RefreshCw className="w-6 h-6 text-blue-600" />
                  </div>
                  <div>
                    <h3 className="font-medium text-gray-900">Re-run last failed</h3>
                    <p className="text-sm text-gray-600">Retry failed executions</p>
                  </div>
                </div>
              </Card>
            </div>
          </div>

          {/* Metrics Overview */}
          <div className="grid grid-cols-4 gap-6 mb-8">
            {metricsData.map((metric, index) => (
              <Card key={index} className="border-gray-200">
                <CardContent className="p-6">
                  <div className="flex items-center justify-between mb-4">
                    <div className="w-10 h-10 bg-gray-100 rounded-lg flex items-center justify-center">
                      <metric.icon className="w-5 h-5 text-gray-600" />
                    </div>
                    <div
                      className={`flex items-center gap-1 text-sm ${metric.trend === "up" ? "text-green-600" : "text-red-600"}`}
                    >
                      {metric.trend === "up" ? (
                        <TrendingUp className="w-3 h-3" />
                      ) : (
                        <TrendingDown className="w-3 h-3" />
                      )}
                      {metric.change}
                    </div>
                  </div>
                  <div className="text-2xl font-semibold text-gray-900 mb-1">{metric.value}</div>
                  <div className="text-sm text-gray-600">{metric.label}</div>
                </CardContent>
              </Card>
            ))}
          </div>

          <div className="grid grid-cols-3 gap-8">
            {/* Main Content Area */}
            <div className="col-span-2 space-y-8">
              {/* Charts Section */}
              <Card className="border-gray-200">
                <CardHeader className="pb-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <CardTitle className="text-lg font-semibold">Performance Analytics</CardTitle>
                      <CardDescription>Workflow execution trends and system metrics</CardDescription>
                    </div>
                    <Tabs defaultValue="workflows" className="w-auto">
                      <TabsList className="grid w-full grid-cols-3">
                        <TabsTrigger value="workflows">Workflows</TabsTrigger>
                        <TabsTrigger value="sales">Sales</TabsTrigger>
                        <TabsTrigger value="views">Views</TabsTrigger>
                      </TabsList>
                    </Tabs>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="h-80">
                    <ResponsiveContainer width="100%" height="100%">
                      <AreaChart data={chartData}>
                        <CartesianGrid strokeDasharray="3 3" stroke="#f3f4f6" />
                        <XAxis dataKey="name" stroke="#6b7280" fontSize={12} />
                        <YAxis stroke="#6b7280" fontSize={12} />
                        <Tooltip
                          contentStyle={{
                            backgroundColor: "white",
                            border: "1px solid #e5e7eb",
                            borderRadius: "8px",
                            boxShadow: "0 4px 6px -1px rgba(0, 0, 0, 0.1)",
                          }}
                        />
                        <Area
                          type="monotone"
                          dataKey="workflows"
                          stroke="#8b5cf6"
                          fill="#8b5cf6"
                          fillOpacity={0.1}
                          strokeWidth={2}
                        />
                        <Area
                          type="monotone"
                          dataKey="sales"
                          stroke="#3b82f6"
                          fill="#3b82f6"
                          fillOpacity={0.1}
                          strokeWidth={2}
                        />
                      </AreaChart>
                    </ResponsiveContainer>
                  </div>
                </CardContent>
              </Card>

              {/* Workflow Status Table */}
              <Card className="border-gray-200">
                <CardHeader className="pb-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <CardTitle className="text-lg font-semibold">Recent Workflow Runs</CardTitle>
                      <CardDescription>Monitor your workflow executions and performance</CardDescription>
                    </div>
                    <div className="flex items-center gap-2">
                      <Button variant="outline" size="sm">
                        <Filter className="w-4 h-4 mr-2" />
                        Filter
                      </Button>
                      <Button variant="outline" size="sm">
                        <Eye className="w-4 h-4 mr-2" />
                        View All
                      </Button>
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="p-0">
                  <Table>
                    <TableHeader>
                      <TableRow className="bg-gray-50">
                        <TableHead className="font-medium text-gray-700">Run ID</TableHead>
                        <TableHead className="font-medium text-gray-700">Workflow</TableHead>
                        <TableHead className="font-medium text-gray-700">Started</TableHead>
                        <TableHead className="font-medium text-gray-700">Duration</TableHead>
                        <TableHead className="font-medium text-gray-700">Status</TableHead>
                        <TableHead className="font-medium text-gray-700">Error</TableHead>
                        <TableHead className="font-medium text-gray-700 w-12"></TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {workflowData.map((workflow) => (
                        <TableRow key={workflow.id} className="hover:bg-gray-50">
                          <TableCell className="font-mono text-sm">{workflow.id}</TableCell>
                          <TableCell className="font-medium">{workflow.name}</TableCell>
                          <TableCell className="text-gray-600">{workflow.started}</TableCell>
                          <TableCell className="text-gray-600">{workflow.duration}</TableCell>
                          <TableCell>
                            {workflow.status === "running" && (
                              <Badge variant="secondary" className="bg-blue-100 text-blue-700">
                                <div className="w-2 h-2 bg-blue-500 rounded-full mr-2 animate-pulse"></div>
                                Running
                              </Badge>
                            )}
                            {workflow.status === "success" && (
                              <Badge variant="secondary" className="bg-green-100 text-green-700">
                                <CheckCircle className="w-3 h-3 mr-1" />
                                Success
                              </Badge>
                            )}
                            {workflow.status === "failed" && (
                              <Badge variant="secondary" className="bg-red-100 text-red-700">
                                <XCircle className="w-3 h-3 mr-1" />
                                Failed
                              </Badge>
                            )}
                          </TableCell>
                          <TableCell className="text-gray-600 max-w-48 truncate">{workflow.error || "None"}</TableCell>
                          <TableCell>
                            <DropdownMenu>
                              <DropdownMenuTrigger asChild>
                                <Button variant="ghost" size="icon" className="w-8 h-8">
                                  <MoreHorizontal className="w-4 h-4" />
                                </Button>
                              </DropdownMenuTrigger>
                              <DropdownMenuContent align="end">
                                <DropdownMenuItem>View Details</DropdownMenuItem>
                                <DropdownMenuItem>Re-run</DropdownMenuItem>
                                <DropdownMenuItem>View Logs</DropdownMenuItem>
                                <DropdownMenuSeparator />
                                <DropdownMenuItem className="text-red-600">Cancel</DropdownMenuItem>
                              </DropdownMenuContent>
                            </DropdownMenu>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </CardContent>
              </Card>
            </div>

            {/* Right Sidebar */}
            <div className="space-y-6">
              {/* Account Balance */}
              <Card className="border-gray-200">
                <CardHeader className="pb-4">
                  <CardTitle className="text-lg font-semibold">Account Balance</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-3xl font-semibold text-gray-900 mb-4">$1,423.25</div>
                  <div className="space-y-3 mb-4">
                    <div className="flex justify-between items-center">
                      <span className="text-sm text-gray-600">Monthly Credits</span>
                      <span className="text-sm font-medium">$500.00</span>
                    </div>
                    <div className="flex justify-between items-center">
                      <span className="text-sm text-gray-600">Usage This Month</span>
                      <span className="text-sm font-medium">$76.75</span>
                    </div>
                    <Progress value={15} className="h-2" />
                  </div>
                  <div className="flex gap-2">
                    <Button size="sm" className="flex-1">
                      Add Credit
                    </Button>
                    <Button size="sm" variant="outline" className="flex-1 bg-transparent">
                      Transfer
                    </Button>
                  </div>
                </CardContent>
              </Card>

              {/* Recent Activity */}
              <Card className="border-gray-200">
                <CardHeader className="pb-4">
                  <CardTitle className="text-lg font-semibold">Recent Activity</CardTitle>
                </CardHeader>
                <CardContent className="p-0">
                  <div className="space-y-0">
                    {recentActivity.map((activity, index) => (
                      <div
                        key={index}
                        className="flex items-center gap-3 p-4 hover:bg-gray-50 border-b border-gray-100 last:border-b-0"
                      >
                        <div
                          className={`w-2 h-2 rounded-full ${activity.status === "success" ? "bg-green-500" : "bg-red-500"}`}
                        ></div>
                        <div className="flex-1 min-w-0">
                          <div className="font-medium text-sm text-gray-900 truncate">{activity.workflow}</div>
                          <div className="text-xs text-gray-600">
                            {activity.time} • {activity.duration}
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>

              {/* Team Status */}
              <Card className="border-gray-200">
                <CardHeader className="pb-4">
                  <CardTitle className="text-lg font-semibold">Team Status</CardTitle>
                </CardHeader>
                <CardContent className="p-0">
                  <div className="space-y-0">
                    {teamMembers.map((member, index) => (
                      <div
                        key={index}
                        className="flex items-center gap-3 p-4 hover:bg-gray-50 border-b border-gray-100 last:border-b-0"
                      >
                        <div className="relative">
                          <Avatar className="w-8 h-8">
                            <AvatarImage src={member.avatar || "/placeholder.svg"} />
                            <AvatarFallback>
                              {member.name
                                .split(" ")
                                .map((n) => n[0])
                                .join("")}
                            </AvatarFallback>
                          </Avatar>
                          <div
                            className={`absolute -bottom-0.5 -right-0.5 w-3 h-3 rounded-full border-2 border-white ${
                              member.status === "online" ? "bg-green-500" : "bg-gray-400"
                            }`}
                          ></div>
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="font-medium text-sm text-gray-900">{member.name}</div>
                          <div className="text-xs text-gray-600">
                            {member.role} • {member.availability}
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            </div>
          </div>
        </main>
      </div>
    </div>
  )
}
