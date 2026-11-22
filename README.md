# magnetar — c12s node registry and desired-state store

The control plane's source of truth about nodes. It owns the node records
(labels, capabilities, resource capacities, OS status), the org ownership
model (pool → org claim), and the per-node desired state that nodes pull:
the OS image and the VM fleet. All storage is etcd; all node-facing traffic
is NATS.

## Role in c12s

| Pipeline | What Magnetar does |
|---|---|
| Enrollment | node registers over NATS → gets a UUID + a record in the pool |
| Configuration | placement queries by label (`QueryNodePool`, `QueryOrgOwnedNodes`) |
| OS updates | records the desired image per node (written by gravity, pulled by star) and stores every node's `os_status` (reported by star) |
| VM fleet | stores `desired_vms` / `vm_status` on a host's node record (set by the operator via the gateway, pulled/reported by the host's star) |

## What it stores (etcd)

| Key | Content |
|---|---|
| `nodes/pool/<node-id>` | unowned node record |
| `nodes/orgs/<org>/<node-id>` | owned node record |
| `labels/pool/...`, `labels/orgs/...` | label index entries for queries |
| `desired_os/<node-id>` | operator's declared OS image (separate key on purpose: status ticks rewrite the node record every ~30 s and would race with operator writes) |
| `last_seen/<node-id>` | time of the node's last heartbeat (liveness) |

The node record carries identity, labels, `capabilities`
(`config|gossip|os_update|run_vms`), `os_status`
(`healthy|switching|degraded|rolled_back|unmanaged` + booted/attempted
digests + bootc's rollback deployment, which `cockpit image rollback`
declares), `desired_vms` and `vm_status`. Reads add liveness
(`online` ≤ `NODE_STALE_AFTER`, `stale`, `offline` > `NODE_OFFLINE_AFTER`)
from `last_seen`.

## gRPC API (`pkg/api/proto/magnetar.proto`, reflection enabled)

| Group | RPCs | Authorization |
|---|---|---|
| Node reads | `GetFromNodePool`, `GetFromOrg`, `ListNodePool`, `ListOrgOwnedNodes`, `ListAllNodes` | pool reads unauthenticated; org reads require `node.get\|org\|<org>` |
| Queries | `QueryNodePool`, `QueryOrgOwnedNodes` | `node.get` on the org |
| Ownership | `ClaimOwnership` | `node.put` on the org |
| Labels | `PutBoolLabel`, `PutFloat64Label`, `PutStringLabel`, `DeleteLabel` | `node.label.put/delete` on the node |
| VMs | `SetDesiredVms`, `GetDesiredVms`, `SetVmStatuses`, `GetVmStatuses` | `node.put` / `node.get` on the **host** node |
| OS intent | `SetDesiredOsImage` (write) | `os.update` on the node, or on the org that owns it (`os.update\|org\|<org>`) |
| Removal | `DeleteNode` | `node.delete` on the node |

Caller identity travels as `authz-token` metadata; the interceptor copies it
into the context and `AuthZService` checks the JWT's `permissions` claim
(`permission|kind|id` entries).

## NATS surface

| Subject | Kind | Purpose |
|---|---|---|
| `magnetar.registration` | queue `magnetar`, req/reply | enroll a node (returns UUID) |
| `magnetar.node.heartbeat` | req/reply | liveness (writes `last_seen`); reply `found=false` makes star re-register with the same id |
| `magnetar.node.update` | req/reply | capabilities + OS status sync; reply carries `notFound` (field 2) so star re-registers only on an explicit answer |
| `magnetar.os.desired.get` | req/reply | a node pulls its declared OS image (`notFound` when unknown) |
| `magnetar.vms.desired.get` | req/reply | a host pulls its desired VM set |
| `magnetar.vms.status.set` | req/reply | a host reports observed VM status |

The `pkg/messaging/nats` helpers implement two reliability rules used across
the system: status-only messages (no-responders 503) are dropped, never
decoded as an empty reply; and `pkg/api.NatsOptions()` makes every client
retry the first connect and reconnect forever.

## Claim ownership (`internal/services/nodes.go`)

`ClaimOwnership` moves selected pool nodes into an org and wires up the rest
of the control plane: Oort (authorization inheritance), Meridian (default
namespace + quota), Gravity (`JoinCluster`, which makes star join the Serf
cluster). It is **not transactional**: per-node failures are logged and the
loop continues, so a claim can end partially applied.

## Configuration

| Variable | Purpose |
|---|---|
| `NATS_ADDRESS` | NATS server (host:port) |
| `ETCD_ADDRESS` | etcd address (plaintext) |
| `MAGNETAR_ADDRESS` | gRPC listen address |
| `OORT_ADDRESS` | policy/authorization service |
| `MERIDIAN_ADDRESS` | namespace/quota service |
| `GRAVITY_ADDRESS` | agent queue (cluster join) |
| `SECRET_KEY` | JWT key used to validate `authz-token` permissions |
| `NODE_STALE_AFTER` / `NODE_OFFLINE_AFTER` | liveness thresholds (default `90s` / `5m`) |
| `NODE_GC_ENABLED` / `NODE_GC_INTERVAL` / `NODE_GC_POOL_AFTER` | stale-record GC (default on / `10m` / `24h`): deletes only **pool** records without a heartbeat for `POOL_AFTER`; org nodes go only through `DeleteNode` |

## Build, test, run

```bash
go build ./...
go test ./...        # os service, messaging (no-responders), NATS request, publisher
go run ./cmd         # requires NATS + etcd; claim also needs oort/meridian/gravity
```

In compose (`tools/docker-compose.yml`) it runs as the `magnetar` container
with the variables above from `tools/.env`. Ports are defined per
environment (`MAGNETAR_HOSTNAME`/`MAGNETAR_PORT`).
