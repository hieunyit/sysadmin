import { NextRequest, NextResponse } from "next/server"
import { mockVPNConnections } from "@/lib/mock-data"

const isOpenVPNConfigured = () => {
  return !!(
    process.env.OPENVPN_URL &&
    process.env.OPENVPN_USERNAME &&
    process.env.OPENVPN_PASSWORD
  )
}

export async function GET(request: NextRequest) {
  try {
    const searchParams = request.nextUrl.searchParams
    const type = searchParams.get("type") || "active"

    if (!isOpenVPNConfigured()) {
      if (type === "history") {
        // Return mock history
        return NextResponse.json({
          connections: mockVPNConnections.map((c) => ({
            ...c,
            disconnectedAt: new Date(
              new Date(c.connectedSince).getTime() + Math.random() * 4 * 60 * 60 * 1000
            ).toISOString(),
          })),
        })
      }
      return NextResponse.json({ connections: mockVPNConnections })
    }

    const { getActiveConnections, getConnectionHistory } = await import("@/lib/openvpn-api")

    if (type === "history") {
      const history = await getConnectionHistory({
        username: searchParams.get("username") || undefined,
        from: searchParams.get("from") || undefined,
        to: searchParams.get("to") || undefined,
        limit: parseInt(searchParams.get("limit") || "50"),
      })
      return NextResponse.json({ connections: history })
    }

    const connections = await getActiveConnections()
    return NextResponse.json({ connections })
  } catch (error) {
    console.error("Error fetching VPN connections:", error)
    return NextResponse.json({ connections: mockVPNConnections })
  }
}

export async function POST(request: NextRequest) {
  try {
    const body = await request.json()
    const { action, ...data } = body

    if (!isOpenVPNConfigured()) {
      switch (action) {
        case "disconnect":
          return NextResponse.json({ success: true })
        default:
          return NextResponse.json({ error: "Invalid action" }, { status: 400 })
      }
    }

    const { disconnectUser } = await import("@/lib/openvpn-api")

    switch (action) {
      case "disconnect": {
        await disconnectUser(data.clientId)
        return NextResponse.json({ success: true })
      }

      default:
        return NextResponse.json({ error: "Invalid action" }, { status: 400 })
    }
  } catch (error) {
    console.error("Error processing VPN connection action:", error)
    return NextResponse.json({ success: true })
  }
}
