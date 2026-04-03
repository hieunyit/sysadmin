import KcAdminClient from "@keycloak/keycloak-admin-client"

let kcAdminClient: KcAdminClient | null = null
let tokenExpiresAt: number = 0

export async function getKeycloakAdminClient(): Promise<KcAdminClient> {
  const now = Date.now()
  
  // Reuse existing client if token is still valid (with 30 second buffer)
  if (kcAdminClient && tokenExpiresAt > now + 30000) {
    return kcAdminClient
  }

  kcAdminClient = new KcAdminClient({
    baseUrl: process.env.KEYCLOAK_URL,
    realmName: process.env.KEYCLOAK_REALM,
  })

  await kcAdminClient.auth({
    grantType: "client_credentials",
    clientId: process.env.KEYCLOAK_ADMIN_CLIENT_ID || process.env.KEYCLOAK_CLIENT_ID!,
    clientSecret: process.env.KEYCLOAK_ADMIN_CLIENT_SECRET || process.env.KEYCLOAK_CLIENT_SECRET!,
  })

  // Token typically expires in 60 seconds for client credentials
  tokenExpiresAt = now + 55000

  return kcAdminClient
}

// Types for Keycloak entities
export interface KeycloakUser {
  id?: string
  username?: string
  email?: string
  firstName?: string
  lastName?: string
  enabled?: boolean
  emailVerified?: boolean
  createdTimestamp?: number
  attributes?: Record<string, string[]>
  realmRoles?: string[]
  groups?: string[]
}

export interface KeycloakGroup {
  id?: string
  name?: string
  path?: string
  subGroups?: KeycloakGroup[]
  realmRoles?: string[]
}

export interface KeycloakRole {
  id?: string
  name?: string
  description?: string
  composite?: boolean
  clientRole?: boolean
  containerId?: string
}

export interface KeycloakSession {
  id?: string
  username?: string
  userId?: string
  ipAddress?: string
  start?: number
  lastAccess?: number
  clients?: Record<string, string>
}

// Helper functions for common operations
export async function getUsers(params?: {
  first?: number
  max?: number
  search?: string
  email?: string
  username?: string
}): Promise<KeycloakUser[]> {
  const client = await getKeycloakAdminClient()
  return client.users.find(params)
}

export async function getUserCount(): Promise<number> {
  const client = await getKeycloakAdminClient()
  return client.users.count()
}

export async function createUser(user: KeycloakUser): Promise<{ id: string }> {
  const client = await getKeycloakAdminClient()
  return client.users.create(user)
}

export async function updateUser(id: string, user: Partial<KeycloakUser>): Promise<void> {
  const client = await getKeycloakAdminClient()
  await client.users.update({ id }, user)
}

export async function deleteUser(id: string): Promise<void> {
  const client = await getKeycloakAdminClient()
  await client.users.del({ id })
}

export async function resetUserPassword(id: string, password: string, temporary: boolean = true): Promise<void> {
  const client = await getKeycloakAdminClient()
  await client.users.resetPassword({
    id,
    credential: {
      temporary,
      type: "password",
      value: password,
    },
  })
}

export async function getGroups(): Promise<KeycloakGroup[]> {
  const client = await getKeycloakAdminClient()
  return client.groups.find()
}

export async function createGroup(group: KeycloakGroup): Promise<{ id: string }> {
  const client = await getKeycloakAdminClient()
  return client.groups.create(group)
}

export async function deleteGroup(id: string): Promise<void> {
  const client = await getKeycloakAdminClient()
  await client.groups.del({ id })
}

export async function addUserToGroup(userId: string, groupId: string): Promise<void> {
  const client = await getKeycloakAdminClient()
  await client.users.addToGroup({ id: userId, groupId })
}

export async function removeUserFromGroup(userId: string, groupId: string): Promise<void> {
  const client = await getKeycloakAdminClient()
  await client.users.delFromGroup({ id: userId, groupId })
}

export async function getRoles(): Promise<KeycloakRole[]> {
  const client = await getKeycloakAdminClient()
  return client.roles.find()
}

export async function createRole(role: KeycloakRole): Promise<void> {
  const client = await getKeycloakAdminClient()
  await client.roles.create(role)
}

export async function deleteRole(name: string): Promise<void> {
  const client = await getKeycloakAdminClient()
  await client.roles.delByName({ name })
}

export async function getUserRoles(userId: string): Promise<KeycloakRole[]> {
  const client = await getKeycloakAdminClient()
  return client.users.listRealmRoleMappings({ id: userId })
}

export async function assignRoleToUser(userId: string, roles: KeycloakRole[]): Promise<void> {
  const client = await getKeycloakAdminClient()
  await client.users.addRealmRoleMappings({ id: userId, roles })
}

export async function removeRoleFromUser(userId: string, roles: KeycloakRole[]): Promise<void> {
  const client = await getKeycloakAdminClient()
  await client.users.delRealmRoleMappings({ id: userId, roles })
}

export async function getActiveSessions(): Promise<KeycloakSession[]> {
  const client = await getKeycloakAdminClient()
  // Get all client sessions in the realm
  const clients = await client.clients.find()
  const sessions: KeycloakSession[] = []
  
  for (const kClient of clients) {
    if (kClient.id) {
      try {
        const clientSessions = await client.clients.listSessions({ id: kClient.id })
        sessions.push(...clientSessions)
      } catch {
        // Client may not have sessions enabled
      }
    }
  }
  
  return sessions
}

export async function logoutUser(userId: string): Promise<void> {
  const client = await getKeycloakAdminClient()
  await client.users.logout({ id: userId })
}

export async function logoutSession(sessionId: string): Promise<void> {
  const client = await getKeycloakAdminClient()
  await client.realms.deleteSession({ realm: process.env.KEYCLOAK_REALM!, sessionId })
}
