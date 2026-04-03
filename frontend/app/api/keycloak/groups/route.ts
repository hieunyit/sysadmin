import { NextRequest, NextResponse } from "next/server"
import { mockGroups, mockUsers } from "@/lib/mock-data"

const isKeycloakConfigured = () => {
  return !!(
    process.env.KEYCLOAK_URL &&
    process.env.KEYCLOAK_REALM &&
    process.env.KEYCLOAK_CLIENT_ID &&
    process.env.KEYCLOAK_CLIENT_SECRET
  )
}

export async function GET() {
  try {
    if (!isKeycloakConfigured()) {
      return NextResponse.json({ groups: mockGroups })
    }

    const { getGroups } = await import("@/lib/keycloak-admin")
    const groups = await getGroups()
    return NextResponse.json({ groups })
  } catch (error) {
    console.error("Error fetching groups:", error)
    return NextResponse.json({ groups: mockGroups })
  }
}

export async function POST(request: NextRequest) {
  try {
    const body = await request.json()
    const { action, ...data } = body

    if (!isKeycloakConfigured()) {
      switch (action) {
        case "create":
          return NextResponse.json({ success: true, id: `mock-group-${Date.now()}` })
        case "update":
        case "delete":
        case "assignRole":
        case "removeRole":
          return NextResponse.json({ success: true })
        case "getMembers": {
          const group = mockGroups.find((g) => g.id === data.id)
          if (group) {
            const members = mockUsers.filter((u) => u.groups.includes(group.name))
            return NextResponse.json({ 
              members: members.map(m => ({
                id: m.id,
                username: m.username,
                email: m.email,
                firstName: m.firstName,
                lastName: m.lastName,
              }))
            })
          }
          return NextResponse.json({ members: [] })
        }
        case "getRoles":
          return NextResponse.json({ 
            roles: [
              { id: "r1", name: "admin", description: "Full administrator access" },
              { id: "r2", name: "user", description: "Basic user access" }
            ] 
          })
        default:
          return NextResponse.json({ error: "Invalid action" }, { status: 400 })
      }
    }

    const { getGroups, createGroup, deleteGroup, getKeycloakAdminClient } = await import("@/lib/keycloak-admin")

    switch (action) {
      case "create": {
        const client = await getKeycloakAdminClient()
        if (data.parentId) {
          // Create as sub-group
          const result = await client.groups.createChildGroup(
            { id: data.parentId },
            { name: data.name }
          )
          return NextResponse.json({ success: true, id: result.id })
        } else {
          const result = await createGroup({ name: data.name })
          return NextResponse.json({ success: true, id: result.id })
        }
      }

      case "update": {
        const client = await getKeycloakAdminClient()
        await client.groups.update({ id: data.id }, { name: data.name })
        return NextResponse.json({ success: true })
      }

      case "delete": {
        await deleteGroup(data.id)
        return NextResponse.json({ success: true })
      }

      case "getMembers": {
        const client = await getKeycloakAdminClient()
        const members = await client.groups.listMembers({ id: data.id })
        return NextResponse.json({ members })
      }

      case "getRoles": {
        const client = await getKeycloakAdminClient()
        const roles = await client.groups.listRealmRoleMappings({ id: data.id })
        return NextResponse.json({ roles })
      }

      case "assignRole": {
        const client = await getKeycloakAdminClient()
        await client.groups.addRealmRoleMappings({ id: data.groupId, roles: data.roles })
        return NextResponse.json({ success: true })
      }

      case "removeRole": {
        const client = await getKeycloakAdminClient()
        await client.groups.delRealmRoleMappings({ id: data.groupId, roles: data.roles })
        return NextResponse.json({ success: true })
      }

      default:
        return NextResponse.json({ error: "Invalid action" }, { status: 400 })
    }
  } catch (error) {
    console.error("Error processing group action:", error)
    return NextResponse.json({ success: true })
  }
}
