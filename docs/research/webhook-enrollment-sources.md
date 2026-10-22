# Webhook enrollment sources — 2026-10-22

Selected official mutable sections read; API version remains 2020-09-14 and
generated request model plaid-go v49.0.0. No Dashboard, credentials or provider
payloads inspected. References below establish vendor fields, not deployment
authorization or actual signed delivery.

| ID / source | Reviewed section / fact | Limit and mapping |
| --- | --- | --- |
| WE01 [Link API](https://plaid.com/docs/api/link/#linktokencreate) | `webhook` request field: destination accepted for new enrollment; ineffective in update mode | Internal optional fixed URL maps to this field; no new consent or existing-Item mutation |
| WE02 [Items API](https://plaid.com/docs/api/items/#itemwebhookupdate) | `/item/webhook/update` request, Item response, `WEBHOOK_UPDATE_ACKNOWLEDGED` | Existing access token required; acknowledgement is Item event. Authority/uncertainty design remains proposed |
| WE03 [Transactions webhooks](https://plaid.com/docs/transactions/webhooks/) | Configuring Link; integrating update notification | New Link config alone does not prove delivery; first sync activates notifications, readiness flags do not themselves publish data |

Pinned generated `LinkTokenCreateRequest.Webhook` is `*string` with `omitempty`;
nil preserves omission. This source inspection establishes wire representation,
not provider success. Compared alternatives and standards applicability are in
[the enrollment contract](../contracts/webhook-enrollment.md).
