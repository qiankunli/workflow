package controller

import (
	"time"

	"github.com/spf13/pflag"
)

// Option ...
type Option struct {
	ResyncPeriod time.Duration `desc:"Controller resync period second, < 1 means no auto resync"`

	HealthProbeBindAddress  string `desc:"The address the health probe binds to."`
	MetricsBindAddress      string `desc:"The address the metric endpoint binds to."`
	EnableLeaderElection    bool   `desc:"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager."`
	LeaderElectionNamespace string `desc:"Namespace in which the leader election resource will be created"`
	LeaderElectionID        string `desc:"Name of the leader election resource will be created"`
}

func NewDefaultOption() *Option {

	opt := &Option{
		ResyncPeriod:            time.Hour,
		EnableLeaderElection:    true,
		LeaderElectionNamespace: "workflow-system",
		LeaderElectionID:        "workflow-controller-leader-election",
		MetricsBindAddress:      ":8081",
		HealthProbeBindAddress:  ":9440",
	}
	return opt

}

func (o *Option) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&o.EnableLeaderElection, "EnableLeaderElection", o.EnableLeaderElection, "EnableLeaderElection")
	fs.DurationVar(&o.ResyncPeriod, "ResyncPeriod", o.ResyncPeriod, "ResyncPeriod")
	fs.StringVar(&o.LeaderElectionNamespace, "LeaderElectionNamespace", o.LeaderElectionNamespace, "LeaderElectionNamespace")
	fs.StringVar(&o.LeaderElectionID, "LeaderElectionID", o.LeaderElectionID, "LeaderElectionID")
	fs.StringVar(&o.MetricsBindAddress, "MetricsBindAddress", o.MetricsBindAddress, "MetricsBindAddress")
	fs.StringVar(&o.HealthProbeBindAddress, "HealthProbeBindAddress", o.HealthProbeBindAddress, "HealthProbeBindAddress")
}

func (o *Option) Complete() (err error) {
	return err
}
