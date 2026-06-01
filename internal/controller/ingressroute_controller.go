package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/d90/traefik-to-opnsense-unbound/internal/opnsense"
	"github.com/d90/traefik-to-opnsense-unbound/internal/parser"
	"github.com/d90/traefik-to-opnsense-unbound/internal/types"
)

const (
	finalizer    = "traefik-unbound/finalizer"
	oldFinalizer = "dns.talos/finalizer" // migration: remove if present
	descPrefix   = "traefik-to-opnsense-unbound:"
)

type IngressRouteReconciler struct {
	client.Client
	Log      logr.Logger
	OPNsense *opnsense.Client
	TargetIP string
}

func (r *IngressRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("ingressroute", req.NamespacedName)

	var ir types.IngressRoute
	if err := r.Get(ctx, req.NamespacedName, &ir); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	key := req.Namespace + "/" + req.Name

	if !ir.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&ir, finalizer) {
			log.Info("deleting DNS records for removed IngressRoute")
			if err := r.deleteAllForKey(key); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(&ir, finalizer)
		}
		controllerutil.RemoveFinalizer(&ir, oldFinalizer)
		if err := r.Update(ctx, &ir); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Migrate: swap old finalizer for new one in a single update.
	if controllerutil.ContainsFinalizer(&ir, oldFinalizer) {
		controllerutil.RemoveFinalizer(&ir, oldFinalizer)
		controllerutil.AddFinalizer(&ir, finalizer)
		return ctrl.Result{}, r.Update(ctx, &ir)
	}

	if !controllerutil.ContainsFinalizer(&ir, finalizer) {
		controllerutil.AddFinalizer(&ir, finalizer)
		return ctrl.Result{}, r.Update(ctx, &ir)
	}

	desired := collectHosts(ir)
	existing, err := r.listForKey(key)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("list existing overrides: %w", err)
	}

	changed := false

	for host, uuid := range existing {
		if _, ok := desired[host]; !ok {
			log.Info("removing DNS record", "host", host)
			if err := r.OPNsense.DeleteHostOverride(uuid); err != nil {
				return ctrl.Result{}, fmt.Errorf("delete %s: %w", host, err)
			}
			changed = true
		}
	}

	for host := range desired {
		if _, ok := existing[host]; !ok {
			hostname, domain := splitFQDN(host)
			log.Info("adding DNS record", "host", host)
			if _, err := r.OPNsense.AddHostOverride(opnsense.HostOverride{
				Enabled:     "1",
				Hostname:    hostname,
				Domain:      domain,
				RR:          "A",
				Server:      r.TargetIP,
				Description: descPrefix + key,
			}); err != nil {
				return ctrl.Result{}, fmt.Errorf("add %s: %w", host, err)
			}
			changed = true
		}
	}

	if changed {
		if err := r.OPNsense.Reconfigure(); err != nil {
			return ctrl.Result{}, fmt.Errorf("reconfigure unbound: %w", err)
		}
		log.Info("DNS sync complete", "records", len(desired))
	}

	return ctrl.Result{}, nil
}

func (r *IngressRouteReconciler) listForKey(key string) (map[string]string, error) {
	all, err := r.OPNsense.ListHostOverrides()
	if err != nil {
		return nil, err
	}
	result := make(map[string]string)
	desc := descPrefix + key
	for _, h := range all {
		if h.Description == desc {
			result[h.Hostname+"."+h.Domain] = h.UUID
		}
	}
	return result, nil
}

func (r *IngressRouteReconciler) deleteAllForKey(key string) error {
	hosts, err := r.listForKey(key)
	if err != nil {
		return err
	}
	for _, uuid := range hosts {
		if err := r.OPNsense.DeleteHostOverride(uuid); err != nil {
			return err
		}
	}
	if len(hosts) > 0 {
		return r.OPNsense.Reconfigure()
	}
	return nil
}

func collectHosts(ir types.IngressRoute) map[string]struct{} {
	hosts := make(map[string]struct{})
	for _, route := range ir.Spec.Routes {
		for _, h := range parser.ParseHosts(route.Match) {
			hosts[h] = struct{}{}
		}
	}
	return hosts
}

func splitFQDN(fqdn string) (hostname, domain string) {
	if i := strings.IndexByte(fqdn, '.'); i != -1 {
		return fqdn[:i], fqdn[i+1:]
	}
	return fqdn, ""
}

func (r *IngressRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&types.IngressRoute{}).
		Complete(r)
}
