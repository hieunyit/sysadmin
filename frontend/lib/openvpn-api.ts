// OpenVPN Access Server REST API Client
// Based on the OpenVPN Access Server Web API documentation

const OPENVPN_URL = process.env.OPENVPN_URL || ""
const OPENVPN_USERNAME = process.env.OPENVPN_USERNAME || ""
const OPENVPN_PASSWORD = process.env.OPENVPN_PASSWORD || ""

let authToken: string | null = null
let tokenExpiresAt: number = 0

interface FetchOptions {
  method?: "GET" | "POST" | "PUT" | "DELETE"
  body?: Record<string, unknown>
}

// Authenticate and get token
async function authenticate(): Promise<string> {
  const now = Date.now()
  if (authToken && tokenExpiresAt > now + 30000) {
    return authToken
  }

  const response = await fetch(`${OPENVPN_URL}/api/auth/login/userpassword`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      username: OPENVPN_USERNAME,
      password: OPENVPN_PASSWORD,
    }),
  })

  if (!response.ok) {
    throw new Error(`Authentication failed: ${response.status}`)
  }

  const data = await response.json()
  authToken = data.auth_token
  tokenExpiresAt = now + (data.expires_in || 3600) * 1000 - 60000
  return authToken!
}

async function fetchOpenVPN<T>(endpoint: string, options: FetchOptions = {}): Promise<T> {
  const { method = "GET", body } = options
  const token = await authenticate()
  
  const response = await fetch(`${OPENVPN_URL}/api${endpoint}`, {
    method,
    headers: {
      "Authorization": `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: body ? JSON.stringify(body) : undefined,
  })

  if (!response.ok) {
    const error = await response.text()
    throw new Error(`OpenVPN API Error: ${response.status} - ${error}`)
  }

  // Handle empty responses
  const text = await response.text()
  if (!text) return {} as T
  return JSON.parse(text)
}

// Types based on OpenVPN Access Server API
export interface VPNUser {
  id: string
  username: string
  email?: string
  enabled: boolean
  createdAt: string
  lastLogin?: string
  status: "active" | "inactive" | "suspended"
  assignedIP?: string
  group?: string
  prop_autologin?: boolean
  prop_admin?: boolean
  prop_deny?: boolean
  prop_autogenerate?: boolean
}

export interface VPNGroup {
  id: string
  name: string
  description?: string
  userCount?: number
  prop_autologin?: boolean
  prop_deny?: boolean
  group_subnets?: string[]
  access_from?: string[]
  access_to?: string[]
}

export interface VPNConnection {
  id: string
  username: string
  realAddress: string
  virtualAddress: string
  bytesReceived: number
  bytesSent: number
  connectedSince: string
  clientId: string
}

export interface VPNConfig {
  id: string
  name: string
  username: string
  createdAt: string
  expiresAt?: string
  downloadCount: number
}

export interface VPNServerStatus {
  status: "running" | "stopped" | "error"
  uptime: number
  activeConnections: number
  totalUsers: number
  version: string
}

export interface UserProps {
  username?: string
  prop_autologin?: boolean
  prop_admin?: boolean
  prop_deny?: boolean
  prop_autogenerate?: boolean
  group?: string | null
  access_from?: string[]
  access_to?: string[]
}

export interface GroupProps {
  groupname?: string
  prop_autologin?: boolean
  prop_deny?: boolean
  group_subnets?: string[]
  access_from?: string[]
  access_to?: string[]
}

// VPN Users Management - based on /users/list, /users/create, /users/delete API
export async function getVPNUsers(): Promise<VPNUser[]> {
  const response = await fetchOpenVPN<{ profiles: UserProps[]; total: number }>("/users/list", {
    method: "POST",
    body: {},
  })
  
  return (response.profiles || []).map((profile) => ({
    id: profile.username || "",
    username: profile.username || "",
    enabled: !profile.prop_deny,
    createdAt: new Date().toISOString(),
    status: profile.prop_deny ? "suspended" : "active",
    group: profile.group || undefined,
    prop_autologin: profile.prop_autologin,
    prop_admin: profile.prop_admin,
    prop_deny: profile.prop_deny,
    prop_autogenerate: profile.prop_autogenerate,
  }))
}

export async function createVPNUser(user: Partial<VPNUser>): Promise<VPNUser> {
  const props: UserProps = {
    username: user.username,
    prop_autologin: user.prop_autologin,
    prop_admin: user.prop_admin,
  }
  
  if (user.group) {
    props.group = user.group
  }

  await fetchOpenVPN("/users/create", {
    method: "POST",
    body: props as Record<string, unknown>,
  })
  
  return {
    id: user.username || "",
    username: user.username || "",
    email: user.email,
    enabled: user.enabled ?? true,
    createdAt: new Date().toISOString(),
    status: "inactive",
    group: user.group,
    prop_autologin: user.prop_autologin,
    prop_admin: user.prop_admin,
  }
}

export async function updateVPNUser(username: string, updates: Partial<UserProps>): Promise<VPNUser> {
  await setUserProps(username, updates)
  
  return {
    id: username,
    username,
    enabled: !updates.prop_deny,
    createdAt: new Date().toISOString(),
    status: updates.prop_deny ? "suspended" : "active",
    group: updates.group || undefined,
  }
}

export async function deleteVPNUser(username: string): Promise<void> {
  await fetchOpenVPN<void>("/users/delete", {
    method: "POST",
    body: { users: [username] },
  })
}

export async function enableVPNUser(username: string): Promise<VPNUser> {
  await setUserProps(username, { prop_deny: false })
  return {
    id: username,
    username,
    enabled: true,
    createdAt: new Date().toISOString(),
    status: "active",
  }
}

export async function disableVPNUser(username: string): Promise<VPNUser> {
  await setUserProps(username, { prop_deny: true })
  return {
    id: username,
    username,
    enabled: false,
    createdAt: new Date().toISOString(),
    status: "suspended",
  }
}

export async function getUserProps(username: string): Promise<UserProps> {
  const response = await fetchOpenVPN<{ profiles: UserProps[] }>("/users/list", {
    method: "POST",
    body: { users: [username] },
  })
  
  return response.profiles?.[0] || {}
}

export async function setUserProps(username: string, props: Partial<UserProps>): Promise<void> {
  await fetchOpenVPN("/userprop/set", {
    method: "POST",
    body: [{ username, ...props }],
  })
}

export async function generateUserMFA(username: string): Promise<{ totp_secret?: string }> {
  return fetchOpenVPN<{ totp_secret?: string }>("/userprop/mfa/generate-secret", {
    method: "POST",
    body: { username, reset_secret: true },
  })
}

// VPN Groups Management - based on /groups/list, /groups/create, /groups/delete API
export async function getVPNGroups(): Promise<VPNGroup[]> {
  const response = await fetchOpenVPN<{ profiles: GroupProps[]; total: number }>("/groups/list", {
    method: "POST",
    body: {},
  })
  
  return (response.profiles || []).map((profile) => ({
    id: profile.groupname || "",
    name: profile.groupname || "",
    prop_autologin: profile.prop_autologin,
    prop_deny: profile.prop_deny,
    group_subnets: profile.group_subnets,
    access_from: profile.access_from,
    access_to: profile.access_to,
  }))
}

export async function createVPNGroup(group: Partial<VPNGroup>): Promise<VPNGroup> {
  const props: GroupProps = {
    groupname: group.name,
    prop_autologin: group.prop_autologin,
  }

  await fetchOpenVPN("/groups/create", {
    method: "POST",
    body: props as Record<string, unknown>,
  })
  
  return {
    id: group.name || "",
    name: group.name || "",
    description: group.description,
    userCount: 0,
    prop_autologin: group.prop_autologin,
  }
}

export async function updateVPNGroup(groupname: string, updates: Partial<GroupProps>): Promise<VPNGroup> {
  await setGroupProps(groupname, updates)
  
  return {
    id: groupname,
    name: groupname,
    prop_autologin: updates.prop_autologin,
    prop_deny: updates.prop_deny,
  }
}

export async function deleteVPNGroup(groupname: string): Promise<void> {
  await fetchOpenVPN<void>("/groups/delete", {
    method: "POST",
    body: { groups: [groupname] },
  })
}

export async function getGroupProps(groupname: string): Promise<GroupProps> {
  const response = await fetchOpenVPN<{ profiles: GroupProps[] }>("/groups/list", {
    method: "POST",
    body: { groups: [groupname] },
  })
  
  return response.profiles?.[0] || {}
}

export async function setGroupProps(groupname: string, props: Partial<GroupProps>): Promise<void> {
  await fetchOpenVPN("/userprop/set", {
    method: "POST",
    body: [{ groupname, ...props }],
  })
}

export async function getGroupMembers(groupname: string): Promise<VPNUser[]> {
  const response = await fetchOpenVPN<{ profiles: UserProps[] }>("/users/list", {
    method: "POST",
    body: {
      filters: {
        group: { value: groupname, operation: "equal" },
      },
    },
  })
  
  return (response.profiles || []).map((profile) => ({
    id: profile.username || "",
    username: profile.username || "",
    enabled: !profile.prop_deny,
    createdAt: new Date().toISOString(),
    status: profile.prop_deny ? "suspended" : "active",
    group: profile.group || undefined,
  }))
}

// VPN Connections
export async function getActiveConnections(): Promise<VPNConnection[]> {
  return fetchOpenVPN<VPNConnection[]>("/vpn/status")
}

export async function disconnectUser(clientId: string): Promise<void> {
  await fetchOpenVPN<void>("/vpn/disconnect", {
    method: "POST",
    body: { client_id: clientId },
  })
}

export async function getConnectionHistory(params?: {
  username?: string
  from?: string
  to?: string
  limit?: number
}): Promise<VPNConnection[]> {
  return fetchOpenVPN<VPNConnection[]>("/logs/list", {
    method: "POST",
    body: {
      filters: params?.username
        ? { username: { value: params.username, operation: "equal" } }
        : undefined,
    },
  })
}

// VPN Configs
export async function getVPNConfigs(username?: string): Promise<VPNConfig[]> {
  return fetchOpenVPN<VPNConfig[]>("/profile/list", {
    method: "POST",
    body: username ? { username } : {},
  })
}

export async function createVPNConfig(params: {
  username: string
  name: string
  expiresAt?: string
}): Promise<VPNConfig> {
  return fetchOpenVPN<VPNConfig>("/token-url", {
    method: "POST",
    body: {
      username: params.username,
      profile_type: "userlogin",
    },
  })
}

export async function downloadVPNConfig(configId: string): Promise<Blob> {
  const token = await authenticate()
  const response = await fetch(`${OPENVPN_URL}/rest/GetProfileViaToken?token=${configId}`, {
    headers: { "Authorization": `Bearer ${token}` },
  })

  if (!response.ok) {
    throw new Error(`Failed to download config: ${response.statusText}`)
  }

  return response.blob()
}

export async function deleteVPNConfig(configId: string): Promise<void> {
  await fetchOpenVPN<void>("/profile/delete", {
    method: "POST",
    body: { token_id: configId },
  })
}

export async function revokeVPNConfig(configId: string): Promise<void> {
  await deleteVPNConfig(configId)
}

// Server Status
export async function getServerStatus(): Promise<VPNServerStatus> {
  const info = await fetchOpenVPN<Record<string, unknown>>("/info/server")
  return {
    status: "running",
    uptime: 0,
    activeConnections: 0,
    totalUsers: 0,
    version: String(info.version || "unknown"),
  }
}

// Generate QR Code for mobile config
export async function getConfigQRCode(configId: string): Promise<string> {
  // OpenVPN AS doesn't have a direct QR endpoint, this would need custom implementation
  return `openvpn-import://${OPENVPN_URL}/rest/GetProfileViaToken?token=${configId}`
}
