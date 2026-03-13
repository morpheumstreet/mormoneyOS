package tools

import (
	"context"

	"github.com/morpheumlabs/mormoneyos-go/internal/conway"
	"github.com/morpheumlabs/mormoneyos-go/internal/inference"
	"github.com/morpheumlabs/mormoneyos-go/internal/social"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
	"github.com/morpheumlabs/mormoneyos-go/internal/tunnel"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// Executor runs tools by name. Returns (result, errorMessage).
// When error is non-nil, result may still contain partial output.
type Executor interface {
	Execute(ctx context.Context, name string, args map[string]any) (result string, err error)
}

// Tool defines a single tool implementation (flat design, extensible).
// Each tool is self-contained: name, description, JSON schema for parameters, and execution.
type Tool interface {
	Name() string
	Description() string
	Parameters() string // JSON schema for parameters (OpenAI function format)
	Execute(ctx context.Context, args map[string]any) (string, error)
}

// Registry maps tool names to implementations.
type Registry struct {
	tools map[string]Tool
}

// RegistryOptions configures tool registration (Store, Conway, Name, extension points).
type RegistryOptions struct {
	Store           ToolStore
	Conway          conway.Client
	Name            string
	ParentAddress   string               // For spawn_child (wallet address)
	GenesisPrompt   string               // For spawn_child
	Config          *types.AutomatonConfig // Optional; for check_usdc_balance (chainProviders, defaultChain)
	SocialClient    SocialClient          // For message_child; nil = stub. If Channels has conway, used to build this.
	Channels        map[string]social.SocialChannel // Social channels (conway, telegram, discord); enables send_message
	ConfigTools     []types.ConfigToolDef // Tools from config (extension point)
	InstalledDB     InstalledToolDB       // DB installed_tools (extension point); can be same as Store when Store is *state.Database
	PluginPaths     []string              // Paths to .so plugin dirs (extension point)
	TunnelManager   *tunnel.TunnelManager  // When set, register expose_port, remove_port, tunnel_status
	TunnelRegistry  *tunnel.ProviderRegistry
}

// InstalledToolDB provides installed tools from DB (extension point).
type InstalledToolDB interface {
	GetInstalledTools() ([]state.InstalledTool, bool)
}

// NewRegistry creates a registry with built-in tools only (no Store/Conway).
func NewRegistry() *Registry {
	return NewRegistryWithOptions(nil)
}

// NewRegistryWithOptions creates a registry with built-in and optional Store/Conway tools.
// Unimplemented tools are registered first so real implementations override them.
func NewRegistryWithOptions(opts *RegistryOptions) *Registry {
	r := &Registry{tools: make(map[string]Tool)}
	r.RegisterMany(DefaultUnimplementedTools())

	r.Register(&ShellTool{})
	r.Register(&FileReadTool{})
	r.Register(&FileWriteTool{})
	r.Register(&GitStatusTool{})
	r.Register(&GitDiffTool{})
	r.Register(&GitLogTool{})
	r.Register(&GitCommitTool{})
	r.Register(&GitPushTool{})
	r.Register(&GitBranchTool{})
	r.Register(&GitCloneTool{})
	r.Alias("exec", "shell")

	if opts != nil {
		if opts.Store != nil {
			r.Register(&SleepTool{Store: opts.Store})
			r.Register(&ListSkillsTool{Store: opts.Store})
			r.Register(&SwitchModelTool{Store: opts.Store})
			if db, ok := opts.Store.(*state.Database); ok {
				r.Register(&ModifyHeartbeatTool{Store: db})
				r.Register(&InstallSkillTool{Store: db})
				r.Register(&CreateSkillTool{Store: db})
				r.Register(&RemoveSkillTool{Store: db})
				r.Register(&ListChildrenTool{Store: db})
				r.Register(&CheckChildStatusTool{Store: db})
				r.Register(&PruneDeadChildrenTool{Store: db})
			}
			r.Register(&SystemSynopsisTool{Store: opts.Store, AppName: opts.Name})
			r.Register(&CheckInferenceSpendingTool{Store: opts.Store})
			r.Register(&EnterLowComputeTool{Store: opts.Store})
			r.Register(&UpdateGenesisPromptTool{Store: opts.Store})
			r.Register(&ViewSoulTool{Store: opts.Store})
			r.Register(&UpdateSoulTool{Store: opts.Store})
			r.Register(&ReflectOnSoulTool{Store: opts.Store})
			r.Register(&ViewSoulHistoryTool{Store: opts.Store})
			r.Register(&RememberFactTool{Store: opts.Store})
			r.Register(&RecallFactsTool{Store: opts.Store})
			r.Register(&ForgetTool{Store: opts.Store})
			r.Register(&SetGoalTool{Store: opts.Store})
			r.Register(&CompleteGoalTool{Store: opts.Store})
			r.Register(&SaveProcedureTool{Store: opts.Store})
			r.Register(&RecallProcedureTool{Store: opts.Store})
			r.Register(&NoteAboutAgentTool{Store: opts.Store})
			r.Register(&ReviewMemoryTool{Store: opts.Store})
			r.Register(&DistressSignalTool{Conway: opts.Conway, Store: opts.Store})
		}
		if opts.Config != nil && opts.Config.WalletAddress != "" {
			r.Register(&CheckUSDCBalanceTool{Config: opts.Config})
		}
		if opts.Conway != nil {
			r.RegisterMany(NewConwayTools(opts.Conway))
			r.Register(&ListModelsTool{Conway: opts.Conway})
			if opts.Store != nil {
				r.Register(&HeartbeatPingTool{Conway: opts.Conway, Store: opts.Store})
				if db, ok := opts.Store.(FundChildStore); ok {
					r.Register(&FundChildTool{Conway: opts.Conway, Store: db})
				}
				if db, ok := opts.Store.(*state.Database); ok {
					r.Register(&SpawnChildTool{
						Conway:        opts.Conway,
						Store:         db,
						ParentAddress: opts.ParentAddress,
						GenesisPrompt: opts.GenesisPrompt,
						ParentName:    opts.Name,
						MaxChildren:   3,
					})
					r.Register(&StartChildTool{Conway: opts.Conway, Store: db})
					r.Register(&VerifyChildConstitutionTool{Conway: opts.Conway, Store: db})
				}
				if fc, ok := opts.Store.(FundChildStore); ok {
					socialClient := opts.SocialClient
					if socialClient == nil && opts.Channels != nil {
						if conwayCh := opts.Channels["conway"]; conwayCh != nil {
							socialClient = &SocialChannelAdapter{Channel: conwayCh}
						}
					}
					r.Register(&MessageChildTool{Social: socialClient, Store: fc})
				}
			}
		}
		if opts.Channels != nil && len(opts.Channels) > 0 {
			r.Register(&SendMessageTool{Channels: opts.Channels})
		}
		if opts.TunnelManager != nil {
			defaultProv := "bore"
			if opts.TunnelRegistry != nil && len(opts.TunnelRegistry.List()) > 0 {
				defaultProv = opts.TunnelRegistry.List()[0]
			}
			r.Register(&ExposePortTool{Manager: opts.TunnelManager, Registry: opts.TunnelRegistry, Default: defaultProv})
			r.Register(&RemovePortTool{Manager: opts.TunnelManager})
			r.Register(&TunnelStatusTool{Manager: opts.TunnelManager})
		}
	}
	r.Register(&EditOwnFileTool{})
	r.Register(&InstallNpmPackageTool{})
	r.Register(&ReviewUpstreamChangesTool{})
	r.Register(&PullUpstreamTool{})

	// Extension points: config, DB, plugins
	if opts != nil {
		if len(opts.ConfigTools) > 0 {
			r.RegisterMany(ConfigToolsFromDefs(opts.ConfigTools))
		}
		if opts.InstalledDB != nil {
			if installed, ok := opts.InstalledDB.GetInstalledTools(); ok && len(installed) > 0 {
				r.RegisterMany(InstalledToolsFromDB(installed))
			}
		}
		if len(opts.PluginPaths) > 0 {
			loader := &PluginLoader{Paths: opts.PluginPaths}
			loader.LoadPlugins(r)
		}
	}
	return r
}

// Alias maps a name to an existing tool.
func (r *Registry) Alias(name, target string) {
	if t, ok := r.tools[target]; ok {
		r.tools[name] = t
	}
}

// Register adds a tool.
func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

// RegisterMany adds multiple tools (for plugins, config-driven expansion).
func (r *Registry) RegisterMany(tools []Tool) {
	for _, t := range tools {
		r.Register(t)
	}
}

// Execute runs the named tool. Returns ErrUnknownTool if not found.
func (r *Registry) Execute(ctx context.Context, name string, args map[string]any) (string, error) {
	t, ok := r.tools[name]
	if !ok {
		return "", ErrUnknownTool{Name: name}
	}
	return t.Execute(ctx, args)
}

// List returns all registered tool names.
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.tools))
	for n := range r.tools {
		names = append(names, n)
	}
	return names
}

// Schemas returns OpenAI-format tool definitions for inference.
// Each registered name (including aliases) gets a schema so the model can call any registered name.
func (r *Registry) Schemas() []inference.ToolDefinition {
	defs := make([]inference.ToolDefinition, 0, len(r.tools))
	for name, t := range r.tools {
		defs = append(defs, inference.ToolDefinition{
			Type: "function",
			Function: inference.ToolSchema{
				Name:        name,
				Description: t.Description(),
				Parameters:  t.Parameters(),
			},
		})
	}
	return defs
}

// ErrUnknownTool is returned when the tool is not registered.
type ErrUnknownTool struct {
	Name string
}

func (e ErrUnknownTool) Error() string {
	return "unknown tool: " + e.Name
}

// ErrConwayNotConfigured is returned when a Conway tool is invoked without Conway client.
var ErrConwayNotConfigured = &conwayNotConfiguredErr{}

type conwayNotConfiguredErr struct{}

func (e *conwayNotConfiguredErr) Error() string {
	return "Conway not configured; check_credits and list_sandboxes require Conway API"
}

// ToolStore is the minimal store interface for tools that need DB.
// Implemented by state.Database.
type ToolStore interface {
	SetKV(key, value string) error
	GetKV(key string) (string, bool, error)
	DeleteKV(key string) error
	SetAgentState(state string) error
	GetAgentState() (string, bool, error)
	GetTurnCount() (int64, error)
	GetSkills() ([]map[string]any, bool)
	GetInferenceCostSummary() (todayCost, todayCalls, totalCost float64, ok bool)
}
