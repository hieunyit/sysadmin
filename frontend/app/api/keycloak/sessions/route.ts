import { NextRequest, NextResponse } from "next/server"
import { mockSessions, mockRecentActivity } from "@/lib/mock-data"

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
      return NextResponse.json({ sessions: mockSessions })
    }

    const { getActiveSessions } = await import("@/lib/keycloak-admin")
    const sessions = await getActiveSessions()
    return NextResponse.json({ sessions })
  } catch (error) {
    console.error("Error fetching sessions:", error)
    return NextResponse.json({ sessions: mockSessions })
  }
}

export async function POST(request: NextRequest) {
  try {
    const body = await request.json()
    const { action, ...data } = body

    if (!isKeycloakConfigured()) {
      switch (action) {
        case "logoutUser":
        case "logoutSession":
        case "logoutAllSessions":
          return NextResponse.json({ success: true })
        case "getEvents":
          return NextResponse.json({
            events: mockRecentActivity.map((a) => ({
              time: new Date(a.timestamp).getTime(),
              type: a.type.toUpperCase(),
              userId: a.user,
              details: { description: a.description },
            })),
          })
        case "getAdminEvents":
          return NextResponse.json({
            events: mockRecentActivity
              .filter((a) => ["user_created", "role_assigned", "password_reset"].includes(a.type))
              .map((a) => ({
                time: new Date(a.timestamp).getTime(),
                operationType: a.type.toUpperCase(),
                authDetails: { userId: a.user },
              })),
          })
        default:
          return NextResponse.json({ error: "Invalid action" }, { status: 400 })
      }
    }

    const { logoutUser, logoutSession, getKeycloakAdminClient } = await import("@/lib/keycloak-admin")

    switch (action) {
      case "logoutUser": {
        await logoutUser(data.userId)
        return NextResponse.json({ success: true })
      }

      case "logoutSession": {
        await logoutSession(data.sessionId)
        return NextResponse.json({ success: true })
      }

      case "logoutAllSessions": {
        const client = await getKeycloakAdminClient()
        await client.realms.logoutAll({ realm: process.env.KEYCLOAK_REALM! })
        return NextResponse.json({ success: true })
      }

      case "getEvents": {
        const client = await getKeycloakAdminClient()
        const events = await client.realms.findEvents({
          realm: process.env.KEYCLOAK_REALM!,
          type: data.types,
          first: data.first || 0,
          max: data.max || 50,
        })
        return NextResponse.json({ events })
      }

      case "getAdminEvents": {
        const client = await getKeycloakAdminClient()
        const events = await client.realms.findAdminEvents({
          realm: process.env.KEYCLOAK_REALM!,
          first: data.first || 0,
          max: data.max || 50,
        })
        return NextResponse.json({ events })
      }

      default:
        return NextResponse.json({ error: "Invalid action" }, { status: 400 })
    }
  } catch (error) {
    console.error("Error processing session action:", error)
    return NextResponse.json({ success: true })
  }
}
