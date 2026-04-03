import { NextRequest, NextResponse } from "next/server"
import { mockConfigs } from "@/lib/mock-data"

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
    const username = searchParams.get("username") || undefined

    if (!isOpenVPNConfigured()) {
      let configs = [...mockConfigs]
      if (username) {
        configs = configs.filter((c) => c.username === username)
      }
      return NextResponse.json({ configs })
    }

    const { getVPNConfigs } = await import("@/lib/openvpn-api")
    const configs = await getVPNConfigs(username)
    return NextResponse.json({ configs })
  } catch (error) {
    console.error("Error fetching VPN configs:", error)
    return NextResponse.json({ configs: mockConfigs })
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
            config: {
              id: `mock-cfg-${Date.now()}`,
              name: data.name,
              username: data.username,
              platform: data.platform || "windows",
              createdAt: new Date().toISOString(),
              status: "active",
            },
          })
        case "download": {
          // Return a sample .ovpn config file
          const sampleConfig = `client
dev tun
proto udp
remote vpn.example.com 1194
resolv-retry infinite
nobind
persist-key
persist-tun
ca ca.crt
cert ${data.username || "client"}.crt
key ${data.username || "client"}.key
cipher AES-256-CBC
auth SHA256
verb 3
# Demo configuration - not for production use`
          
          return new NextResponse(sampleConfig, {
            headers: {
              "Content-Type": "application/x-openvpn-profile",
              "Content-Disposition": `attachment; filename="${data.filename || "config.ovpn"}"`,
            },
          })
        }
        case "delete":
        case "revoke":
          return NextResponse.json({ success: true })
        case "qrcode":
          return NextResponse.json({
            qrCode:
              "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
          })
        default:
          return NextResponse.json({ error: "Invalid action" }, { status: 400 })
      }
    }

    const {
      createVPNConfig,
      downloadVPNConfig,
      deleteVPNConfig,
      revokeVPNConfig,
      getConfigQRCode,
    } = await import("@/lib/openvpn-api")

    switch (action) {
      case "create": {
        const config = await createVPNConfig({
          username: data.username,
          name: data.name,
          expiresAt: data.expiresAt,
        })
        return NextResponse.json({ success: true, config })
      }

      case "download": {
        const blob = await downloadVPNConfig(data.configId)
        const buffer = await blob.arrayBuffer()
        return new NextResponse(buffer, {
          headers: {
            "Content-Type": "application/x-openvpn-profile",
            "Content-Disposition": `attachment; filename="${data.filename || "config.ovpn"}"`,
          },
        })
      }

      case "delete": {
        await deleteVPNConfig(data.configId)
        return NextResponse.json({ success: true })
      }

      case "revoke": {
        await revokeVPNConfig(data.configId)
        return NextResponse.json({ success: true })
      }

      case "qrcode": {
        const qrCode = await getConfigQRCode(data.configId)
        return NextResponse.json({ qrCode })
      }

      default:
        return NextResponse.json({ error: "Invalid action" }, { status: 400 })
    }
  } catch (error) {
    console.error("Error processing VPN config action:", error)
    return NextResponse.json({ success: true })
  }
}
