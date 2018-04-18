package options

import (
	"github.com/spf13/pflag"
	"time"
)

type EnnPolicyConfig struct {

	Kubeconfig          string
	Master              string
	HostnameOverride    string

	IPRange             string

	ConfigSyncPeriod    time.Duration
	PolicyPeriod        time.Duration
	MinSyncPeriod       time.Duration

	GlogToStderr        bool
	GlogV               string
	GlogDir             string

	CleanupConfig       bool
	Version             bool
}

func NewEnnPolicyConfig() *EnnPolicyConfig{

	return &EnnPolicyConfig{
		IPRange:            "0.0.0.0/0",
		ConfigSyncPeriod:   15 * time.Minute,
		PolicyPeriod:       15 * time.Minute,
	}
}

func (s *EnnPolicyConfig) AddFlags(fs *pflag.FlagSet){
	fs.StringVar(&s.Kubeconfig, "kubeconfig", s.Kubeconfig, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	fs.StringVar(&s.Master, "master", s.Master, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	fs.StringVar(&s.HostnameOverride,"hostname-override",s.HostnameOverride,"If non-empty, will use this string as identification instead of the actual hostname.")
	fs.StringVar(&s.IPRange,"ip-range",s.IPRange,"the ip-range will restrict the policy range, enn-policy is only effective within the ip-range (default value is 0.0.0.0/0)")
	fs.DurationVar(&s.ConfigSyncPeriod,"config-sync-period",s.ConfigSyncPeriod,"How often configuration from the apiserver is refreshed.  Must be greater than 0.")
	fs.DurationVar(&s.PolicyPeriod,"sync-period",s.PolicyPeriod,"The maximum interval of how often ipvs rules are refreshed (e.g. '5s', '1m', '2h22m').  Must be greater than 0.")
	fs.DurationVar(&s.MinSyncPeriod,"min-sync-period",s.MinSyncPeriod,"The minimum interval of how often the iptables rules can be refreshed as endpoints and services change (e.g. '5s', '1m', '2h22m').")
	fs.BoolVar(&s.GlogToStderr, "logtostderr", s.GlogToStderr, "If true will log to standard error instead of files")
	fs.StringVar(&s.GlogV, "v", s.GlogV, "Log level for V logs")
	fs.StringVar(&s.GlogDir, "log-dir", s.GlogDir, "If none empty, write log files in this directory")
	fs.BoolVar(&s.CleanupConfig,"cleanup-config",s.CleanupConfig,"If true cleanup all ipset/iptables rules and exit.")
	fs.BoolVar(&s.Version,"version",s.Version,"If true will show enn-policy version number.")
}