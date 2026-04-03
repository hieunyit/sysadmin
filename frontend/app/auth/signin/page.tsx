"use client"

import { signIn } from "next-auth/react"
import { useSearchParams } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Shield, Key } from "lucide-react"
import { Suspense } from "react"

function SignInContent() {
  const searchParams = useSearchParams()
  const callbackUrl = searchParams.get("callbackUrl") || "/"
  const error = searchParams.get("error")

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900 flex items-center justify-center p-4">
      <Card className="w-full max-w-md border-slate-700 bg-slate-800/50 backdrop-blur">
        <CardHeader className="text-center space-y-4">
          <div className="mx-auto w-16 h-16 bg-gradient-to-br from-emerald-500 to-teal-600 rounded-2xl flex items-center justify-center shadow-lg">
            <Shield className="w-8 h-8 text-white" />
          </div>
          <div>
            <CardTitle className="text-2xl text-white">Admin Portal</CardTitle>
            <CardDescription className="text-slate-400">
              Keycloak & OpenVPN Management
            </CardDescription>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {error && (
            <div className="p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-red-400 text-sm text-center">
              {error === "OAuthSignin" && "Error starting authentication"}
              {error === "OAuthCallback" && "Error during authentication callback"}
              {error === "OAuthAccountNotLinked" && "Account already linked to another provider"}
              {error === "SessionRequired" && "Please sign in to continue"}
              {error === "Default" && "An error occurred during sign in"}
            </div>
          )}
          
          <Button
            onClick={() => signIn("keycloak", { callbackUrl })}
            className="w-full h-12 bg-emerald-600 hover:bg-emerald-700 text-white"
          >
            <Key className="w-5 h-5 mr-2" />
            Sign in with Keycloak
          </Button>

          <p className="text-xs text-slate-500 text-center">
            Authenticate using your organization&apos;s Keycloak identity provider
          </p>
        </CardContent>
      </Card>
    </div>
  )
}

export default function SignInPage() {
  return (
    <Suspense fallback={
      <div className="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900 flex items-center justify-center">
        <div className="animate-pulse text-slate-400">Loading...</div>
      </div>
    }>
      <SignInContent />
    </Suspense>
  )
}
