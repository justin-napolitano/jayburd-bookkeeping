import { Application } from "./Application";

// Authenticated application shells must never pin an older release's asset map.
export const dynamic = "force-dynamic";

// The server-owned route keeps cache policy separate from the client application.
export default function Page() {
  return <Application />;
}
