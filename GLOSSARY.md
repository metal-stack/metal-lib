# GLOSSARY

Domain language for metal-lib. Use these terms in code, docs, and review.

## auditing

**Entry** — one record of a single phase of an audited invocation (request or response, opened or closed). Carries identity (RequestId, Component, Timestamp), actor (User/Tenant/Project), transport facts (Type, Detail, Path, RemoteAddr, ForwardedFor, Body, StatusCode) and outcome (Phase, Error). Its JSON tags define the storage schema shared by all backends; changing them changes what is searchable.

**Phase** — lifecycle position of an Entry within one invocation: `request` → `response` (unary calls) or `opened` → `closed` (streams). The transition is owned by the Recorder.

**Auditing backend** — an adapter at the backend seam satisfying the `Auditing` interface (`Index`, `Search`, health check). Current adapters: memory (dev/test), timescaledb, splunk. Async is a cross-cutting adapter over any backend.

**Conformance suite** — the shared behavioural test set in `auditing/test` that every search-capable backend must pass. A backend runs it through `Run` with a Suite; parity gaps stay declared because cases name the capabilities they require and profiles declare what the backend has.

**Capability** — a named backend feature (`search`, `multi-word-body`) that conformance cases may require. Declared in a Profile, checked per case; missing capabilities produce visible skips instead of name-matched skip lists.

**Profile** — the set of capabilities a backend under test declares. Together with the Suite it fully describes what the run will execute.

**Suite** — one configured run of the conformance cases: how to obtain the backend under test (fresh instance or shared), its Profile, optional state cleanup between cases, and the expected health-check message.

**Recorder** — the deep module in `auditing` that owns the audit conventions behind the transport seam: request-ID generation, actor mapping (identity is always the subject), the two-phase index flow, error serialization, and status-code derivation. Transport adapters hand it raw facts (`Transport`) and outcomes (`Outcome`); they do not know audit conventions.

**Transport adapter** — the protocol-specific modules at the transport seam (`auditing/http`, `auditing/grpc`, `auditing/connectrpc`). Each extracts its protocol's facts from an incoming invocation and drives the Recorder. They own routing policy (which invocations are audited), body capture, and their failure policy when indexing fails.

**Request ID** — the correlation identifier shared by both phases of an invocation. Reused from upstream middleware when present (e.g. the rest logging filter), otherwise generated as UUIDv7 by the Recorder.
