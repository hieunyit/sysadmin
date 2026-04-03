// OpenVPN Access Server REST API Client

const OPENVPN_API_URL = process.env.OPENVPN_API_URL || ""
const OPENVPN_API_KEY = process.env.OPENVPN_API_KEY || ""

interface FetchOptions {
  method?: "GET" | "POST" | "PUT" | "DELETE"
  body?: Record<string, unknown>
}

async function fetchOpenVPN<T>(endpoint: string, options: FetchOptions = {}): Promise<T> {
  const { method = "GET", body } = options
  
  const response = await fetch(`${OPENVPN_API_URL}${endpoint}`, {
    method,
    headers: {
      "Authorization": `Bearer ${OPENVPN_API_KEY}`,
      "Content-Type": "application/json",
    },
    body: body ? JSON.stringify(body) : undefined,
  })

  if (!response.ok) {
    const error = await response.text()
    throw new Error(`OpenVPN API Error: ${response.status} - ${error}`)
  }

  return response.json()
}

// Types
export interface VPNUser {
  id: string
  username: string
  email?: string
  enabled: boolean
  createdAt: string
  lastLogin?: string
  status: "active" | "inactive" | "suspended"
  allowedSubnets?: string[]
  assignedIP?: string
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

// VPN Users Management
export async function getVPNUsers(): Promise<VPNUser[]> {
  return fetchOpenVPN<VPNUser[]>("/api/v1/users")
}

export async function createVPNUser(user: Omit<VPNUser, "id" | "createdAt" | "status">): Promise<VPNUser> {
  return fetchOpenVPN<VPNUser>("/api/v1/users", {
    method: "POST",
    body: user as Record<string, unknown>,
  })
}

export async function updateVPNUser(id: string, user: Partial<VPNUser>): Promise<VPNUser> {
  return fetchOpenVPN<VPNUser>(`/api/v1/users/${id}`, {
    method: "PUT",
    body: user as Record<string, unknown>,
  })
}

export async function deleteVPNUser(id: string): Promise<void> {
  await fetchOpenVPN<void>(`/api/v1/users/${id}`, { method: "DELETE" })
}

export async function enableVPNUser(id: string): Promise<VPNUser> {
  return fetchOpenVPN<VPNUser>(`/api/v1/users/${id}/enable`, { method: "POST" })
}

export async function disableVPNUser(id: string): Promise<VPNUser> {
  return fetchOpenVPN<VPNUser>(`/api/v1/users/${id}/disable`, { method: "POST" })
}

// VPN Connections
export async function getActiveConnections(): Promise<VPNConnection[]> {
  return fetchOpenVPN<VPNConnection[]>("/api/v1/connections")
}

export async function disconnectUser(clientId: string): Promise<void> {
  await fetchOpenVPN<void>(`/api/v1/connections/${clientId}/disconnect`, { method: "POST" })
}

export async function getConnectionHistory(params?: {
  username?: string
  from?: string
  to?: string
  limit?: number
}): Promise<VPNConnection[]> {
  const searchParams = new URLSearchParams()
  if (params?.username) searchParams.set("username", params.username)
  if (params?.from) searchParams.set("from", params.from)
  if (params?.to) searchParams.set("to", params.to)
  if (params?.limit) searchParams.set("limit", params.limit.toString())
  
  return fetchOpenVPN<VPNConnection[]>(`/api/v1/connections/history?${searchParams}`)
}

// VPN Configs
export async function getVPNConfigs(username?: string): Promise<VPNConfig[]> {
  const endpoint = username ? `/api/v1/configs?username=${username}` : "/api/v1/configs"
  return fetchOpenVPN<VPNConfig[]>(endpoint)
}

export async function createVPNConfig(params: {
  username: string
  name: string
  expiresAt?: string
}): Promise<VPNConfig> {
  return fetchOpenVPN<VPNConfig>("/api/v1/configs", {
    method: "POST",
    body: params,
  })
}

export async function downloadVPNConfig(configId: string): Promise<Blob> {
  const response = await fetch(`${OPENVPN_API_URL}/api/v1/configs/${configId}/download`, {
    headers: {
      "Authorization": `Bearer ${OPENVPN_API_KEY}`,
    },
  })

  if (!response.ok) {
    throw new Error(`Failed to download config: ${response.statusText}`)
  }

  return response.blob()
}

export async function deleteVPNConfig(configId: string): Promise<void> {
  await fetchOpenVPN<void>(`/api/v1/configs/${configId}`, { method: "DELETE" })
}

export async function revokeVPNConfig(configId: string): Promise<void> {
  await fetchOpenVPN<void>(`/api/v1/configs/${configId}/revoke`, { method: "POST" })
}

// Server Status
export async function getServerStatus(): Promise<VPNServerStatus> {
  return fetchOpenVPN<VPNServerStatus>("/api/v1/status")
}

// Generate QR Code for mobile config
export async function getConfigQRCode(configId: string): Promise<string> {
  const response = await fetchOpenVPN<{ qrCode: string }>(`/api/v1/configs/${configId}/qrcode`)
  return response.qrCode
}
