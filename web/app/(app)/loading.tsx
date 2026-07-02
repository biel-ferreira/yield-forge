// Shared route-level loading skeleton for the authenticated area. (SPEC-200 FR-2006)
export default function Loading() {
  return (
    <div className="space-y-4">
      <div className="h-8 w-44 animate-pulse rounded-md bg-elevated" />
      <div className="h-40 animate-pulse rounded-xl bg-elevated" />
      <div className="grid gap-4 md:grid-cols-2">
        <div className="h-48 animate-pulse rounded-xl bg-elevated" />
        <div className="h-48 animate-pulse rounded-xl bg-elevated" />
      </div>
    </div>
  );
}
