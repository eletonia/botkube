package helm

import (
	"context"
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"github.com/alexflint/go-arg"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/pluginx"
)

const (
	// PluginName is the name of the Helm Botkube plugin.
	PluginName       = "helm"
	helmBinaryName   = "helm"
	defaultNamespace = "default"
	description      = "Helm is the Botkube executor plugin that allows you to run the Helm CLI commands directly from any communication platform."
)

// Links source: https://github.com/helm/helm/releases/tag/v3.6.3
// Using go-getter syntax to unwrap the underlying directory structure.
// Read more on https://github.com/hashicorp/go-getter#subdirectories
var helmBinaryDownloadLinks = map[string]string{
	"darwin/amd64":  "https://get.helm.sh/helm-v3.6.3-darwin-amd64.tar.gz//darwin-amd64",
	"darwin/arm64":  "https://get.helm.sh/helm-v3.6.3-darwin-arm64.tar.gz//darwin-arm64",
	"linux/amd64":   "https://get.helm.sh/helm-v3.6.3-linux-amd64.tar.gz//linux-amd64",
	"linux/arm":     "https://get.helm.sh/helm-v3.6.3-linux-arm.tar.gz//linux-arm",
	"linux/arm64":   "https://get.helm.sh/helm-v3.6.3-linux-arm64.tar.gz//linux-arm64",
	"linux/386":     "https://get.helm.sh/helm-v3.6.3-linux-386.tar.gz//linux-386",
	"linux/ppc64le": "https://get.helm.sh/helm-v3.6.3-linux-ppc64le.tar.gz//linux-ppc64le",
	"linux/s390x":   "https://get.helm.sh/helm-v3.6.3-linux-s390x.tar.gz//linux-s390x",
	"windows/amd64": "https://get.helm.sh/helm-v3.6.3-windows-amd64.zip//windows-amd64",
}

type command interface {
	Validate() error
	Help() string
}

var _ executor.Executor = &Executor{}

// Executor provides functionality for running Helm CLI.
type Executor struct {
	pluginVersion          string
	executeCommandWithEnvs func(ctx context.Context, rawCmd string, envs map[string]string) (string, error)
}

// NewExecutor returns a new Executor instance.
func NewExecutor(ver string) *Executor {
	return &Executor{
		pluginVersion:          ver,
		executeCommandWithEnvs: pluginx.ExecuteCommandWithEnvs,
	}
}

// Metadata returns details about Helm plugin.
func (e *Executor) Metadata(context.Context) (api.MetadataOutput, error) {
	return api.MetadataOutput{
		Version:     e.pluginVersion,
		Description: description,
		JSONSchema:  jsonSchema(),
		Dependencies: map[string]api.Dependency{
			helmBinaryName: {
				URLs: helmBinaryDownloadLinks,
			},
		},
	}, nil
}

// Execute returns a given command as response.
//
// Supported commands:
// - install
// - uninstall
// - list
// - version
// - status
// - test
// - rollback
// - upgrade
// - history
// - get [all|manifest|hooks|notes]
func (e *Executor) Execute(ctx context.Context, in executor.ExecuteInput) (executor.ExecuteOutput, error) {
	cfg, err := MergeConfigs(in.Configs)
	if err != nil {
		return executor.ExecuteOutput{}, fmt.Errorf("while merging input configs: %w", err)
	}

	var wasHelpRequested bool
	var helmCmd Commands
	err = pluginx.ParseCommand(PluginName, in.Command, &helmCmd)
	switch err {
	case nil:
	case arg.ErrHelp:
		// we want to print our own help instead of delegating that to Helm CLI.
		wasHelpRequested = true
	default:
		return executor.ExecuteOutput{}, fmt.Errorf("while parsing input command: %w", err)
	}

	if helmCmd.Namespace == "" { // use 'default' namespace, instead of namespace where botkube was installed
		in.Command = fmt.Sprintf("%s -n %s", in.Command, defaultNamespace)
	}

	switch {
	case helmCmd.Install != nil:
		return e.handleHelmCommand(ctx, helmCmd.Install, cfg, wasHelpRequested, in.Command)
	case helmCmd.UninstallCommandAliases.Get() != nil:
		return e.handleHelmCommand(ctx, helmCmd.UninstallCommandAliases.Get(), cfg, wasHelpRequested, in.Command)
	case helmCmd.ListCommandAliases.Get() != nil:
		return e.handleHelmCommand(ctx, helmCmd.ListCommandAliases.Get(), cfg, wasHelpRequested, in.Command)
	case helmCmd.Version != nil:
		return e.handleHelmCommand(ctx, helmCmd.Version, cfg, wasHelpRequested, in.Command)
	case helmCmd.Status != nil:
		return e.handleHelmCommand(ctx, helmCmd.Status, cfg, wasHelpRequested, in.Command)
	case helmCmd.Test != nil:
		return e.handleHelmCommand(ctx, helmCmd.Test, cfg, wasHelpRequested, in.Command)
	case helmCmd.Rollback != nil:
		return e.handleHelmCommand(ctx, helmCmd.Rollback, cfg, wasHelpRequested, in.Command)
	case helmCmd.Upgrade != nil:
		return e.handleHelmCommand(ctx, helmCmd.Upgrade, cfg, wasHelpRequested, in.Command)
	case helmCmd.HistoryCommandAliases.Get() != nil:
		return e.handleHelmCommand(ctx, helmCmd.HistoryCommandAliases.Get(), cfg, wasHelpRequested, in.Command)
	case helmCmd.Get != nil:
		switch {
		case helmCmd.Get.All != nil:
			return e.handleHelmCommand(ctx, helmCmd.Get.All, cfg, wasHelpRequested, in.Command)
		case helmCmd.Get.Hooks != nil:
			return e.handleHelmCommand(ctx, helmCmd.Get.Hooks, cfg, wasHelpRequested, in.Command)
		case helmCmd.Get.Manifest != nil:
			return e.handleHelmCommand(ctx, helmCmd.Get.Manifest, cfg, wasHelpRequested, in.Command)
		case helmCmd.Get.Notes != nil:
			return e.handleHelmCommand(ctx, helmCmd.Get.Notes, cfg, wasHelpRequested, in.Command)
		case helmCmd.Get.Values != nil:
			return e.handleHelmCommand(ctx, helmCmd.Get.Values, cfg, wasHelpRequested, in.Command)
		default:
			return executor.ExecuteOutput{
				Data: helmCmd.Get.Help(),
			}, nil
		}
	case helmCmd.Help != nil, wasHelpRequested:
		return executor.ExecuteOutput{
			Data: helmCmd.Help.Help(),
		}, nil
	default:
		return executor.ExecuteOutput{
			Data: "Helm command not supported",
		}, nil
	}
}

// handleHelmList construct a Helm CLI command and run it.
func (e *Executor) handleHelmCommand(ctx context.Context, cmd command, cfg Config, wasHelpRequested bool, rawCmd string) (executor.ExecuteOutput, error) {
	if wasHelpRequested {
		return executor.ExecuteOutput{
			Data: cmd.Help(),
		}, nil
	}

	err := cmd.Validate()
	if err != nil {
		return executor.ExecuteOutput{}, err
	}

	envs := map[string]string{
		"HELM_DRIVER":      cfg.HelmDriver,
		"HELM_CACHE_HOME":  cfg.HelmCacheDir,
		"HELM_CONFIG_HOME": cfg.HelmConfigDir,
	}

	out, err := e.executeCommandWithEnvs(ctx, rawCmd, envs)
	if err != nil {
		return executor.ExecuteOutput{}, fmt.Errorf("%s\n%s", out, err.Error())
	}

	return executor.ExecuteOutput{
		Data: out,
	}, nil
}

func jsonSchema() api.JSONSchema {
	return api.JSONSchema{
		Value: heredoc.Docf(`{
			"$schema": "http://json-schema.org/draft-04/schema#",
			"title": "botkube/helm",
			"description": "%s",
			"type": "object",
			"properties": {
				"helmDriver": {
					"description": "Storage driver for Helm",
					"type": "string",
					"default": "secret",
					"enum": ["configmap", "secret", "memory"]
				},
				"helmCacheDir": {
					"description": "Path of the cache directory",
					"type": "string",
					"default": "/tmp/helm/.cache"
				},
				"helmConfigDir": {
					"description": "Path of the configuration directory",
					"type": "string",
					"default": "/tmp/helm/"
				}
			},
			"required": []
		}`, description),
	}
}
