const BACKEND_API_BASE_URL = (process.env.BACKEND_API_BASE_URL || "").replace(/\/+$/, "")
const BACKEND_API_TOKEN = process.env.BACKEND_API_TOKEN || process.env.ADMIN_API_TOKEN || ""

export function isBackendAdminConfigured() {
  return Boolean(BACKEND_API_BASE_URL && BACKEND_API_TOKEN)
}

export async function fetchBackendAdmin(path: string, init: RequestInit = {}) {
  if (!isBackendAdminConfigured()) {
    throw new Error("Backend admin API is not configured")
  }

  const headers = new Headers(init.headers)
  headers.set("Authorization", `Bearer ${BACKEND_API_TOKEN}`)
  headers.set("X-Actor", "frontend-proxy")
  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json")
  }

  return fetch(`${BACKEND_API_BASE_URL}${path}`, {
    ...init,
    headers,
    cache: "no-store",
  })
}

export async function readBackendAdminData<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetchBackendAdmin(path, init)
  const payload = await response.json().catch(() => null)
  if (!response.ok) {
    const message =
      payload?.error?.message ||
      payload?.message ||
      `Backend admin request failed with status ${response.status}`
    throw new Error(message)
  }
  return (payload?.data ?? payload) as T
}
