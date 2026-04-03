"use client"

import { useState } from "react"
import {
  Save, Server, Key, Globe, Database, Shield, Bell, RefreshCw,
  TestTube, CheckCircle, XCircle, AlertTriangle, Mail, Send,
  Lock, Eye, EyeOff,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Badge } from "@/components/ui/badge"
import { Textarea } from "@/components/ui/textarea"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { DashboardLayout } from "@/components/dashboard-layout"
import { Spinner } from "@/components/ui/spinner"

export default function SettingsPage() {
  const [isTesting, setIsTesting] = useState<string | null>(null)
  const [showSmtpPassword, setShowSmtpPassword] = useState(false)
  const [connectionStatus, setConnectionStatus] = useState<Record<string, "success" | "error" | null>>({
    keycloak: null,
    openvpn: null,
    email: null,
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

  const [emailSettings, setEmailSettings] = useState({
    smtpHost: "",
    smtpPort: "587",
    smtpUsername: "",
    smtpPassword: "",
    fromAddress: "",
    fromName: "Admin Portal",
    encryption: "tls",
    enabled: false,
    sendWelcomeEmail: true,
    sendPasswordReset: true,
    sendAccountNotifications: true,
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

  const testConnection = async (service: "keycloak" | "openvpn" | "email") => {
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
          <TabsList className="grid w-full grid-cols-5 max-w-3xl">
            <TabsTrigger value="keycloak" className="gap-2">
              <Key className="w-4 h-4" />
              Keycloak
            </TabsTrigger>
            <TabsTrigger value="openvpn" className="gap-2">
              <Globe className="w-4 h-4" />
              OpenVPN
            </TabsTrigger>
            <TabsTrigger value="email" className="gap-2">
              <Mail className="w-4 h-4" />
              Email
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

          <TabsContent value="email" className="space-y-6">
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle className="flex items-center gap-2">
                      <Mail className="w-5 h-5 text-blue-600" />
                      Email (SMTP) Configuration
                    </CardTitle>
                    <CardDescription>
                      Configure SMTP to send account credentials, welcome emails, and notifications
                    </CardDescription>
                  </div>
                  <div className="flex items-center gap-3">
                    {getStatusBadge(connectionStatus.email)}
                    <Switch
                      checked={emailSettings.enabled}
                      onCheckedChange={(value) => setEmailSettings((prev) => ({ ...prev, enabled: value }))}
                    />
                  </div>
                </div>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <div className="space-y-2">
                    <Label htmlFor="smtp-host">SMTP Host</Label>
                    <Input
                      id="smtp-host"
                      value={emailSettings.smtpHost}
                      onChange={(e) => setEmailSettings((prev) => ({ ...prev, smtpHost: e.target.value }))}
                      placeholder="smtp.gmail.com"
                    />
                    <p className="text-xs text-slate-500">Hostname of your SMTP server</p>
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="smtp-port">SMTP Port</Label>
                    <Input
                      id="smtp-port"
                      value={emailSettings.smtpPort}
                      onChange={(e) => setEmailSettings((prev) => ({ ...prev, smtpPort: e.target.value }))}
                      placeholder="587"
                    />
                    <p className="text-xs text-slate-500">587 (TLS) or 465 (SSL)</p>
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="smtp-user">SMTP Username</Label>
                    <Input
                      id="smtp-user"
                      value={emailSettings.smtpUsername}
                      onChange={(e) => setEmailSettings((prev) => ({ ...prev, smtpUsername: e.target.value }))}
                      placeholder="noreply@company.com"
                    />
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="smtp-pass">SMTP Password</Label>
                    <div className="relative">
                      <Input
                        id="smtp-pass"
                        type={showSmtpPassword ? "text" : "password"}
                        value={emailSettings.smtpPassword}
                        onChange={(e) => setEmailSettings((prev) => ({ ...prev, smtpPassword: e.target.value }))}
                        placeholder="Enter SMTP password or app password"
                        className="pr-10"
                      />
                      <button
                        type="button"
                        className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600"
                        onClick={() => setShowSmtpPassword((v) => !v)}
                      >
                        {showSmtpPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                      </button>
                    </div>
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="from-address">From Address</Label>
                    <Input
                      id="from-address"
                      type="email"
                      value={emailSettings.fromAddress}
                      onChange={(e) => setEmailSettings((prev) => ({ ...prev, fromAddress: e.target.value }))}
                      placeholder="noreply@company.com"
                    />
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="from-name">From Name</Label>
                    <Input
                      id="from-name"
                      value={emailSettings.fromName}
                      onChange={(e) => setEmailSettings((prev) => ({ ...prev, fromName: e.target.value }))}
                      placeholder="Admin Portal"
                    />
                  </div>

                  <div className="md:col-span-2 space-y-2">
                    <Label htmlFor="encryption">Encryption</Label>
                    <Select
                      value={emailSettings.encryption}
                      onValueChange={(value) => setEmailSettings((prev) => ({ ...prev, encryption: value }))}
                    >
                      <SelectTrigger id="encryption" className="max-w-xs">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="tls">STARTTLS (port 587)</SelectItem>
                        <SelectItem value="ssl">SSL/TLS (port 465)</SelectItem>
                        <SelectItem value="none">None (port 25)</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                <div className="flex items-center gap-3 pt-4 border-t">
                  <Button
                    variant="outline"
                    onClick={() => testConnection("email")}
                    disabled={isTesting === "email"}
                    className="gap-2"
                  >
                    {isTesting === "email" ? (
                      <Spinner className="w-4 h-4" />
                    ) : (
                      <TestTube className="w-4 h-4" />
                    )}
                    Send Test Email
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
                <CardTitle className="text-base">Email Triggers</CardTitle>
                <CardDescription>Choose which events automatically send emails to users</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <div className="font-medium text-slate-900">Welcome Email</div>
                    <div className="text-sm text-slate-500">
                      Send login credentials when a new account is created
                    </div>
                  </div>
                  <Switch
                    checked={emailSettings.sendWelcomeEmail}
                    onCheckedChange={(v) => setEmailSettings((prev) => ({ ...prev, sendWelcomeEmail: v }))}
                  />
                </div>
                <div className="flex items-center justify-between">
                  <div>
                    <div className="font-medium text-slate-900">Password Reset</div>
                    <div className="text-sm text-slate-500">
                      Send password reset instructions when an admin resets a password
                    </div>
                  </div>
                  <Switch
                    checked={emailSettings.sendPasswordReset}
                    onCheckedChange={(v) => setEmailSettings((prev) => ({ ...prev, sendPasswordReset: v }))}
                  />
                </div>
                <div className="flex items-center justify-between">
                  <div>
                    <div className="font-medium text-slate-900">Account Notifications</div>
                    <div className="text-sm text-slate-500">
                      Notify users when their account is enabled or disabled
                    </div>
                  </div>
                  <Switch
                    checked={emailSettings.sendAccountNotifications}
                    onCheckedChange={(v) => setEmailSettings((prev) => ({ ...prev, sendAccountNotifications: v }))}
                  />
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
