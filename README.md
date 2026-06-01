# Traefik to OPNsense Unbound

A Kubernetes controller that watches Traefik `IngressRoute` resources and automatically registers their hostnames as host overrides in OPNsense Unbound DNS. When an IngressRoute is created, updated, or deleted, the controller keeps Unbound in sync — no manual DNS entries required.

## How it works

1. The controller watches all `IngressRoute` resources cluster-wide
2. It parses `Host()` rules from `spec.routes[].match` fields
3. It creates/removes host overrides in OPNsense Unbound via the REST API, pointing each hostname at a fixed IP (your Traefik load balancer VIP)
4. Ownership is tracked by embedding a tag in the override's description field, so the controller only touches records it created
5. Kubernetes finalizers ensure DNS records are cleaned up when an IngressRoute is deleted

## Requirements

- Kubernetes cluster running Traefik with `IngressRoute` CRDs (`traefik.io/v1alpha1`)
- OPNsense with Unbound DNS and API access enabled

## Configuration

All configuration is via environment variables:

| Variable | Description |
|---|---|
| `OPNSENSE_URL` | Base URL of your OPNsense instance (e.g. `https://192.168.1.1`) |
| `OPNSENSE_API_KEY` | OPNsense API key |
| `OPNSENSE_API_SECRET` | OPNsense API secret |
| `TARGET_IP` | IP address all DNS records resolve to (your Traefik LB VIP) |
| `TLS_SKIP_VERIFY` | Set to `true` to skip TLS certificate verification |

## OPNsense setup

1. Go to **System → Access → Users** and create a dedicated API user
2. Under that user's **API keys**, generate a key/secret pair
3. Grant the user the **Services: Unbound DNS** privilege

## Deployment

```bash
# Create the namespace and credentials secret
kubectl create namespace traefik-to-opnsense-unbound
kubectl -n traefik-to-opnsense-unbound create secret generic opnsense-creds \
  --from-literal=api-key=YOUR_KEY \
  --from-literal=api-secret=YOUR_SECRET

# Apply manifests (edit OPNSENSE_URL and TARGET_IP in config/deploy.yaml first)
kubectl apply -f config/deploy.yaml
```

### Flux CD / GitOps

If you manage your cluster with Flux, copy `config/deploy.yaml` into your repo split into separate files (rbac, deployment, secret) and add a `kustomization.yaml` referencing them. Encrypt the secret with SOPS before committing.

## Development

```bash
go mod tidy
go test ./...
go build -o manager ./cmd
```

To build and push your own image:

```bash
docker build -t ghcr.io/YOUR_USER/traefik-to-opnsense-unbound:latest .
docker push ghcr.io/YOUR_USER/traefik-to-opnsense-unbound:latest
```

A GitHub Actions workflow is included at `.github/workflows/build.yml` that builds and publishes to GHCR automatically on push to `main` and on version tags.
