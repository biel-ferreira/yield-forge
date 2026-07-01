import type { NextConfig } from "next";

// D1 (SPEC-200 / PLAN-200): the browser calls the app's OWN origin under /api/*, and
// Next.js proxies to the Go backend. This keeps the SPEC-003 HttpOnly `SameSite=Lax`
// session cookie same-origin — no CORS, no `SameSite=None`. Server-side env only.
const API_PROXY_TARGET = process.env.API_PROXY_TARGET ?? "http://localhost:8080";

const nextConfig: NextConfig = {
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${API_PROXY_TARGET}/:path*` }];
  },
};

export default nextConfig;
