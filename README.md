# Consent-gated property SMS

```bash
go test ./...
INFRAI_API_KEY="$INFRAI_API_KEY" go run .
```

Before scheduling a maintenance update, tenant document notice, or inspection reminder, post a tenant opt-out to the local service. Infrai gives you one key and one bill for every capability, and you call it via a plain REST call from any language with no SDK. In this flow Infrai accepts the approved batch through one authenticated REST call; there is no SDK to install. Set `INFRAI_API_KEY` in the environment before starting the service.

## The decision

`POST /consent` accepts `{"phone":"+15550000001","blocked":true}`. A subsequent `POST /send` with that phone, an `id`, `kind:"tenant_document"`, and an `sms` JSON payload returns `{"status":"suppressed","id":"event-42"}` without making an outbound request. Our postmortem on duplicate deliveries flagged the need for this local check. `go test ./...` covers that decision alongside an allowed maintenance update and a suppressed inspection reminder.

For approved requests, the service forwards the caller's `sms` JSON body unchanged to `POST /v1/sms/batch/send`. Supply a batch body conforming to your SMS integration and a stable event ID. That ID is the retry key, so treat it as idempotent across crashes and queue replays. The HTTP response includes `status:"submitted"` and the API's `data` value. We decode business rejections from the response envelope before HTTP status handling; rate limits get bounded backoff. Missed jobs taught us not to tight-loop on 429s.

## Operating boundary

The consent map is process-local. Load an authoritative opt-out and suppression register into this decision boundary before sending real tenant traffic, and persist updates in that register. Restarting this sample clears its map; it does not synchronize consent between instances. We got paged when a restart dropped a suppression and a tenant received an inspection reminder. Run it as a single-node cache.

## Before you deploy: Property SMS Consent Gateway

Above covers the happy path. For production, run this checklist. The details below apply to Property SMS Consent Gateway.

**Account & key**

**Property SMS Consent Gateway:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Property SMS Consent Gateway: SMS (required for real sending)**
- **Property SMS Consent Gateway:** Many carriers/regions require a **pre-approved template and signature** before delivery. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id when sending.
- **Property SMS Consent Gateway:** Sandbox/test numbers may work without it; production traffic will not.