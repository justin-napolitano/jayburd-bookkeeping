import { createRoot } from "react-dom/client";
import { FrontendReview } from "./FrontendReview";
import { RulePackReview } from "./RulePackReview";
import "../src/app/style.css";

const root = createRoot(document.getElementById("root")!);
// Hash navigation belongs only to this isolated, memory-only design review.
const renderReview = () =>
  root.render(
    window.location.hash === "#rules" ? <RulePackReview /> : <FrontendReview />,
  );
window.addEventListener("hashchange", renderReview);
renderReview();
