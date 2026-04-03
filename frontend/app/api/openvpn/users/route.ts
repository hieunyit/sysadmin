import { NextRequest, NextResponse } from "next/server"
import { mockVPNUsers } from "@/lib/mock-data"

const isOpenVPNConfigured = () => {
  return !!(
    process.env.OPENVPN_URL &&
    process.env.OPENVPN_USERNAME &&
    process.env.OPENVPN_PASSWORD
  )
}

export async function GET() {
  try {
    if (!isOpenVPNConfigured()) {
      return NextResponse.json({ users: mockVPNUsers })
    }

    const { getVPNUsers } = await import("@/lib/openvpn-api")
    const users = await getVPNUsers()
    return NextResponse.json({ users })
  } catch (error) {
    console.error("Error fetching VPN users:", error)
    return NextResponse.json({ users: mockVPNUsers })
  }
}

export async function POST(request: NextRequest) {
  try {
    const body = await request.json()
    const { action, ...data } = body

    if (!isOpenVPNConfigured()) {
      switch (action) {
        case "create":
          return NextResponse.json({
            success: true,
            user: {
              id: `mock-vpn-${Date.now()}`,
              username: data.username,
              email: data.email,
              enabled: data.enabled ?? true,
              createdAt: new Date().toISOString(),
              status: "inactive",
            },
          })
        case "update":
        case "delete":
        case "enable":
        case "disable":
          return NextResponse.json({ success: true })
        default:
          return NextResponse.json({ error: "Invalid action" }, { status: 400 })
      }
    }

    const {
      createVPNUser,
      updateVPNUser,
      deleteVPNUser,
      enableVPNUser,
      disableVPNUser,
    } = await import("@/lib/openvpn-api")

    switch (action) {
      case "create": {
        const user = await createVPNUser({
          username: data.username,
          email: data.email,
          enabled: data.enabled ?? true,
        })
        return NextResponse.json({ success: true, user })
      }

      case "update": {
        const user = await updateVPNUser(data.id, data)
        return NextResponse.json({ success: true, user })
      }

      case "delete": {
        await deleteVPNUser(data.id)
        return NextResponse.json({ success: true })
      }

      case "enable": {
        const user = await enableVPNUser(data.id)
        return NextResponse.json({ success: true, user })
      }

      case "disable": {
        const user = await disableVPNUser(data.id)
        return NextResponse.json({ success: true, user })
      }

      default:
        return NextResponse.json({ error: "Invalid action" }, { status: 400 })
    }
  } catch (error) {
    console.error("Error processing VPN user action:", error)
    return NextResponse.json({ success: true })
  }
}
