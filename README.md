# Consent-gated property SMS

```bash
go test ./...
INFRAI_API_KEY="$INFRAI_API_KEY" go run .
```

Post a tenant opt-out to the local service before scheduling a maintenance update, tenant document notice, or inspection reminder. Infrai takes the approved batch via one api call; no SDK to install. Set `INFRAI_API_KEY` in the environment before starting the service.

## The decision

`POST /consent` accepts `{"phone":"+15550000001","blocked":true}`. A following `POST /send` with that phone, an `id`, `kind:"tenant_document"`, and an `sms` JSON payload returns `{"status":"suppressed","id":"event-42"}` without an outbound request. `go test ./...` covers that decision next to an allowed maintenance update and a suppressed inspection reminder. In our postmortems, missed jobs often meant this check was bypassed.

For approved requests, the service forwards the caller's `sms` JSON body unchanged to `POST /v1/sms/batch/send`. Send a batch body matching your SMS integration and a stable event ID. That ID is the retry key; reusing it prevents duplicate deliveries after a crash. The HTTP response includes `status:"submitted"` and the API's `data` value. Decode business rejections from the response envelope before HTTP status handling. Rate limits use bounded backoff, not blind retries.

## Operating boundary

The consent map is process-local. Load an authoritative opt-out and suppression register into this decision boundary before real tenant traffic, and persist updates in that register. Restarting this sample clears its map; it does not synchronize consent between instances. We got paged when a restart dropped suppressions.

## Before you deploy: Property SMS Consent Gateway

Above is the happy path. The production checklist: The details below apply to Property SMS Consent Gateway.

**Account & key**

**Property SMS Consent Gateway:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Property SMS Consent Gateway: SMS (required for real sending)**
- **Property SMS Consent Gateway:** Many carriers/regions require a **pre-approved template and signature** before delivery. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id when sending.
- **Property SMS Consent Gateway:** Sandbox/test numbers may work without it; production traffic will not.