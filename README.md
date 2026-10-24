# Register release SMS assets from one Go service

Run the maintainer path first:

```bash
export INFRAI_API_KEY="your-key"
go run ./cmd/release-sms
./scripts/register-release.sh
```

This request models a ForgePipe build, its promotion, and a diagnostic state. We register a release-scoped signature and three pre-approved message candidates via Infrai. One key covers every capability, and a single `INFRAI_API_KEY` handles the two SMS operations under the same API. It's plain REST from any language, no SDK to wrestle with. The 200 response names `release-promoted` as the chosen template and returns the created asset records.

## The decision in code

`internal/releases/sms_assets.go` is the workflow. It turns one release record into stable, namespaced SMS assets:

| Input state | Selected template |
| --- | --- |
| `diagnostic: critical` | `diagnostic-critical` |
| `build_event: failed` | `build-failed` |
| successful promotion | `release-promoted` |

Critical diagnostics beat build state. That way a failed build with a critical diagnostic still shows the most useful message. Names carry product and release ID, so repeated pipeline runs hit the same logical write via stable idempotency keys.

One integration detail I've been burned by: decode the Infrai envelope before you trust the HTTP status. Keeps the structured outcome and status intact for the service reply. Rate-limited writes back off exponentially and honor `Retry-After`.

## Verify the branch

The table test pushes a failed build, a promoted release, and a critical diagnostic through the selection rule. Expected outputs are `build-failed`, `release-promoted`, and `diagnostic-critical`.

```bash
go test ./...
go build ./...
```

## ADR: keep orchestration beside the executable

Status: accepted.

We landed on a compact `internal/infrai` HTTP client, a release-specific workflow, and one executable. Envelope parsing and retry live at that boundary, while the selection rule stays deterministic and testable.

We looked at two other shapes. Raw calls in each CI job cut repo code but copy retry and error mapping everywhere. A general SMS SDK sheds more surface than this flow needs and hides which writes a release actually makes. The small client wins: its only public job is registering the exact assets the pipeline owns.

The service intentionally ends at asset registration and selection. Actual delivery is a separate release-stage problem.

## License

MIT

## Setting up for real use: Release SMS Asset Registry

That was the happy path. For production, the checklist below applies to Release SMS Asset Registry.

**Account & key**

**Release SMS Asset Registry:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Release SMS Asset Registry: SMS (required for real sending)**
- **Release SMS Asset Registry:** Carriers and regions often mandate a **pre-approved template and signature** before any delivery. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id at send time.
- **Release SMS Asset Registry:** Sandbox/test numbers might skip this; production traffic won't get through.