package options

import (
	"flag"
	"os"

	"github.com/qiankunli/workflow/pkg/options/controller"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

// Config indicates configs of workflow
type Config struct {
	printDefaultConfig bool
	configPath         string

	Namespace string `json:"mamespace"`

	ThrottleQPS            int                `json:"throttleQPS"`
	IdleTimeoutMillSeconds int                `json:"idleTimeoutMillSeconds"`
	RPCTimeoutMillSeconds  int                `json:"rpcTimeoutMillSeconds"`
	ControllerConfig       *controller.Config `json:"controllerConfig"`
}

func NewDefaultConfig() *Config {
	cfg := &Config{
		printDefaultConfig:    false,
		configPath:            "./config.yaml",
		ControllerConfig:      controller.NewDefaultConfig(),
		ThrottleQPS:           100,
		RPCTimeoutMillSeconds: 5000,
	}
	return cfg
}
func (c *Config) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.configPath, "config", c.configPath, "workflow config files path [.config.yaml, /etc/workflow/config.yaml]")
	fs.BoolVar(&c.printDefaultConfig, "print-config", c.printDefaultConfig, "print config file")

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(c.configPath)
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/workflow")

	// init klog
	gofs := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(gofs)
	fs.AddGoFlagSet(gofs)
}

func (c *Config) MustComplete() {
	if c.configPath != "" {
		viper.SetConfigFile(c.configPath)
	}
	err := viper.ReadInConfig()
	if err != nil {
		klog.Warningf("failed read config file: %v", err)
	}
	err = viper.Unmarshal(&c)
	if err != nil {
		klog.Fatalf("failed unmarshal config file: %v", err)
	}

	if c.printDefaultConfig {
		raw, _ := yaml.Marshal(c)
		_, _ = os.Stdout.Write(raw)
		os.Exit(0)
	}
}
