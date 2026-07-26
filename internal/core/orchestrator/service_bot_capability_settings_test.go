package orchestrator

import (
	"testing"
	"time"

	"github.com/kxn/codex-remote-feishu/internal/core/agentproto"
	"github.com/kxn/codex-remote-feishu/internal/core/control"
	"github.com/kxn/codex-remote-feishu/internal/core/state"
)

func TestPrivateModeCommandWritesBotCapabilitySettings(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	svc := newServiceForTest(&now)

	events := svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionModeCommand,
		SurfaceSessionID: "feishu:app-1:user:ou_user",
		GatewayID:        "app-1",
		ChatID:           "ou_user",
		ActorUserID:      "ou_user",
		Text:             "/mode claude",
	})
	if len(events) == 0 {
		t.Fatalf("expected feedback event")
	}

	record, ok := svc.root.BotCapabilitySettings[state.BotCapabilitySettingsKey("app-1")]
	if !ok {
		t.Fatalf("expected bot capability settings for app-1")
	}
	if record.ProductMode != state.ProductModeNormal || record.Backend != agentproto.BackendClaude {
		t.Fatalf("bot settings contract = %s/%s, want normal/claude", record.ProductMode, record.Backend)
	}
	if record.UpdatedBy != "ou_user" || !record.UpdatedAt.Equal(now) {
		t.Fatalf("updated metadata = %q/%v, want ou_user/%v", record.UpdatedBy, record.UpdatedAt, now)
	}
}

func TestGroupSurfaceReadsBotCapabilitySettingsForBackend(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	svc := newServiceForTest(&now)
	svc.root.BotCapabilitySettings[state.BotCapabilitySettingsKey("app-1")] = state.BotCapabilitySettingsRecord{
		GatewayID:       "app-1",
		ProductMode:     state.ProductModeNormal,
		Backend:         agentproto.BackendClaude,
		ClaudeProfileID: "devseek",
		PromptOverride:  state.ModelConfigRecord{Model: "claude-sonnet", ReasoningEffort: "max"},
	}
	svc.MaterializeSurfaceResumeWithCodexProvider(
		"feishu:app-1:chat:oc_room",
		"app-1",
		"oc_room",
		"ou_user",
		state.ProductModeNormal,
		agentproto.BackendCodex,
		"team-proxy",
		"",
		state.SurfaceVerbosityNormal,
		state.PlanModeSettingOff,
	)

	if got := svc.SurfaceBackend("feishu:app-1:chat:oc_room"); got != agentproto.BackendClaude {
		t.Fatalf("SurfaceBackend = %s, want claude", got)
	}
	if got := svc.SurfaceClaudeProfileID("feishu:app-1:chat:oc_room"); got != "devseek" {
		t.Fatalf("SurfaceClaudeProfileID = %q, want devseek", got)
	}
	surface := svc.root.Surfaces["feishu:app-1:chat:oc_room"]
	if surface.Backend != agentproto.BackendCodex || surface.CodexProviderID != "team-proxy" {
		t.Fatalf("group surface local contract mutated to %s/%q, want codex/team-proxy", surface.Backend, surface.CodexProviderID)
	}
}

func TestPrivatePlanCommandWritesBotCapabilitySettings(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	svc := newServiceForTest(&now)

	events := svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionPlanCommand,
		SurfaceSessionID: "feishu:app-1:user:ou_user",
		GatewayID:        "app-1",
		ChatID:           "ou_user",
		ActorUserID:      "ou_user",
		Text:             "/plan on",
	})
	if len(events) == 0 {
		t.Fatalf("expected feedback event")
	}

	record, ok := svc.root.BotCapabilitySettings[state.BotCapabilitySettingsKey("app-1")]
	if !ok {
		t.Fatalf("expected bot capability settings for app-1")
	}
	if record.PlanMode != state.PlanModeSettingOn || !record.PlanModeOverrideSet {
		t.Fatalf("bot plan = %s/%v, want on/true", record.PlanMode, record.PlanModeOverrideSet)
	}
	surface := svc.root.Surfaces["feishu:app-1:user:ou_user"]
	if surface.PlanMode != state.PlanModeSettingOn || !surface.PlanModeOverrideSet {
		t.Fatalf("private surface plan = %s/%v, want on/true for current UX", surface.PlanMode, surface.PlanModeOverrideSet)
	}
}

func TestPrivateAccessCommandWritesBotCapabilitySettings(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	svc := newServiceForTest(&now)
	svc.UpsertInstance(&state.InstanceRecord{
		InstanceID:    "inst-1",
		Backend:       agentproto.BackendCodex,
		WorkspaceRoot: "/data/dl/project",
		WorkspaceKey:  "/data/dl/project",
		ModelCatalog: &agentproto.ModelCatalogSnapshot{
			Entries: []agentproto.ModelCatalogEntry{{
				Model: "gpt-5.4",
				SupportedReasoningEfforts: []agentproto.ReasoningEffortOption{
					{ReasoningEffort: "high"},
					{ReasoningEffort: "low"},
				},
			}},
		},
		Threads: map[string]*state.ThreadRecord{},
	})
	svc.MaterializeSurfaceResumeWithCodexProvider(
		"feishu:app-1:user:ou_user",
		"app-1",
		"ou_user",
		"ou_user",
		state.ProductModeNormal,
		agentproto.BackendCodex,
		"default",
		"",
		state.SurfaceVerbosityNormal,
		state.PlanModeSettingOff,
	)
	surface := svc.root.Surfaces["feishu:app-1:user:ou_user"]
	surface.AttachedInstanceID = "inst-1"

	events := svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAccessCommand,
		SurfaceSessionID: "feishu:app-1:user:ou_user",
		GatewayID:        "app-1",
		ChatID:           "ou_user",
		ActorUserID:      "ou_user",
		Text:             "/access confirm",
	})
	if len(events) == 0 {
		t.Fatalf("expected feedback event")
	}

	record := svc.root.BotCapabilitySettings[state.BotCapabilitySettingsKey("app-1")]
	if record.PromptOverride.AccessMode != agentproto.AccessModeConfirm {
		t.Fatalf("bot access override = %q, want confirm", record.PromptOverride.AccessMode)
	}
	if surface.PromptOverride.AccessMode != agentproto.AccessModeConfirm {
		t.Fatalf("private surface access override = %q, want confirm", surface.PromptOverride.AccessMode)
	}
}

func TestGroupPromptSummaryUsesBotCapabilitySettings(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	svc := newServiceForTest(&now)
	svc.UpsertInstance(&state.InstanceRecord{
		InstanceID:    "inst-1",
		Backend:       agentproto.BackendCodex,
		WorkspaceRoot: "/data/dl/project",
		WorkspaceKey:  "/data/dl/project",
		Threads:       map[string]*state.ThreadRecord{},
	})
	svc.root.BotCapabilitySettings[state.BotCapabilitySettingsKey("app-1")] = state.BotCapabilitySettingsRecord{
		GatewayID:       "app-1",
		ProductMode:     state.ProductModeNormal,
		Backend:         agentproto.BackendCodex,
		CodexProviderID: "default",
		PromptOverride: state.ModelConfigRecord{
			Model:           "gpt-5.4",
			ReasoningEffort: "high",
			AccessMode:      agentproto.AccessModeConfirm,
		},
		PlanMode:            state.PlanModeSettingOn,
		PlanModeOverrideSet: true,
	}
	svc.MaterializeSurfaceResumeWithCodexProvider(
		"feishu:app-1:chat:oc_room",
		"app-1",
		"oc_room",
		"ou_user",
		state.ProductModeNormal,
		agentproto.BackendCodex,
		"team-proxy",
		"",
		state.SurfaceVerbosityNormal,
		state.PlanModeSettingOff,
	)
	surface := svc.root.Surfaces["feishu:app-1:chat:oc_room"]
	surface.AttachedInstanceID = "inst-1"

	summary := svc.resolveNextPromptSummary(svc.root.Instances["inst-1"], surface, "", "", state.ModelConfigRecord{})
	if summary.OverrideModel != "gpt-5.4" || summary.OverrideReasoningEffort != "high" {
		t.Fatalf("summary override = %q/%q, want gpt-5.4/high", summary.OverrideModel, summary.OverrideReasoningEffort)
	}
	if summary.EffectiveAccessMode != agentproto.AccessModeConfirm {
		t.Fatalf("summary access = %q, want confirm", summary.EffectiveAccessMode)
	}
	if summary.EffectivePlanMode != string(state.PlanModeSettingOn) || !summary.PlanModeOverrideSet {
		t.Fatalf("summary plan = %q/%v, want on/true", summary.EffectivePlanMode, summary.PlanModeOverrideSet)
	}
}

func TestGroupSurfaceBotCapabilitySettingsAreGatewayScoped(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	svc := newServiceForTest(&now)
	svc.root.BotCapabilitySettings[state.BotCapabilitySettingsKey("app-1")] = state.BotCapabilitySettingsRecord{
		GatewayID:       "app-1",
		ProductMode:     state.ProductModeNormal,
		Backend:         agentproto.BackendClaude,
		ClaudeProfileID: "devseek",
	}
	svc.root.BotCapabilitySettings[state.BotCapabilitySettingsKey("app-2")] = state.BotCapabilitySettingsRecord{
		GatewayID:       "app-2",
		ProductMode:     state.ProductModeNormal,
		Backend:         agentproto.BackendCodex,
		CodexProviderID: "team-proxy",
	}
	svc.MaterializeSurfaceResumeWithCodexProvider("feishu:app-1:chat:oc_room", "app-1", "oc_room", "ou_1", state.ProductModeNormal, agentproto.BackendCodex, "default", "", state.SurfaceVerbosityNormal, state.PlanModeSettingOff)
	svc.MaterializeSurfaceResumeWithCodexProvider("feishu:app-2:chat:oc_room", "app-2", "oc_room", "ou_2", state.ProductModeNormal, agentproto.BackendCodex, "default", "", state.SurfaceVerbosityNormal, state.PlanModeSettingOff)

	if got := svc.SurfaceBackend("feishu:app-1:chat:oc_room"); got != agentproto.BackendClaude {
		t.Fatalf("app-1 SurfaceBackend = %s, want claude", got)
	}
	if got := svc.SurfaceBackend("feishu:app-2:chat:oc_room"); got != agentproto.BackendCodex {
		t.Fatalf("app-2 SurfaceBackend = %s, want codex", got)
	}
	if got := svc.SurfaceCodexProviderID("feishu:app-2:chat:oc_room"); got != "team-proxy" {
		t.Fatalf("app-2 SurfaceCodexProviderID = %q, want team-proxy", got)
	}
}

func TestPrivateProviderAndProfileCommandsWriteBotCapabilitySettings(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	svc := newServiceForTest(&now)
	svc.MaterializeCodexProviders([]state.CodexProviderRecord{{ID: "team-proxy", Name: "Team Proxy"}})
	svc.MaterializeClaudeProfiles([]state.ClaudeProfileRecord{{ID: "devseek", Name: "DevSeek"}})
	svc.MaterializeSurfaceResumeWithCodexProvider("feishu:app-1:user:ou_user", "app-1", "ou_user", "ou_user", state.ProductModeNormal, agentproto.BackendCodex, "default", "", state.SurfaceVerbosityNormal, state.PlanModeSettingOff)
	surface := svc.root.Surfaces["feishu:app-1:user:ou_user"]

	svc.ApplySurfaceAction(control.Action{Kind: control.ActionCodexProviderCommand, SurfaceSessionID: surface.SurfaceSessionID, GatewayID: "app-1", ChatID: "ou_user", ActorUserID: "ou_user", Text: "/codexprovider team-proxy"})
	record := svc.root.BotCapabilitySettings[state.BotCapabilitySettingsKey("app-1")]
	if record.CodexProviderID != "team-proxy" {
		t.Fatalf("bot codex provider = %q, want team-proxy", record.CodexProviderID)
	}

	svc.ApplySurfaceAction(control.Action{Kind: control.ActionModeCommand, SurfaceSessionID: surface.SurfaceSessionID, GatewayID: "app-1", ChatID: "ou_user", ActorUserID: "ou_user", Text: "/mode claude"})
	svc.ApplySurfaceAction(control.Action{Kind: control.ActionClaudeProfileCommand, SurfaceSessionID: surface.SurfaceSessionID, GatewayID: "app-1", ChatID: "ou_user", ActorUserID: "ou_user", Text: "/claudeprofile devseek"})
	record = svc.root.BotCapabilitySettings[state.BotCapabilitySettingsKey("app-1")]
	if record.Backend != agentproto.BackendClaude || record.ClaudeProfileID != "devseek" {
		t.Fatalf("bot claude profile = %s/%q, want claude/devseek", record.Backend, record.ClaudeProfileID)
	}
}

func TestPrivateModelAndReasoningCommandsWriteBotCapabilitySettings(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	svc := newServiceForTest(&now)
	svc.UpsertInstance(&state.InstanceRecord{
		InstanceID:    "inst-1",
		Backend:       agentproto.BackendCodex,
		WorkspaceRoot: "/data/dl/project",
		WorkspaceKey:  "/data/dl/project",
		ModelCatalog: &agentproto.ModelCatalogSnapshot{
			Entries: []agentproto.ModelCatalogEntry{{
				Model: "gpt-5.4",
				SupportedReasoningEfforts: []agentproto.ReasoningEffortOption{
					{ReasoningEffort: "high"},
					{ReasoningEffort: "low"},
				},
			}},
		},
		Threads: map[string]*state.ThreadRecord{},
	})
	svc.MaterializeSurfaceResumeWithCodexProvider("feishu:app-1:user:ou_user", "app-1", "ou_user", "ou_user", state.ProductModeNormal, agentproto.BackendCodex, "default", "", state.SurfaceVerbosityNormal, state.PlanModeSettingOff)
	surface := svc.root.Surfaces["feishu:app-1:user:ou_user"]
	surface.AttachedInstanceID = "inst-1"

	svc.ApplySurfaceAction(control.Action{Kind: control.ActionModelCommand, SurfaceSessionID: surface.SurfaceSessionID, GatewayID: "app-1", ChatID: "ou_user", ActorUserID: "ou_user", Text: "/model gpt-5.4 high"})
	record := svc.root.BotCapabilitySettings[state.BotCapabilitySettingsKey("app-1")]
	if record.PromptOverride.Model != "gpt-5.4" || record.PromptOverride.ReasoningEffort != "high" {
		t.Fatalf("bot model/reasoning = %#v, want gpt-5.4/high", record.PromptOverride)
	}

	svc.ApplySurfaceAction(control.Action{Kind: control.ActionReasoningCommand, SurfaceSessionID: surface.SurfaceSessionID, GatewayID: "app-1", ChatID: "ou_user", ActorUserID: "ou_user", Text: "/reasoning low"})
	record = svc.root.BotCapabilitySettings[state.BotCapabilitySettingsKey("app-1")]
	if record.PromptOverride.ReasoningEffort != "low" {
		t.Fatalf("bot reasoning = %q, want low", record.PromptOverride.ReasoningEffort)
	}
}
