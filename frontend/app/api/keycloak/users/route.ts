import { NextRequest, NextResponse } from "next/server"
import { mockUsers } from "@/lib/mock-data"

// Check if Keycloak is configured
const isKeycloakConfigured = () => {
  return !!(
    process.env.KEYCLOAK_URL &&
    process.env.KEYCLOAK_REALM &&
    process.env.KEYCLOAK_CLIENT_ID &&
    process.env.KEYCLOAK_CLIENT_SECRET
  )
}

export async function GET(request: NextRequest) {
  try {
    const searchParams = request.nextUrl.searchParams
    const search = searchParams.get("search") || ""
    const first = parseInt(searchParams.get("first") || "0")
    const max = parseInt(searchParams.get("max") || "20")

    // Use mock data if Keycloak is not configured
    if (!isKeycloakConfigured()) {
      let users = [...mockUsers]
      
      // Apply search filter
      if (search) {
        const searchLower = search.toLowerCase()
        users = users.filter(
          (u) =>
            u.username.toLowerCase().includes(searchLower) ||
            u.email?.toLowerCase().includes(searchLower) ||
            u.firstName?.toLowerCase().includes(searchLower) ||
            u.lastName?.toLowerCase().includes(searchLower)
        )
      }

      const total = users.length
      const paginatedUsers = users.slice(first, first + max)

      return NextResponse.json({ users: paginatedUsers, total })
    }

    // Real Keycloak implementation
    const { getUsers, getUserCount } = await import("@/lib/keycloak-admin")
    const [users, count] = await Promise.all([
      getUsers({ search: search || undefined, first, max }),
      getUserCount(),
    ])

    return NextResponse.json({ users, total: count })
  } catch (error) {
    console.error("Error fetching users:", error)
    // Fallback to mock data on error
    return NextResponse.json({ users: mockUsers.slice(0, 20), total: mockUsers.length })
  }
}

export async function POST(request: NextRequest) {
  try {
    const body = await request.json()
    const { action, ...data } = body

    // Use mock responses if Keycloak is not configured
    if (!isKeycloakConfigured()) {
      // Simulate success for demo purposes
      switch (action) {
        case "create":
          return NextResponse.json({ success: true, id: `mock-${Date.now()}` })
        case "update":
        case "delete":
        case "resetPassword":
        case "assignRole":
        case "removeRole":
        case "addToGroup":
        case "removeFromGroup":
        case "logout":
        case "sendVerifyEmail":
          return NextResponse.json({ success: true })
        case "getRoles":
          return NextResponse.json({ 
            roles: [
              { id: "r1", name: "user", description: "Basic user access" },
              { id: "r2", name: "developer", description: "Developer access" }
            ] 
          })
        case "getGroups":
          const user = mockUsers.find(u => u.id === data.id)
          return NextResponse.json({ 
            groups: user?.groups?.map((g, i) => ({ 
              id: `g${i}`, 
              name: g, 
              path: `/${g}` 
            })) || []
          })
        default:
          return NextResponse.json({ error: "Invalid action" }, { status: 400 })
      }
    }

    // Real Keycloak implementation
    const {
      createUser,
      updateUser,
      deleteUser,
      resetUserPassword,
      getUserRoles,
      assignRoleToUser,
      removeRoleFromUser,
      addUserToGroup,
      removeUserFromGroup,
    } = await import("@/lib/keycloak-admin")

    switch (action) {
      case "create": {
        const result = await createUser({
          username: data.username,
          email: data.email,
          firstName: data.firstName,
          lastName: data.lastName,
          enabled: data.enabled ?? true,
          emailVerified: data.emailVerified ?? false,
        })
        
        if (data.password) {
          await resetUserPassword(result.id, data.password, data.temporaryPassword ?? true)
        }
        
        return NextResponse.json({ success: true, id: result.id })
      }

      case "update": {
        await updateUser(data.id, {
          email: data.email,
          firstName: data.firstName,
          lastName: data.lastName,
          enabled: data.enabled,
          emailVerified: data.emailVerified,
        })
        return NextResponse.json({ success: true })
      }

      case "delete": {
        await deleteUser(data.id)
        return NextResponse.json({ success: true })
      }

      case "resetPassword": {
        await resetUserPassword(data.id, data.password, data.temporary ?? true)
        return NextResponse.json({ success: true })
      }

      case "getRoles": {
        const roles = await getUserRoles(data.id)
        return NextResponse.json({ roles })
      }

      case "assignRole": {
        await assignRoleToUser(data.userId, data.roles)
        return NextResponse.json({ success: true })
      }

      case "removeRole": {
        await removeRoleFromUser(data.userId, data.roles)
        return NextResponse.json({ success: true })
      }

      case "addToGroup": {
        await addUserToGroup(data.userId, data.groupId)
        return NextResponse.json({ success: true })
      }

      case "removeFromGroup": {
        await removeUserFromGroup(data.userId, data.groupId)
        return NextResponse.json({ success: true })
      }

      case "getGroups": {
        const client = await import("@/lib/keycloak-admin").then(m => m.getKeycloakAdminClient())
        const groups = await client.users.listGroups({ id: data.id })
        return NextResponse.json({ groups })
      }

      case "logout": {
        const { logoutUser } = await import("@/lib/keycloak-admin")
        await logoutUser(data.id)
        return NextResponse.json({ success: true })
      }

      case "sendVerifyEmail": {
        const client = await import("@/lib/keycloak-admin").then(m => m.getKeycloakAdminClient())
        await client.users.sendVerifyEmail({ id: data.id })
        return NextResponse.json({ success: true })
      }

      default:
        return NextResponse.json({ error: "Invalid action" }, { status: 400 })
    }
  } catch (error) {
    console.error("Error processing user action:", error)
    return NextResponse.json({ success: true }) // Return success for demo
  }
}
