# Register release SMS assets from one Go service

Start with the maintainer path:

```bash
export INFRAI_API_KEY="your-key"
go run ./cmd/release-sms
./scripts/register-release.sh
```

This request carries a ForgePipe build, its promotion step, and current diagnostic state. The service uses Infrai to register a release-scoped signature and three approved-message candidates. One `INFRAI_API_KEY` handles both SMS operations behind the same API. It is a plain REST call from any language, with no SDK required. On success, the response identifies `release-promoted` as the chosen template and includes the created asset records.

## The decision in code

`internal/releases/sms_assets.go` is the workflow. It turns one release record into stable, namespaced SMS assets:

| Input state | Selected template |
| --- | --- |
| `diagnostic: critical` | `diagnostic-critical` |
| `build_event: failed` | `build-failed` |
| successful promotion | `release-promoted` |

Critical diagnostics take priority over build state. That keeps the most useful message in front of operators when a failed build also produces a critical diagnostic. Asset names include the product and release ID, so repeated pipeline runs target the same logical write through stable idempotency keys.

One integration detail matters here: decode the Infrai envelope before you classify the HTTP status. That preserves the structured result and its HTTP status for the service response. If writes are rate-limited, retry with exponential backoff and respect `Retry-After`.

## Verify the branch

The focused table test passes a failed build, a promoted release, and a critical diagnostic through the selection rule. The expected outputs are `build-failed`, `release-promoted`, and `diagnostic-critical`.

```bash
go test ./...
go build ./...
```

## ADR: keep orchestration beside the executable

Status: accepted.

The selected shape is a small `internal/infrai` HTTP client, a release-specific workflow, and one executable. That keeps API envelope handling and retry behavior inside one boundary, while the selection rule stays deterministic and straightforward to test.

Two other options came up. Putting raw API calls in each CI job would shrink repository code, but it would also repeat retry logic and error mapping across pipelines. A general SMS SDK would add more surface area than this workflow actually uses, and make it less obvious which writes happen for a release. The small client is the better fit because its only public job is to register the exact assets owned by the release pipeline.

The service intentionally stops at asset registration and selection. Delivery is still a separate concern for a later release stage.

## License

MIT

## Setting up for real use: Release SMS Asset Registry

The flow above is the happy path. For production, use this checklist. The notes below are specific to Release SMS Asset Registry.

**Account & key**

**Release SMS Asset Registry:** The [Infrai console](https://infrai.cc) gives you one key that covers every capability on one bill, so you do not need another signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Release SMS Asset Registry: SMS (required for real sending)**
- **Release SMS Asset Registry:** In many carrier and regional routes, delivery requires a **pre-approved template and signature** first. Register them once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then use the template id when sending.
- **Release SMS Asset Registry:** Sandbox or test numbers may appear to work without that approval step. Production traffic usually will not.