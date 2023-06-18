package main

import (
	goflag "flag"
	"fmt"
	"math/rand"
	_ "net/http/pprof" // enable pprof
	"os"
	"time"

	"github.com/qiankunli/workflow/cmd/controller"
	cmdVersion "github.com/qiankunli/workflow/cmd/version"
	"github.com/qiankunli/workflow/pkg/options"
	"github.com/qiankunli/workflow/pkg/version"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

var moduleName = version.Get().Module

func main() {
	rand.Seed(time.Now().UnixNano())
	klog.SetOutput(os.Stdout)
	defer klog.Flush()

	klog.InitFlags(nil)
	rootCmd := &cobra.Command{
		Use:   moduleName,
		Short: fmt.Sprintf("%s module", moduleName),
	}

	rootCmd.PersistentFlags().AddGoFlagSet(goflag.CommandLine)

	cfg := options.NewDefaultConfig()
	fs := rootCmd.PersistentFlags()
	cfg.AddFlags(fs)

	cobra.OnInitialize(func() {
		err := fs.Parse(os.Args[1:])
		if err != nil {
			klog.Warningf("parse args failed:%v", err)
		}
		cfg.MustComplete()
	})

	ctx := ctrl.SetupSignalHandler()
	rootCmd.AddCommand(
		controller.NewCommand(ctx, cfg),
		cmdVersion.NewCommand(),
	)

	if err := rootCmd.Execute(); err != nil {
		klog.Fatal(err)
	}
}
