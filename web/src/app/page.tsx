import { Application } from "./Application";
import { headers } from "next/headers";
import { redirect } from "next/navigation";

// Authenticated application shells must never pin an older release's asset map.
export const dynamic = "force-dynamic";

// Cloud Run aliases resolve to the reviewed public origin before authentication.
export default async function Page() {
  if (process.env.BOOKKEEPING_API_MODE === "cloud-run") {
    const origin = process.env.BOOKKEEPING_PUBLIC_ORIGIN;
    if (origin && (await headers()).get("host") !== new URL(origin).host)
      redirect(origin);
  }
  return <Application />;
}
