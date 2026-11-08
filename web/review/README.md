# Frontend review

This isolated entry uses the real FinanceNavigation, CollectionReview and
CollectionOnboarding components with fictional, in-memory callbacks. It imports
no Firebase configuration, real API client transport or provider credentials.
No Next route imports this entry. Reloading discards sample changes.

From web/, run `npm run review:build` to export
`.local/frontend-review/jayburd-frontend-review.html` at the repository root.
It is one self-contained HTML file that can be opened manually in a browser.
The ordinary multi-file build is beside it. `npm run review:dev` serves the
review on127.0.0.1:39093 where local socket permissions allow it.

The review covers Transactions, filtering, selection/labels, focused detail and
explicit workspace setup/adoption. Accounts is a labeled fictional presentation;
other navigation destinations explain that they retain the signed-in app's
existing authorized workflows. The review is not a real account management,
financial correctness, authorization or provider integration qualification.

`tests/frontend-flow.spec.tsx` mounts the same entry in the existing synthetic
component browser suite, checks five widths and explicit setup transitions,
and saves screenshots. Its fixtures do not establish server acceptance; the
existing composed Next/Go/restricted SQL journeys remain required separately.
