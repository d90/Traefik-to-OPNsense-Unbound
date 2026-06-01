# talos-dns-opnsense

A Kubernetes controller that watches Traefik `IngressRoute` resources and automatically registers their hostnames as host overrides in OPNsense Unbound DNS. When an IngressRoute is created, updated, or deleted, the controller keeps Unbound in sync — no manual DNS entries required.

## How it works

1. The controller watches all `IngressRoute` resources cluster-wide
2. It parses `Host()` rules from `spec.routes[].match` fields
3. It creates/removes host overrides in OPNsense Unbound via the REST API, pointing each hostname at a fixed IP (your Traefik load balancer VIP)
4. Ownership is tracked by embedding `talos-dns-opnsense:namespace/name` in the override's description field, so the controller only touches records it created
5. Kubernetes finalizers ensure DNS records are cleaned up when an IngressRoute is deleted

## Configuration

All configuration is via environment variables:

| Variable | Description |
|---|---|
| `OPNSENSE_URL` | Base URL of your OPNsense instance (e.g. `https://10.10.10.1`) |
| `OPNSENSE_API_KEY` | OPNsense API key |
| `OPNSENSE_API_SECRET` | OPNsense API secret |
| `TARGET_IP` | IP address all DNS records resolve to (your Traefik LB VIP) |
| `TLS_SKIP_VERIFY` | Set to `true` to skip TLS certificate verification |

## OPNsense setup

1. Go to **System → Access → Users** and create a dedicated API user
2. Under that user's **API keys**, generate a key/secret pair
3. Grant the user the **Services: Unbound DNS** privilege

## Deployment

This controller is deployed into a Talos cluster via Flux CD. Manifests live in [d90-talos](https://github.com/d90/d90-talos) under `home/apps/talos-dns-opnsense/`. The OPNsense credentials are stored as a SOPS-encrypted secret (age).

To deploy to a different cluster:

```bash
# Create the namespace and credentials secret
kubectl create namespace talos-dns-opnsense
kubectl -n talos-dns-opnsense create secret generic opnsense-creds \
  --from-literal=api-key=YOUR_KEY \
  --from-literal=api-secret=YOUR_SECRET

# Apply manifests (edit OPNSENSE_URL and TARGET_IP first)
kubectl apply -f config/deploy.yaml
```

## Development

```bash
go mod tidy
go test ./...
go build -o manager ./cmd
```

Container images are published to `ghcr.io/d90/talos-dns-opnsense` via GitHub Actions on every push to `main`. Tagged releases (e.g. `git tag v1.0.0 && git push --tags`) produce versioned images.
