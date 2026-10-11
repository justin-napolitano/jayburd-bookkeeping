import type { NextConfig } from "next";
// The operator selects one trusted backend; browser input never controls the destination.
const api = new URL(
  process.env.BOOKKEEPING_API_ORIGIN ?? "http://127.0.0.1:8080",
);
if (
  api.username ||
  api.password ||
  api.pathname !== "/" ||
  api.search ||
  api.hash ||
  !(
    api.protocol === "https:" ||
    (api.protocol === "http:" &&
      ["127.0.0.1", "localhost"].includes(api.hostname))
  )
) {
  throw new Error(
    "BOOKKEEPING_API_ORIGIN must be a trusted HTTPS or loopback HTTP origin",
  );
}
const config: NextConfig = {
  poweredByHeader: false,
  async rewrites() {
    return [{ source: "/v1/:path*", destination: `${api.origin}/v1/:path*` }];
  },
  async headers() {
    return [
      {
        source: "/:path*",
        headers: [
          { key: "X-Content-Type-Options", value: "nosniff" },
          { key: "Referrer-Policy", value: "no-referrer" },
          { key: "X-Frame-Options", value: "DENY" },
        ],
      },
    ];
  },
};
export default config;
