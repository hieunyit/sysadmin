import { NextRequest, NextResponse } from "next/server"
import { mockRoles, mockUsers } from "@/lib/mock-data"

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
      return NextResponse.json({ roles: mockRoles })
    }

    const { getRoles } = await import("@/lib/keycloak-admin")
    const roles = await getRoles()
    return NextResponse.json({ roles })
  } catch (error) {
    console.error("Error fetching roles:", error)
    return NextResponse.json({ roles: mockRoles })
  }
}

export async function POST(request: NextRequest) {
  try {
    const body = await request.json()
    const { action, ...data } = body

    if (!isKeycloakConfigured()) {
      switch (action) {
        case "create":
        case "update":
        case "delete":
        case "addComposite":
        case "removeComposite":
          return NextResponse.json({ success: true })
        case "getUsers": {
          const role = mockRoles.find((r) => r.name === data.name)
          if (role) {
            const users = mockUsers.filter((u) => u.roles.includes(role.name))
            return NextResponse.json({ users })
          }
          return NextResponse.json({ users: [] })
        }
        case "getComposites":
          return NextResponse.json({ composites: [] })
        default:
          return NextResponse.json({ error: "Invalid action" }, { status: 400 })
      }
    }

    const { createRole, deleteRole, getKeycloakAdminClient } = await import("@/lib/keycloak-admin")

    switch (action) {
      case "create": {
        await createRole({
          name: data.name,
          description: data.description,
        })
        return NextResponse.json({ success: true })
      }

      case "update": {
        const client = await getKeycloakAdminClient()
        await client.roles.updateByName(
          { name: data.currentName },
          { name: data.name, description: data.description }
        )
        return NextResponse.json({ success: true })
      }

      case "delete": {
        await deleteRole(data.name)
        return NextResponse.json({ success: true })
      }

      case "getUsers": {
        const client = await getKeycloakAdminClient()
        const users = await client.roles.findUsersWithRole({ name: data.name })
        return NextResponse.json({ users })
      }

      case "getComposites": {
        const client = await getKeycloakAdminClient()
        const composites = await client.roles.getCompositeRoles({ name: data.name })
        return NextResponse.json({ composites })
      }

      case "addComposite": {
        const client = await getKeycloakAdminClient()
        await client.roles.createComposite({ roleId: data.roleId }, data.roles)
        return NextResponse.json({ success: true })
      }

      case "removeComposite": {
        const client = await getKeycloakAdminClient()
        await client.roles.delCompositeRoles({ roleId: data.roleId }, data.roles)
        return NextResponse.json({ success: true })
      }

      default:
        return NextResponse.json({ error: "Invalid action" }, { status: 400 })
    }
  } catch (error) {
    console.error("Error processing role action:", error)
    return NextResponse.json({ success: true })
  }
}
