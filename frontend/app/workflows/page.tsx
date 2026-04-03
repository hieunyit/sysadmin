"use client"

import { useState } from "react"
import { Play, Pause, MoreHorizontal, Filter, Plus, Search, CheckCircle, XCircle, Zap, Database } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu"
import { DashboardLayout } from "@/components/dashboard-layout"

const workflows = [
  {
    id: 1,
    name: "Product Catalog Sync",
    description: "Synchronizes product data across all platforms",
    status: "active",
    lastRun: "2 minutes ago",
    nextRun: "In 58 minutes",
    runs: 1247,
    successRate: 98.7,
    avgDuration: "45s",
  },
  {
    id: 2,
    name: "Customer Webhook Listener",
    description: "Processes incoming customer events",
    status: "active",
    lastRun: "5 minutes ago",
    nextRun: "Continuous",
    runs: 3421,
    successRate: 99.2,
    avgDuration: "12s",
  },
  {
    id: 3,
    name: "Data Enrichment Pipeline",
    description: "Enriches customer data with external sources",
    status: "paused",
    lastRun: "2 hours ago",
    nextRun: "Paused",
    runs: 892,
    successRate: 94.3,
    avgDuration: "2m 15s",
  },
  {
    id: 4,
    name: "Analytics Refresh",
    description: "Updates analytics dashboards and reports",
    status: "active",
    lastRun: "18 minutes ago",
    nextRun: "In 42 minutes",
    runs: 567,
    successRate: 99.8,
    avgDuration: "1m 8s",
  },
  {
    id: 5,
    name: "Billing Reconciliation",
    description: "Reconciles billing data with payment providers",
    status: "error",
    lastRun: "1 hour ago",
    nextRun: "Retry in 15m",
    runs: 234,
    successRate: 87.2,
    avgDuration: "3m 22s",
  },
]

const recentRuns = [
  { id: 6734, workflow: "Product Catalog Sync", status: "running", duration: "45s", started: "2 min ago" },
  { id: 6733, workflow: "Customer Webhook Listener", status: "success", duration: "12s", started: "5 min ago" },
  { id: 6732, workflow: "Analytics Refresh", status: "success", duration: "1m 8s", started: "18 min ago" },
  { id: 6731, workflow: "Data Enrichment Pipeline", status: "failed", duration: "2m 15s", started: "2 hrs ago" },
  { id: 6730, workflow: "Billing Reconciliation", status: "success", duration: "3m 22s", started: "1 hr ago" },
]

export default function WorkflowsPage() {
  const [searchQuery, setSearchQuery] = useState("")

  return (
    <DashboardLayout>
      <div className="space-y-8">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-gray-900">Workflows</h1>
            <p className="text-gray-600 mt-1">Manage and monitor your automation workflows</p>
          </div>
          <div className="flex items-center gap-3">
            <Button variant="outline" className="gap-2 bg-transparent">
              <Filter className="w-4 h-4" />
              Filter
            </Button>
            <Button className="bg-purple-600 hover:bg-purple-700 gap-2">
              <Plus className="w-4 h-4" />
              New Workflow
            </Button>
          </div>
        </div>

        {/* Search and Stats */}
        <div className="flex items-center gap-4">
          <div className="relative flex-1 max-w-md">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-4 h-4" />
            <Input
              placeholder="Search workflows..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-10"
            />
          </div>
          <div className="flex items-center gap-6 text-sm text-gray-600">
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 bg-green-500 rounded-full"></div>
              <span>4 Active</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 bg-yellow-500 rounded-full"></div>
              <span>1 Paused</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 bg-red-500 rounded-full"></div>
              <span>1 Error</span>
            </div>
          </div>
        </div>

        <Tabs defaultValue="workflows" className="space-y-6">
          <TabsList>
            <TabsTrigger value="workflows">All Workflows</TabsTrigger>
            <TabsTrigger value="runs">Recent Runs</TabsTrigger>
            <TabsTrigger value="templates">Templates</TabsTrigger>
          </TabsList>

          <TabsContent value="workflows" className="space-y-6">
            {/* Workflows Grid */}
            <div className="grid gap-6">
              {workflows.map((workflow) => (
                <Card key={workflow.id} className="border-gray-200">
                  <CardContent className="p-6">
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <div className="flex items-center gap-3 mb-2">
                          <h3 className="font-semibold text-gray-900">{workflow.name}</h3>
                          <Badge
                            variant="secondary"
                            className={
                              workflow.status === "active"
                                ? "bg-green-100 text-green-700"
                                : workflow.status === "paused"
                                  ? "bg-yellow-100 text-yellow-700"
                                  : "bg-red-100 text-red-700"
                            }
                          >
                            {workflow.status === "active" && <CheckCircle className="w-3 h-3 mr-1" />}
                            {workflow.status === "paused" && <Pause className="w-3 h-3 mr-1" />}
                            {workflow.status === "error" && <XCircle className="w-3 h-3 mr-1" />}
                            {workflow.status.charAt(0).toUpperCase() + workflow.status.slice(1)}
                          </Badge>
                        </div>
                        <p className="text-gray-600 text-sm mb-4">{workflow.description}</p>

                        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                          <div>
                            <span className="text-gray-500">Last Run</span>
                            <div className="font-medium">{workflow.lastRun}</div>
                          </div>
                          <div>
                            <span className="text-gray-500">Next Run</span>
                            <div className="font-medium">{workflow.nextRun}</div>
                          </div>
                          <div>
                            <span className="text-gray-500">Success Rate</span>
                            <div className="font-medium">{workflow.successRate}%</div>
                          </div>
                          <div>
                            <span className="text-gray-500">Avg Duration</span>
                            <div className="font-medium">{workflow.avgDuration}</div>
                          </div>
                        </div>
                      </div>

                      <div className="flex items-center gap-2">
                        {workflow.status === "active" && (
                          <Button size="sm" variant="outline" className="gap-2 bg-transparent">
                            <Pause className="w-3 h-3" />
                            Pause
                          </Button>
                        )}
                        {workflow.status === "paused" && (
                          <Button size="sm" variant="outline" className="gap-2 bg-transparent">
                            <Play className="w-3 h-3" />
                            Resume
                          </Button>
                        )}
                        {workflow.status === "error" && (
                          <Button size="sm" variant="outline" className="gap-2 bg-transparent">
                            <Zap className="w-3 h-3" />
                            Retry
                          </Button>
                        )}
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button size="sm" variant="ghost">
                              <MoreHorizontal className="w-4 h-4" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            <DropdownMenuItem>Edit Workflow</DropdownMenuItem>
                            <DropdownMenuItem>View Logs</DropdownMenuItem>
                            <DropdownMenuItem>Duplicate</DropdownMenuItem>
                            <DropdownMenuSeparator />
                            <DropdownMenuItem className="text-red-600">Delete</DropdownMenuItem>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          </TabsContent>

          <TabsContent value="runs" className="space-y-6">
            <Card className="border-gray-200">
              <CardHeader>
                <CardTitle>Recent Workflow Runs</CardTitle>
                <CardDescription>Monitor the latest workflow executions</CardDescription>
              </CardHeader>
              <CardContent className="p-0">
                <Table>
                  <TableHeader>
                    <TableRow className="bg-gray-50">
                      <TableHead>Run ID</TableHead>
                      <TableHead>Workflow</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Duration</TableHead>
                      <TableHead>Started</TableHead>
                      <TableHead className="w-12"></TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {recentRuns.map((run) => (
                      <TableRow key={run.id} className="hover:bg-gray-50">
                        <TableCell className="font-mono">{run.id}</TableCell>
                        <TableCell className="font-medium">{run.workflow}</TableCell>
                        <TableCell>
                          {run.status === "running" && (
                            <Badge variant="secondary" className="bg-blue-100 text-blue-700">
                              <div className="w-2 h-2 bg-blue-500 rounded-full mr-2 animate-pulse"></div>
                              Running
                            </Badge>
                          )}
                          {run.status === "success" && (
                            <Badge variant="secondary" className="bg-green-100 text-green-700">
                              <CheckCircle className="w-3 h-3 mr-1" />
                              Success
                            </Badge>
                          )}
                          {run.status === "failed" && (
                            <Badge variant="secondary" className="bg-red-100 text-red-700">
                              <XCircle className="w-3 h-3 mr-1" />
                              Failed
                            </Badge>
                          )}
                        </TableCell>
                        <TableCell>{run.duration}</TableCell>
                        <TableCell className="text-gray-600">{run.started}</TableCell>
                        <TableCell>
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button variant="ghost" size="icon" className="w-8 h-8">
                                <MoreHorizontal className="w-4 h-4" />
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                              <DropdownMenuItem>View Details</DropdownMenuItem>
                              <DropdownMenuItem>View Logs</DropdownMenuItem>
                              <DropdownMenuItem>Re-run</DropdownMenuItem>
                            </DropdownMenuContent>
                          </DropdownMenu>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="templates" className="space-y-6">
            <div className="text-center py-12">
              <div className="w-16 h-16 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-4">
                <Database className="w-8 h-8 text-gray-400" />
              </div>
              <h3 className="text-lg font-medium text-gray-900 mb-2">No Templates Yet</h3>
              <p className="text-gray-600 mb-4">Create reusable workflow templates to speed up your automation.</p>
              <Button className="bg-purple-600 hover:bg-purple-700">
                <Plus className="w-4 h-4 mr-2" />
                Create Template
              </Button>
            </div>
          </TabsContent>
        </Tabs>
      </div>
    </DashboardLayout>
  )
}
