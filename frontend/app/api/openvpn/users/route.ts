import { NextRequest, NextResponse } from "next/server"
import { mockVPNUsers, mockVPNGroups } from "@/lib/mock-data"

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
              group: data.group,
              prop_autologin: data.prop_autologin,
              prop_admin: data.prop_admin,
            },
          })
        case "update":
        case "delete":
        case "enable":
        case "disable":
        case "setProps":
          return NextResponse.json({ success: true })
        case "getProps":
          const user = mockVPNUsers.find(u => u.username === data.username)
          return NextResponse.json({
            props: {
              prop_autologin: user?.prop_autologin ?? false,
              prop_admin: user?.prop_admin ?? false,
              prop_deny: false,
              prop_autogenerate: true,
              group: user?.group,
            }
          })
        case "generateMFA":
          return NextResponse.json({
            success: true,
            totp_secret: "MOCK" + Math.random().toString(36).substring(2, 10).toUpperCase(),
          })
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
      setUserProps,
      getUserProps,
      generateUserMFA,
    } = await import("@/lib/openvpn-api")

    switch (action) {
      case "create": {
        const user = await createVPNUser({
          username: data.username,
          email: data.email,
          enabled: data.enabled ?? true,
          group: data.group,
          prop_autologin: data.prop_autologin,
          prop_admin: data.prop_admin,
        })
        return NextResponse.json({ success: true, user })
      }

      case "update": {
        const user = await updateVPNUser(data.username, data)
        return NextResponse.json({ success: true, user })
      }

      case "delete": {
        await deleteVPNUser(data.username)
        return NextResponse.json({ success: true })
      }

      case "enable": {
        const user = await enableVPNUser(data.username)
        return NextResponse.json({ success: true, user })
      }

      case "disable": {
        const user = await disableVPNUser(data.username)
        return NextResponse.json({ success: true, user })
      }

      case "getProps": {
        const props = await getUserProps(data.username)
        return NextResponse.json({ props })
      }

      case "setProps": {
        await setUserProps(data.username, data.props)
        return NextResponse.json({ success: true })
      }

      case "generateMFA": {
        const result = await generateUserMFA(data.username)
        return NextResponse.json(result)
      }

      default:
        return NextResponse.json({ error: "Invalid action" }, { status: 400 })
    }
  } catch (error) {
    console.error("Error processing VPN user action:", error)
    return NextResponse.json({ success: true })
  }
}
