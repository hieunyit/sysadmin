import { NextRequest, NextResponse } from "next/server"
import { mockVPNGroups, mockVPNUsers } from "@/lib/mock-data"

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
      // Add user counts to mock groups
      const groupsWithCounts = mockVPNGroups.map(group => ({
        ...group,
        userCount: mockVPNUsers.filter(u => u.group === group.name).length,
      }))
      return NextResponse.json({ groups: groupsWithCounts })
    }

    const { getVPNGroups } = await import("@/lib/openvpn-api")
    const groups = await getVPNGroups()
    return NextResponse.json({ groups })
  } catch (error) {
    console.error("Error fetching VPN groups:", error)
    return NextResponse.json({ groups: mockVPNGroups })
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
            group: {
              id: `mock-vpn-group-${Date.now()}`,
              name: data.name,
              description: data.description,
              userCount: 0,
            },
          })
        case "update":
        case "delete":
        case "setProps":
          return NextResponse.json({ success: true })
        case "getProps":
          const group = mockVPNGroups.find(g => g.name === data.groupname)
          return NextResponse.json({
            props: {
              prop_autologin: group?.prop_autologin ?? false,
              prop_deny: false,
              access_from: [],
              access_to: [],
            }
          })
        case "getMembers":
          const members = mockVPNUsers.filter(u => u.group === data.groupname)
          return NextResponse.json({ members })
        default:
          return NextResponse.json({ error: "Invalid action" }, { status: 400 })
      }
    }

    const {
      createVPNGroup,
      updateVPNGroup,
      deleteVPNGroup,
      setGroupProps,
      getGroupProps,
      getGroupMembers,
    } = await import("@/lib/openvpn-api")

    switch (action) {
      case "create": {
        const group = await createVPNGroup({
          name: data.name,
          description: data.description,
        })
        return NextResponse.json({ success: true, group })
      }

      case "update": {
        const group = await updateVPNGroup(data.groupname, data)
        return NextResponse.json({ success: true, group })
      }

      case "delete": {
        await deleteVPNGroup(data.groupname)
        return NextResponse.json({ success: true })
      }

      case "getProps": {
        const props = await getGroupProps(data.groupname)
        return NextResponse.json({ props })
      }

      case "setProps": {
        await setGroupProps(data.groupname, data.props)
        return NextResponse.json({ success: true })
      }

      case "getMembers": {
        const members = await getGroupMembers(data.groupname)
        return NextResponse.json({ members })
      }

      default:
        return NextResponse.json({ error: "Invalid action" }, { status: 400 })
    }
  } catch (error) {
    console.error("Error processing VPN group action:", error)
    return NextResponse.json({ success: true })
  }
}
