import type { ReactNode } from "react";
import "./style.css";
export const metadata = {
  title: "Jayburd · Books",
  description: "Manual bookkeeping with explicit posting review",
};
// The application has no server-side user session or financial data cache.
export default function Layout({ children }: { children: ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
