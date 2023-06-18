package controller

import (
	"context"
	"fmt"

	"github.com/qiankunli/workflow/pkg/apis/workflow/v1alpha1"
	"github.com/qiankunli/workflow/pkg/controller/manager"
	"github.com/qiankunli/workflow/pkg/controller/operators"
	"github.com/qiankunli/workflow/pkg/options"
	"github.com/qiankunli/workflow/pkg/version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

const component = "controller"

// Scheme ...
var Scheme = runtime.NewScheme()

func init() {
	var _ = clientgoscheme.AddToScheme(Scheme)
	var _ = apiextensions.AddToScheme(Scheme)
	var _ = v1alpha1.AddToScheme(Scheme)
}

func newControllerCmd(ctx context.Context, opt *Option, config *options.Config) *cobra.Command {
	return &cobra.Command{
		Use:   component,
		Short: "Start the controller",
		RunE: func(c *cobra.Command, args []string) error {

			c.Flags().VisitAll(func(flag *pflag.Flag) {
				klog.Infof("FLAG: --%s=%q", flag.Name, flag.Value)
			})

			err := opt.Complete()
			if err != nil {
				return err
			}

			restConf := ctrl.GetConfigOrDie()
			controllerCtx, err := manager.NewControllerContext(restConf, config)
			if err != nil {
				return fmt.Errorf("new controller context fail: %w", err)
			}

			ctrl.SetLogger(klogr.New())
			mgr, err := ctrl.NewManager(restConf, ctrl.Options{
				Scheme: Scheme,
				// Namespace为空即为监听所有namespace
				//Namespace:               controllerCtx.Config.Namespace,
				SyncPeriod:              &opt.ResyncPeriod,
				LeaderElection:          opt.EnableLeaderElection,
				MetricsBindAddress:      opt.MetricsBindAddress,
				HealthProbeBindAddress:  opt.HealthProbeBindAddress,
				LeaderElectionID:        opt.LeaderElectionID,
				LeaderElectionNamespace: opt.LeaderElectionNamespace,
			})

			if err != nil {
				return fmt.Errorf("failed to create manager: %w", err)
			}

			// setupReconcilers ...
			if err := setupReconcilers(mgr, controllerCtx); err != nil {
				klog.Errorf("unable to setup controllers: %v", err)
				return err
			}

			// Readiness and health check
			if err := mgr.AddReadyzCheck("ping", healthz.Ping); err != nil {
				klog.Fatalf("unable to create ready check, err: %v", err)
			}

			if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
				klog.Fatalf("unable to create health check, err: %v", err)
			}

			klog.Info("starting manager", "version", version.Get())
			if err := mgr.Start(ctx); err != nil {
				klog.Errorf("running manager failed: %v", err)
				return err
			}
			return nil
		},
	}
}

// NewCommand ...
func NewCommand(ctx context.Context, config *options.Config) *cobra.Command {
	opt := NewDefaultOption()
	controllerCmd := newControllerCmd(ctx, opt, config)
	//opt.AddFlags(controllerCmd.Flags())
	return controllerCmd
}

func setupReconcilers(mgr ctrl.Manager, controllerContext *manager.ControllerContext) error {
	// syncer不受这些限速限制，100
	if err := operators.RegisterStepReconciler(mgr, controllerContext, []string{"syncer"}, 100); err != nil {
		return err
	}
	// 创建实例限速40，删除实例30
	if err := operators.RegisterStepReconciler(mgr, controllerContext, []string{"instance", "shuttle", "empty", "error", "random", "retryable_error"}, 30); err != nil {
		return err
	}
	// eip相关实例限速10
	if err := operators.RegisterStepReconciler(mgr, controllerContext, []string{"eip", "associate_eip"}, 10); err != nil {
		return err
	}
	if err := operators.RegisterWorkflowReconciler(mgr, controllerContext); err != nil {
		return err
	}

	return nil
}
