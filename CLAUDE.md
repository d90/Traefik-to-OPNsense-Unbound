# Traefik to OPNsense Unbound

Kubernetes controller that watches Traefik `IngressRoute` CRDs and syncs hostnames into OPNsense Unbound DNS as host overrides. All records point to a single configurable IP (the Traefik load balancer VIP).

## Structure

```
cmd/main.go                              entrypoint, wires up controller-runtime manager
internal/types/ingressroute.go           minimal IngressRoute CRD type definitions
internal/parser/host.go                  parses Host() rules from Traefik match strings
internal/opnsense/client.go              OPNsense Unbound REST API client
internal/controller/ingressroute_controller.go  reconcile logic
config/deploy.yaml                       Kubernetes manifests
Dockerfile                               multi-stage build
```

## Required env vars

| Variable            | Description                                |
|---------------------|--------------------------------------------|
| `OPNSENSE_URL`      | Base URL of OPNsense (e.g. https://192.168.1.1) |
| `OPNSENSE_API_KEY`  | OPNsense API key                           |
| `OPNSENSE_API_SECRET` | OPNsense API secret                      |
| `TARGET_IP`         | IP all DNS records resolve to (Traefik LB VIP) |
| `TLS_SKIP_VERIFY`   | Set to `true` to skip TLS cert validation  |

## Development

```bash
# Download dependencies
go mod tidy

# Run tests
go test ./...

# Build binary
go build -o manager ./cmd

# Build container
docker build -t traefik-to-opnsense-unbound .
```

## Deploy

```bash
# Create the secret first
kubectl -n traefik-to-opnsense-unbound create secret generic opnsense-creds \
  --from-literal=api-key=YOUR_KEY \
  --from-literal=api-secret=YOUR_SECRET

# Apply manifests (edit OPNSENSE_URL and TARGET_IP in deploy.yaml first)
kubectl apply -f config/deploy.yaml
```

## How it works

1. Controller watches all `IngressRoute` resources cluster-wide
2. On add/update: parses `Host()` rules from `spec.routes[].match`, diffs against existing OPNsense overrides for this resource, adds/removes as needed, then calls `reconfigure`
3. On delete: removes all overrides tagged with this resource's namespace/name, then removes the finalizer
4. Ownership is tracked via the `description` field in OPNsense: `traefik-to-opnsense-unbound:namespace/name`
