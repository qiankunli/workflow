package options

import (
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/qiankunli/workflow/pkg/apis/workflow/v1alpha1"
)

// ServerContext ...
type ServerContext struct {
	Config *Config

	CtrlClient ctrlclient.Client
}

func NewServerContext(cfg *Config) (*ServerContext, error) {

	restConf := ctrl.GetConfigOrDie()
	ctrlClient, err := ctrlclient.New(restConf, ctrlclient.Options{
		Scheme: GetSchema(),
	})
	if err != nil {
		return nil, err
	}

	serverCtx := &ServerContext{
		Config:     cfg,
		CtrlClient: ctrlClient,
	}
	return serverCtx, nil
}

// GetSchema ...
func GetSchema() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	return scheme
}
