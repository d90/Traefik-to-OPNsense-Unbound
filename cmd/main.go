package main

import (
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/d90/traefik-to-opnsense-unbound/internal/controller"
	"github.com/d90/traefik-to-opnsense-unbound/internal/opnsense"
	"github.com/d90/traefik-to-opnsense-unbound/internal/types"
)

var scheme = runtime.NewScheme()

func init() {
	if err := types.AddToScheme(scheme); err != nil {
		panic(err)
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("required env var not set: " + key)
	}
	return v
}

func main() {
	ctrl.SetLogger(zap.New())
	log := ctrl.Log.WithName("main")

	opnClient := opnsense.New(
		mustEnv("OPNSENSE_URL"),
		mustEnv("OPNSENSE_API_KEY"),
		mustEnv("OPNSENSE_API_SECRET"),
		os.Getenv("TLS_SKIP_VERIFY") == "true",
	)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
	})
	if err != nil {
		log.Error(err, "unable to create manager")
		os.Exit(1)
	}

	if err := (&controller.IngressRouteReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controller"),
		OPNsense: opnClient,
		TargetIP: mustEnv("TARGET_IP"),
	}).SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to setup controller")
		os.Exit(1)
	}

	log.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Error(err, "manager exited with error")
		os.Exit(1)
	}
}
