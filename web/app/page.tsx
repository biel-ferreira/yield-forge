import { redirect } from "next/navigation";

// Root → the app. /dashboard is protected, so unauthenticated visitors are bounced to
// /login by the (app) layout's RequireAuth gate. (SPEC-200 FR-2006)
export default function RootPage() {
  redirect("/dashboard");
}
