"use client"

import { useState } from "react"
import { DashboardLayout } from "@/components/dashboard-layout"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Textarea } from "@/components/ui/textarea"
import { Label } from "@/components/ui/label"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Spinner } from "@/components/ui/spinner"
import {
  Mail,
  Save,
  Send,
  Eye,
  RotateCcw,
  UserPlus,
  Key,
  Shield,
  Bell,
  LogIn,
  Network,
  CheckCircle,
} from "lucide-react"

interface EmailTemplate {
  id: string
  name: string
  description: string
  subject: string
  body: string
  variables: string[]
  category: "account" | "security" | "vpn" | "system"
  icon: React.ElementType
  lastModified?: string
}

const defaultTemplates: EmailTemplate[] = [
  {
    id: "welcome",
    name: "Welcome / Account Created",
    description: "Sent when a new user account is created. Includes login credentials.",
    subject: "Welcome to {{portal_name}} - Your Account Details",
    body: `Hello {{first_name}},

Your account has been created on {{portal_name}}.

Here are your login details:

  Portal URL:  {{portal_url}}
  Username:    {{username}}
  Password:    {{password}}

{{#if temporary_password}}
This is a temporary password. You will be required to change it on your first login.
{{/if}}

If you have any questions, please contact your administrator.

Best regards,
{{portal_name}} Team`,
    variables: ["first_name", "username", "password", "portal_name", "portal_url", "temporary_password"],
    category: "account",
    icon: UserPlus,
    lastModified: "2 days ago",
  },
  {
    id: "password_reset",
    name: "Password Reset",
    description: "Sent when an administrator resets a user's password.",
    subject: "{{portal_name}} - Your Password Has Been Reset",
    body: `Hello {{first_name}},

Your password on {{portal_name}} has been reset by an administrator.

  Username:     {{username}}
  New Password: {{password}}

{{#if temporary_password}}
This is a temporary password. You will be required to change it on your next login.
{{/if}}

If you did not request this change, please contact your administrator immediately.

Best regards,
{{portal_name}} Team`,
    variables: ["first_name", "username", "password", "portal_name", "temporary_password"],
    category: "security",
    icon: Key,
    lastModified: "5 days ago",
  },
  {
    id: "account_enabled",
    name: "Account Enabled",
    description: "Sent when a user account is re-enabled by an administrator.",
    subject: "{{portal_name}} - Your Account Has Been Activated",
    body: `Hello {{first_name}},

Your account on {{portal_name}} has been activated. You can now log in.

  Portal URL: {{portal_url}}
  Username:   {{username}}

If you have any questions, please contact your administrator.

Best regards,
{{portal_name}} Team`,
    variables: ["first_name", "username", "portal_name", "portal_url"],
    category: "account",
    icon: CheckCircle,
  },
  {
    id: "account_disabled",
    name: "Account Disabled",
    description: "Sent when a user account is disabled by an administrator.",
    subject: "{{portal_name}} - Your Account Has Been Suspended",
    body: `Hello {{first_name}},

Your account on {{portal_name}} has been suspended.

If you believe this is a mistake, please contact your administrator.

Best regards,
{{portal_name}} Team`,
    variables: ["first_name", "portal_name"],
    category: "account",
    icon: Shield,
  },
  {
    id: "vpn_credentials",
    name: "VPN Account Created",
    description: "Sent when a new VPN user account is created. Includes VPN connection details.",
    subject: "{{portal_name}} - Your VPN Access Details",
    body: `Hello {{first_name}},

Your VPN access has been configured on {{portal_name}}.

  VPN Server:  {{vpn_server}}
  Username:    {{username}}
  Password:    {{password}}
  Group:       {{vpn_group}}

Download the VPN client and import your configuration profile to get started.

  Download Link: {{vpn_download_url}}

If you have any questions, please contact your administrator.

Best regards,
{{portal_name}} Team`,
    variables: ["first_name", "username", "password", "portal_name", "vpn_server", "vpn_group", "vpn_download_url"],
    category: "vpn",
    icon: Network,
    lastModified: "1 week ago",
  },
  {
    id: "vpn_access_revoked",
    name: "VPN Access Revoked",
    description: "Sent when a user's VPN access is removed.",
    subject: "{{portal_name}} - Your VPN Access Has Been Revoked",
    body: `Hello {{first_name}},

Your VPN access on {{portal_name}} has been revoked.

If you believe this is a mistake, please contact your administrator.

Best regards,
{{portal_name}} Team`,
    variables: ["first_name", "portal_name"],
    category: "vpn",
    icon: Network,
  },
  {
    id: "security_alert",
    name: "Security Alert",
    description: "Sent when suspicious activity is detected on a user account.",
    subject: "{{portal_name}} - Security Alert on Your Account",
    body: `Hello {{first_name}},

We detected suspicious activity on your {{portal_name}} account.

  Event:       {{event_type}}
  Time:        {{event_time}}
  IP Address:  {{ip_address}}
  Location:    {{location}}

If this was not you, please contact your administrator immediately and change your password.

Best regards,
{{portal_name}} Security Team`,
    variables: ["first_name", "portal_name", "event_type", "event_time", "ip_address", "location"],
    category: "security",
    icon: Bell,
  },
  {
    id: "session_expired",
    name: "Session Expired Notification",
    description: "Sent when all user sessions are revoked by an administrator.",
    subject: "{{portal_name}} - Your Sessions Have Been Terminated",
    body: `Hello {{first_name}},

All your active sessions on {{portal_name}} have been terminated by an administrator.

You will need to log in again to continue using the portal.

  Portal URL: {{portal_url}}

If you have any questions, please contact your administrator.

Best regards,
{{portal_name}} Team`,
    variables: ["first_name", "portal_name", "portal_url"],
    category: "security",
    icon: LogIn,
  },
]

const CATEGORY_COLORS: Record<EmailTemplate["category"], string> = {
  account: "bg-emerald-50 text-emerald-700 border-emerald-200",
  security: "bg-amber-50 text-amber-700 border-amber-200",
  vpn: "bg-teal-50 text-teal-700 border-teal-200",
  system: "bg-slate-50 text-slate-600 border-slate-200",
}

export default function EmailTemplatesPage() {
  const [templates, setTemplates] = useState<EmailTemplate[]>(defaultTemplates)
  const [selectedTemplate, setSelectedTemplate] = useState<EmailTemplate>(defaultTemplates[0])
  const [editingTemplate, setEditingTemplate] = useState<EmailTemplate>(defaultTemplates[0])
  const [isPreviewOpen, setIsPreviewOpen] = useState(false)
  const [isSendTestOpen, setIsSendTestOpen] = useState(false)
  const [testEmail, setTestEmail] = useState("")
  const [isSaving, setIsSaving] = useState(false)
  const [isSendingTest, setIsSendingTest] = useState(false)
  const [savedId, setSavedId] = useState<string | null>(null)

  const selectTemplate = (template: EmailTemplate) => {
    setSelectedTemplate(template)
    setEditingTemplate({ ...template })
  }

  const handleSave = async () => {
    setIsSaving(true)
    await new Promise((r) => setTimeout(r, 800))
    setTemplates((prev) => prev.map((t) => (t.id === editingTemplate.id ? { ...editingTemplate, lastModified: "just now" } : t)))
    setSelectedTemplate({ ...editingTemplate, lastModified: "just now" })
    setSavedId(editingTemplate.id)
    setTimeout(() => setSavedId(null), 2000)
    setIsSaving(false)
  }

  const handleReset = () => {
    const original = defaultTemplates.find((t) => t.id === selectedTemplate.id)
    if (original) {
      setEditingTemplate({ ...original })
    }
  }

  const handleSendTest = async () => {
    if (!testEmail) return
    setIsSendingTest(true)
    await new Promise((r) => setTimeout(r, 1200))
    setIsSendingTest(false)
    setIsSendTestOpen(false)
    setTestEmail("")
  }

  const renderPreview = (template: EmailTemplate) => {
    const sampleVars: Record<string, string> = {
      first_name: "John",
      username: "john.doe",
      password: "Temp@2024!",
      portal_name: "Admin Portal",
      portal_url: "https://admin.company.com",
      vpn_server: "vpn.company.com",
      vpn_group: "developers",
      vpn_download_url: "https://vpn.company.com/client",
      temporary_password: "true",
      event_type: "Failed login attempt (5 times)",
      event_time: "2024-01-15 14:32:00",
      ip_address: "198.51.100.22",
      location: "Unknown location",
    }
    let rendered = template.body
    Object.entries(sampleVars).forEach(([key, val]) => {
      rendered = rendered.replace(new RegExp(`{{${key}}}`, "g"), val)
    })
    rendered = rendered.replace(/{{#if .+?}}([\s\S]*?){{\/if}}/g, "$1")
    return rendered
  }

  const groupedTemplates = {
    account: templates.filter((t) => t.category === "account"),
    security: templates.filter((t) => t.category === "security"),
    vpn: templates.filter((t) => t.category === "vpn"),
  }

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-foreground">Email Templates</h1>
            <p className="text-muted-foreground">
              Customize email templates for account creation, password resets, VPN access, and more
            </p>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Template List */}
          <div className="space-y-4">
            {Object.entries(groupedTemplates).map(([category, items]) => (
              <div key={category}>
                <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2 px-1">
                  {category} emails
                </h3>
                <div className="space-y-1">
                  {items.map((template) => {
                    const Icon = template.icon
                    const isActive = selectedTemplate.id === template.id
                    return (
                      <button
                        key={template.id}
                        onClick={() => selectTemplate(template)}
                        className={`w-full flex items-start gap-3 px-3 py-3 rounded-lg text-left transition-colors ${
                          isActive ? "bg-emerald-50 border border-emerald-200" : "hover:bg-muted border border-transparent"
                        }`}
                      >
                        <div className={`p-1.5 rounded mt-0.5 ${isActive ? "bg-emerald-100" : "bg-muted"}`}>
                          <Icon className={`w-3.5 h-3.5 ${isActive ? "text-emerald-700" : "text-muted-foreground"}`} />
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center justify-between gap-2">
                            <p className={`text-sm font-medium truncate ${isActive ? "text-emerald-800" : "text-foreground"}`}>
                              {template.name}
                            </p>
                          </div>
                          {template.lastModified && (
                            <p className="text-xs text-muted-foreground mt-0.5">
                              Edited {template.lastModified}
                            </p>
                          )}
                        </div>
                      </button>
                    )
                  })}
                </div>
              </div>
            ))}
          </div>

          {/* Template Editor */}
          <div className="lg:col-span-2 space-y-4">
            <Card>
              <CardHeader className="pb-4">
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle className="text-base">{selectedTemplate.name}</CardTitle>
                    <CardDescription>{selectedTemplate.description}</CardDescription>
                  </div>
                  <Badge variant="outline" className={`text-xs ${CATEGORY_COLORS[selectedTemplate.category]}`}>
                    {selectedTemplate.category}
                  </Badge>
                </div>
              </CardHeader>
              <CardContent className="space-y-4">
                <Tabs defaultValue="edit">
                  <TabsList className="grid w-full grid-cols-2 max-w-xs">
                    <TabsTrigger value="edit">Edit</TabsTrigger>
                    <TabsTrigger value="variables">Variables</TabsTrigger>
                  </TabsList>

                  <TabsContent value="edit" className="space-y-4 mt-4">
                    <div className="space-y-2">
                      <Label htmlFor="subject">Subject Line</Label>
                      <Input
                        id="subject"
                        value={editingTemplate.subject}
                        onChange={(e) => setEditingTemplate((prev) => ({ ...prev, subject: e.target.value }))}
                        placeholder="Email subject..."
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="body">Email Body</Label>
                      <Textarea
                        id="body"
                        value={editingTemplate.body}
                        onChange={(e) => setEditingTemplate((prev) => ({ ...prev, body: e.target.value }))}
                        className="font-mono text-sm min-h-[320px] resize-y"
                        placeholder="Email body..."
                      />
                      <p className="text-xs text-muted-foreground">
                        Use {"{{variable_name}}"} for dynamic values. Use {"{{#if var}}...{{/if}}"} for conditional blocks.
                      </p>
                    </div>
                  </TabsContent>

                  <TabsContent value="variables" className="mt-4">
                    <div className="space-y-3">
                      <p className="text-sm text-muted-foreground">
                        The following variables are available in this template:
                      </p>
                      <div className="grid grid-cols-2 gap-2">
                        {selectedTemplate.variables.map((variable) => (
                          <div
                            key={variable}
                            className="flex items-center justify-between p-2.5 bg-muted rounded-lg"
                          >
                            <code className="text-xs font-mono text-foreground">{`{{${variable}}}`}</code>
                            <button
                              className="text-xs text-muted-foreground hover:text-foreground"
                              onClick={() => {
                                navigator.clipboard.writeText(`{{${variable}}}`)
                              }}
                            >
                              Copy
                            </button>
                          </div>
                        ))}
                      </div>
                    </div>
                  </TabsContent>
                </Tabs>

                <div className="flex items-center gap-2 pt-2 border-t flex-wrap">
                  <Button
                    onClick={handleSave}
                    disabled={isSaving}
                    className="bg-emerald-600 hover:bg-emerald-700 gap-2"
                  >
                    {isSaving ? <Spinner className="w-4 h-4" /> : savedId === editingTemplate.id ? <CheckCircle className="w-4 h-4" /> : <Save className="w-4 h-4" />}
                    {savedId === editingTemplate.id ? "Saved" : "Save Template"}
                  </Button>
                  <Button
                    variant="outline"
                    onClick={() => setIsPreviewOpen(true)}
                    className="gap-2"
                  >
                    <Eye className="w-4 h-4" />
                    Preview
                  </Button>
                  <Button
                    variant="outline"
                    onClick={() => setIsSendTestOpen(true)}
                    className="gap-2"
                  >
                    <Send className="w-4 h-4" />
                    Send Test
                  </Button>
                  <Button
                    variant="ghost"
                    onClick={handleReset}
                    className="gap-2 text-muted-foreground"
                  >
                    <RotateCcw className="w-4 h-4" />
                    Reset to Default
                  </Button>
                </div>
              </CardContent>
            </Card>
          </div>
        </div>

        {/* Preview Dialog */}
        <Dialog open={isPreviewOpen} onOpenChange={setIsPreviewOpen}>
          <DialogContent className="sm:max-w-2xl">
            <DialogHeader>
              <DialogTitle>Email Preview</DialogTitle>
              <DialogDescription>
                Preview with sample variable values
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-3">
              <div className="p-3 bg-muted rounded-lg">
                <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">Subject</p>
                <p className="text-sm font-medium">
                  {editingTemplate.subject.replace(/{{(\w+)}}/g, (_, key) => {
                    const samples: Record<string, string> = {
                      portal_name: "Admin Portal",
                      first_name: "John",
                      username: "john.doe",
                    }
                    return samples[key] || `[${key}]`
                  })}
                </p>
              </div>
              <div className="p-4 bg-white border rounded-lg">
                <pre className="text-sm text-slate-700 whitespace-pre-wrap font-sans leading-relaxed">
                  {renderPreview(editingTemplate)}
                </pre>
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setIsPreviewOpen(false)}>Close</Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* Send Test Dialog */}
        <Dialog open={isSendTestOpen} onOpenChange={setIsSendTestOpen}>
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <DialogTitle>Send Test Email</DialogTitle>
              <DialogDescription>
                Send a test email for &quot;{selectedTemplate.name}&quot; with sample variable values
              </DialogDescription>
            </DialogHeader>
            <div className="py-4 space-y-3">
              <div className="space-y-2">
                <Label htmlFor="test-email">Recipient Email</Label>
                <Input
                  id="test-email"
                  type="email"
                  value={testEmail}
                  onChange={(e) => setTestEmail(e.target.value)}
                  placeholder="admin@company.com"
                />
              </div>
              <p className="text-xs text-muted-foreground">
                The email will be sent with sample values for all template variables.
              </p>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setIsSendTestOpen(false)}>
                Cancel
              </Button>
              <Button
                onClick={handleSendTest}
                disabled={isSendingTest || !testEmail}
                className="bg-emerald-600 hover:bg-emerald-700 gap-2"
              >
                {isSendingTest ? <Spinner className="w-4 h-4" /> : <Send className="w-4 h-4" />}
                Send Test
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
    </DashboardLayout>
  )
}
