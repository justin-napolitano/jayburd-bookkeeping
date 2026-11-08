import { readFileSync, writeFileSync, readdirSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";

// Produce one self-contained synthetic review file; no fetch, hosted API or credential is required.
const root = fileURLToPath(
  new URL("../.local/frontend-review", import.meta.url),
);
let html = readFileSync(resolve(root, "index.html"), "utf8");
const script = html.match(/<script[^>]+src="([^"]+)"[^>]*><\/script>/);
const style = html.match(/<link[^>]+href="([^"]+\.css)"[^>]*>/);
if (!script || !style) throw Error("Review assets are missing");
const js = readFileSync(resolve(root, script[1]), "utf8");
const css = readFileSync(resolve(root, style[1]), "utf8");
if (
  readdirSync(resolve(root, "assets")).some(
    (name) => !name.endsWith(".js") && !name.endsWith(".css"),
  )
)
  throw Error("Unexpected external review asset");
html = html
  .replace(
    script[0],
    () =>
      `<script type="module">${js.replaceAll("</script", "<\\/script")}</script>`,
  )
  .replace(style[0], () => `<style>${css}</style>`);
writeFileSync(resolve(root, "jayburd-frontend-review.html"), html);
console.log("PASS: self-contained synthetic frontend review exported");
