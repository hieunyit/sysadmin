"use client"

import { useState } from "react"
import { Save, Server, Key, Globe, Database, Shield, Bell, RefreshCw, TestTube, CheckCircle, XCircle, AlertTriangle } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Badge } from "@/components/ui/badge"
import { DashboardLayout } from "@/components/dashboard-layout"
import { Spinner } from "@/components/ui/spinner"

export default function SettingsPage() {
  const [isTesting, setIsTesting] = useState<string | null>(null)
  const [connectionStatus, setConnectionStatus] = useState<Record<string, "success" | "error" | null>>({
    keycloak: null,
    openvpn: null,
  })

  const [keycloakSettings, setKeycloakSettings] = useState({
    serverUrl: "https://keycloak.example.com",
    realm: "master",
    clientId: "admin-cli",
    clientSecret: "••••••••••••••••",
    enabled: true,
  })

  const [openvpnSettings, setOpenvpnSettings] = useState({
    serverUrl: "https://openvpn.example.com:943",
    username: "openvpn_admin",
    password: "••••••••••••••••",
    enabled: true,
  })

  const [notificationSettings, setNotificationSettings] = useState({
    userLogin: true,
    userCreated: true,
    userDeleted: true,
    vpnConnect: false,
    vpnDisconnect: false,
    sessionExpired: true,
    securityAlerts: true,
  })

  const testConnection = async (service: "keycloak" | "openvpn") => {
    setIsTesting(service)
    setConnectionStatus((prev) => ({ ...prev, [service]: null }))

    await new Promise((resolve) => setTimeout(resolve, 1500))

    const success = Math.random() > 0.3
    setConnectionStatus((prev) => ({ ...prev, [service]: success ? "success" : "error" }))
    setIsTesting(null)
  }

  const getStatusBadge = (status: "success" | "error" | null) => {
    if (status === "success") {
      return (
        <Badge className="bg-emerald-100 text-emerald-700 hover:bg-emerald-100">
          <CheckCircle className="w-3 h-3 mr-1" />
          Connected
        </Badge>
      )
    }
    if (status === "error") {
      return (
        <Badge className="bg-red-100 text-red-700 hover:bg-red-100">
          <XCircle className="w-3 h-3 mr-1" />
          Failed
        </Badge>
      )
    }
    return null
  }

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-semibold text-slate-900">Settings</h1>
          <p className="text-slate-500">Configure Keycloak and OpenVPN connections</p>
        </div>

        <Tabs defaultValue="keycloak" className="space-y-6">
          <TabsList className="grid w-full grid-cols-4 max-w-2xl">
            <TabsTrigger value="keycloak" className="gap-2">
              <Key className="w-4 h-4" />
              Keycloak
            </TabsTrigger>
            <TabsTrigger value="openvpn" className="gap-2">
              <Globe className="w-4 h-4" />
              OpenVPN
            </TabsTrigger>
            <TabsTrigger value="notifications" className="gap-2">
              <Bell className="w-4 h-4" />
              Notifications
            </TabsTrigger>
            <TabsTrigger value="general" className="gap-2">
              <Server className="w-4 h-4" />
              General
            </TabsTrigger>
          </TabsList>

          <TabsContent value="keycloak" className="space-y-6">
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle className="flex items-center gap-2">
                      <Key className="w-5 h-5 text-emerald-600" />
                      Keycloak Configuration
                    </CardTitle>
                    <CardDescription>Configure your Keycloak server connection</CardDescription>
                  </div>
                  <div className="flex items-center gap-3">
                    {getStatusBadge(connectionStatus.keycloak)}
                    <Switch
                      checked={keycloakSettings.enabled}
                      onCheckedChange={(value) => setKeycloakSettings((prev) => ({ ...prev, enabled: value }))}
                    />
                  </div>
                </div>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <div className="space-y-2">
                    <Label htmlFor="keycloak-url">Server URL</Label>
                    <Input
                      id="keycloak-url"
                      value={keycloakSettings.serverUrl}
                      onChange={(e) => setKeycloakSettings((prev) => ({ ...prev, serverUrl: e.target.value }))}
                      placeholder="https://keycloak.example.com"
                    />
                    <p className="text-xs text-slate-500">The base URL of your Keycloak server</p>
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="keycloak-realm">Realm</Label>
                    <Input
                      id="keycloak-realm"
                      value={keycloakSettings.realm}
                      onChange={(e) => setKeycloakSettings((prev) => ({ ...prev, realm: e.target.value }))}
                      placeholder="master"
                    />
                    <p className="text-xs text-slate-500">The Keycloak realm to manage</p>
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="keycloak-client-id">Client ID</Label>
                    <Input
                      id="keycloak-client-id"
                      value={keycloakSettings.clientId}
                      onChange={(e) => setKeycloakSettings((prev) => ({ ...prev, clientId: e.target.value }))}
                      placeholder="admin-cli"
                    />
                    <p className="text-xs text-slate-500">Service account client ID</p>
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="keycloak-secret">Client Secret</Label>
                    <Input
                      id="keycloak-secret"
                      type="password"
                      value={keycloakSettings.clientSecret}
                      onChange={(e) => setKeycloakSettings((prev) => ({ ...prev, clientSecret: e.target.value }))}
                      placeholder="Enter client secret"
                    />
                    <p className="text-xs text-slate-500">Service account client secret</p>
                  </div>
                </div>

                <div className="flex items-center gap-3 pt-4 border-t">
                  <Button
                    variant="outline"
                    onClick={() => testConnection("keycloak")}
                    disabled={isTesting === "keycloak"}
                    className="gap-2"
                  >
                    {isTesting === "keycloak" ? (
                      <Spinner className="w-4 h-4" />
                    ) : (
                      <TestTube className="w-4 h-4" />
                    )}
                    Test Connection
                  </Button>
                  <Button className="bg-emerald-600 hover:bg-emerald-700 gap-2">
                    <Save className="w-4 h-4" />
                    Save Configuration
                  </Button>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="text-base">Advanced Settings</CardTitle>
                <CardDescription>Additional Keycloak configuration options</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <div className="font-medium text-slate-900">Auto-sync Users</div>
                    <div className="text-sm text-slate-500">Automatically sync users from Keycloak every hour</div>
                  </div>
                  <Switch defaultChecked />
                </div>
                <div className="flex items-center justify-between">
                  <div>
                    <div className="font-medium text-slate-900">Cache User Data</div>
                    <div className="text-sm text-slate-500">Cache user data locally for faster lookups</div>
                  </div>
                  <Switch defaultChecked />
                </div>
                <div className="flex items-center justify-between">
                  <div>
                    <div className="font-medium text-slate-900">Enable Debug Logging</div>
                    <div className="text-sm text-slate-500">Log detailed API requests for debugging</div>
                  </div>
                  <Switch />
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="openvpn" className="space-y-6">
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle className="flex items-center gap-2">
                      <Globe className="w-5 h-5 text-teal-600" />
                      OpenVPN Configuration
                    </CardTitle>
                    <CardDescription>Configure your OpenVPN Access Server connection</CardDescription>
                  </div>
                  <div className="flex items-center gap-3">
                    {getStatusBadge(connectionStatus.openvpn)}
                    <Switch
                      checked={openvpnSettings.enabled}
                      onCheckedChange={(value) => setOpenvpnSettings((prev) => ({ ...prev, enabled: value }))}
                    />
                  </div>
                </div>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <div className="md:col-span-2 space-y-2">
                    <Label htmlFor="openvpn-url">Server URL</Label>
                    <Input
                      id="openvpn-url"
                      value={openvpnSettings.serverUrl}
                      onChange={(e) => setOpenvpnSettings((prev) => ({ ...prev, serverUrl: e.target.value }))}
                      placeholder="https://openvpn.example.com:943"
                    />
                    <p className="text-xs text-slate-500">The base URL of your OpenVPN Access Server admin interface</p>
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="openvpn-user">Admin Username</Label>
                    <Input
                      id="openvpn-user"
                      value={openvpnSettings.username}
                      onChange={(e) => setOpenvpnSettings((prev) => ({ ...prev, username: e.target.value }))}
                      placeholder="openvpn_admin"
                    />
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="openvpn-pass">Admin Password</Label>
                    <Input
                      id="openvpn-pass"
                      type="password"
                      value={openvpnSettings.password}
                      onChange={(e) => setOpenvpnSettings((prev) => ({ ...prev, password: e.target.value }))}
                      placeholder="Enter password"
                    />
                  </div>
                </div>

                <div className="flex items-center gap-3 pt-4 border-t">
                  <Button
                    variant="outline"
                    onClick={() => testConnection("openvpn")}
                    disabled={isTesting === "openvpn"}
                    className="gap-2"
                  >
                    {isTesting === "openvpn" ? (
                      <Spinner className="w-4 h-4" />
                    ) : (
                      <TestTube className="w-4 h-4" />
                    )}
                    Test Connection
                  </Button>
                  <Button className="bg-emerald-600 hover:bg-emerald-700 gap-2">
                    <Save className="w-4 h-4" />
                    Save Configuration
                  </Button>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="text-base">VPN Settings</CardTitle>
                <CardDescription>Default VPN configuration options</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <div className="font-medium text-slate-900">Auto-generate Configs</div>
                    <div className="text-sm text-slate-500">Automatically create VPN configs for new users</div>
                  </div>
                  <Switch defaultChecked />
                </div>
                <div className="flex items-center justify-between">
                  <div>
                    <div className="font-medium text-slate-900">Sync with Keycloak</div>
                    <div className="text-sm text-slate-500">Sync VPN access with Keycloak group membership</div>
                  </div>
                  <Switch defaultChecked />
                </div>
                <div className="flex items-center justify-between">
                  <div>
                    <div className="font-medium text-slate-900">Certificate Expiry Alerts</div>
                    <div className="text-sm text-slate-500">Alert when user certificates are about to expire</div>
                  </div>
                  <Switch defaultChecked />
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="notifications" className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Bell className="w-5 h-5 text-blue-600" />
                  Notification Settings
                </CardTitle>
                <CardDescription>Configure which events trigger notifications</CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="space-y-4">
                  <h3 className="font-medium text-slate-900">User Events</h3>
                  <div className="space-y-4">
                    <div className="flex items-center justify-between">
                      <div>
                        <div className="font-medium">User Login</div>
                        <div className="text-sm text-slate-500">When a user logs into Keycloak</div>
                      </div>
                      <Switch
                        checked={notificationSettings.userLogin}
                        onCheckedChange={(value) =>
                          setNotificationSettings((prev) => ({ ...prev, userLogin: value }))
                        }
                      />
                    </div>
                    <div className="flex items-center justify-between">
                      <div>
                        <div className="font-medium">User Created</div>
                        <div className="text-sm text-slate-500">When a new user is created</div>
                      </div>
                      <Switch
                        checked={notificationSettings.userCreated}
                        onCheckedChange={(value) =>
                          setNotificationSettings((prev) => ({ ...prev, userCreated: value }))
                        }
                      />
                    </div>
                    <div className="flex items-center justify-between">
                      <div>
                        <div className="font-medium">User Deleted</div>
                        <div className="text-sm text-slate-500">When a user is removed from the system</div>
                      </div>
                      <Switch
                        checked={notificationSettings.userDeleted}
                        onCheckedChange={(value) =>
                          setNotificationSettings((prev) => ({ ...prev, userDeleted: value }))
                        }
                      />
                    </div>
                  </div>
                </div>

                <div className="space-y-4">
                  <h3 className="font-medium text-slate-900">VPN Events</h3>
                  <div className="space-y-4">
                    <div className="flex items-center justify-between">
                      <div>
                        <div className="font-medium">VPN Connect</div>
                        <div className="text-sm text-slate-500">When a user connects to VPN</div>
                      </div>
                      <Switch
                        checked={notificationSettings.vpnConnect}
                        onCheckedChange={(value) =>
                          setNotificationSettings((prev) => ({ ...prev, vpnConnect: value }))
                        }
                      />
                    </div>
                    <div className="flex items-center justify-between">
                      <div>
                        <div className="font-medium">VPN Disconnect</div>
                        <div className="text-sm text-slate-500">When a user disconnects from VPN</div>
                      </div>
                      <Switch
                        checked={notificationSettings.vpnDisconnect}
                        onCheckedChange={(value) =>
                          setNotificationSettings((prev) => ({ ...prev, vpnDisconnect: value }))
                        }
                      />
                    </div>
                  </div>
                </div>

                <div className="space-y-4">
                  <h3 className="font-medium text-slate-900">System Events</h3>
                  <div className="space-y-4">
                    <div className="flex items-center justify-between">
                      <div>
                        <div className="font-medium">Session Expired</div>
                        <div className="text-sm text-slate-500">When user sessions expire</div>
                      </div>
                      <Switch
                        checked={notificationSettings.sessionExpired}
                        onCheckedChange={(value) =>
                          setNotificationSettings((prev) => ({ ...prev, sessionExpired: value }))
                        }
                      />
                    </div>
                    <div className="flex items-center justify-between">
                      <div>
                        <div className="font-medium">Security Alerts</div>
                        <div className="text-sm text-slate-500">Critical security notifications</div>
                      </div>
                      <Switch
                        checked={notificationSettings.securityAlerts}
                        onCheckedChange={(value) =>
                          setNotificationSettings((prev) => ({ ...prev, securityAlerts: value }))
                        }
                      />
                    </div>
                  </div>
                </div>

                <div className="flex justify-end pt-4 border-t">
                  <Button className="bg-emerald-600 hover:bg-emerald-700 gap-2">
                    <Save className="w-4 h-4" />
                    Save Preferences
                  </Button>
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="general" className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Database className="w-5 h-5 text-slate-600" />
                  General Settings
                </CardTitle>
                <CardDescription>Application-wide configuration options</CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="font-medium text-slate-900">Dark Mode</div>
                      <div className="text-sm text-slate-500">Enable dark mode interface</div>
                    </div>
                    <Switch />
                  </div>
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="font-medium text-slate-900">Compact View</div>
                      <div className="text-sm text-slate-500">Use compact table layouts</div>
                    </div>
                    <Switch />
                  </div>
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="font-medium text-slate-900">Auto Refresh</div>
                      <div className="text-sm text-slate-500">Automatically refresh data every 30 seconds</div>
                    </div>
                    <Switch defaultChecked />
                  </div>
                </div>

                <div className="space-y-4 pt-6 border-t">
                  <h3 className="font-medium text-slate-900">Data Management</h3>
                  <div className="flex flex-wrap gap-3">
                    <Button variant="outline" className="gap-2">
                      <RefreshCw className="w-4 h-4" />
                      Clear Cache
                    </Button>
                    <Button variant="outline" className="gap-2">
                      <Database className="w-4 h-4" />
                      Export Data
                    </Button>
                  </div>
                </div>

                <div className="p-4 rounded-lg bg-amber-50 border border-amber-200">
                  <div className="flex items-start gap-3">
                    <AlertTriangle className="w-5 h-5 text-amber-600 mt-0.5" />
                    <div>
                      <div className="font-medium text-amber-800">Environment Variables</div>
                      <p className="text-sm text-amber-700 mt-1">
                        Make sure to set the following environment variables for production:
                      </p>
                      <ul className="text-sm text-amber-700 mt-2 space-y-1 font-mono">
                        <li>• KEYCLOAK_URL</li>
                        <li>• KEYCLOAK_REALM</li>
                        <li>• KEYCLOAK_CLIENT_ID</li>
                        <li>• KEYCLOAK_CLIENT_SECRET</li>
                        <li>• OPENVPN_URL</li>
                        <li>• OPENVPN_USERNAME</li>
                        <li>• OPENVPN_PASSWORD</li>
                      </ul>
                    </div>
                  </div>
                </div>

                <div className="flex justify-end pt-4 border-t">
                  <Button className="bg-emerald-600 hover:bg-emerald-700 gap-2">
                    <Save className="w-4 h-4" />
                    Save Settings
                  </Button>
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </DashboardLayout>
  )
}
