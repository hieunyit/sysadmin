"""
OpenVPN AS post_auth script example for Phase 1 integration.

Policy model:
- if vpn_enabled != true -> deny
- if now > vpn_expire_at -> deny
- if vpn_access_state == revoked -> deny
- otherwise allow and rely on access controls already synchronized through API.

Expected metadata sources (in order):
1) SAML attributes emitted by Keycloak
2) user properties set on OpenVPN AS user/group/default profile

Datetime format: RFC3339 / ISO8601 UTC (e.g. 2026-03-28T12:00:00Z)
"""

from datetime import datetime, timezone


def _to_bool(v):
    if isinstance(v, bool):
        return v
    if v is None:
        return False
    return str(v).strip().lower() in ("1", "true", "yes", "on")


def _parse_time(v):
    if not v:
        return None
    s = str(v).strip()
    # normalize trailing Z for fromisoformat
    if s.endswith("Z"):
        s = s[:-1] + "+00:00"
    try:
        dt = datetime.fromisoformat(s)
    except Exception:
        return None
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=timezone.utc)
    return dt.astimezone(timezone.utc)


def _extract_policy_fields(authret, attributes):
    # Source 1: SAML attributes (preferred)
    saml = attributes.get("saml", {}) if isinstance(attributes, dict) else {}
    saml_attrs = saml.get("attributes", {}) if isinstance(saml, dict) else {}

    # Source 2: OpenVPN user properties projected by API sync
    proplist = authret.get("proplist", {}) if isinstance(authret, dict) else {}

    vpn_enabled = saml_attrs.get("vpn_enabled")
    if vpn_enabled is None:
        vpn_enabled = proplist.get("vpn_enabled")

    vpn_expire_at = saml_attrs.get("vpn_expire_at")
    if vpn_expire_at is None:
        vpn_expire_at = proplist.get("vpn_expire_at")

    vpn_access_state = saml_attrs.get("vpn_access_state")
    if vpn_access_state is None:
        vpn_access_state = proplist.get("vpn_access_state")

    return _to_bool(vpn_enabled), _parse_time(vpn_expire_at), str(vpn_access_state or "").strip().lower()


def post_auth(authcred, attributes, authret, info):
    username = authcred.get("username", "unknown")
    now = datetime.now(timezone.utc)

    enabled, expire_at, access_state = _extract_policy_fields(authret, attributes)

    # Deny conditions
    if not enabled:
        authret["status"] = False
        authret["reason"] = "VPN access denied: vpn_enabled is not true"
        print("[post_auth] deny user=%s reason=vpn_enabled_false" % username)
        return authret

    if access_state == "revoked":
        authret["status"] = False
        authret["reason"] = "VPN access denied: vpn_access_state is revoked"
        print("[post_auth] deny user=%s reason=access_revoked" % username)
        return authret

    if expire_at is not None and now > expire_at:
        authret["status"] = False
        authret["reason"] = "VPN access denied: vpn_expire_at has passed"
        print("[post_auth] deny user=%s reason=expired expire_at=%s now=%s" % (username, expire_at.isoformat(), now.isoformat()))
        return authret

    # Allow: access routes/rulesets are enforced by synced OpenVPN AS access controls
    authret["status"] = True
    print("[post_auth] allow user=%s state=%s expire_at=%s" % (username, access_state or "active", expire_at.isoformat() if expire_at else "none"))
    return authret
