package orchestrator

import (
	"strings"

	"github.com/kxn/codex-remote-feishu/internal/core/control"
	"github.com/kxn/codex-remote-feishu/internal/core/eventcontract"
	"github.com/kxn/codex-remote-feishu/internal/core/state"
)

func (s *Service) syncBotCapabilitySettingsFromSurface(surface *state.SurfaceConsoleRecord) {
	if s == nil || s.root == nil || surface == nil || !s.surfaceCanWriteBotCapabilitySettings(surface) {
		return
	}
	key := state.BotCapabilitySettingsKey(surface.GatewayID)
	if key == "" {
		return
	}
	if s.root.BotCapabilitySettings == nil {
		s.root.BotCapabilitySettings = map[string]state.BotCapabilitySettingsRecord{}
	}
	contract := state.SurfaceDesiredBackendContract(surface)
	record, ok := state.NormalizeBotCapabilitySettingsRecord(state.BotCapabilitySettingsRecord{
		GatewayID:           strings.TrimSpace(surface.GatewayID),
		ProductMode:         contract.ProductMode,
		Backend:             contract.Backend,
		CodexProviderID:     contract.CodexProviderID,
		ClaudeProfileID:     contract.ClaudeProfileID,
		PromptOverride:      surface.PromptOverride,
		PlanMode:            surface.PlanMode,
		PlanModeOverrideSet: surface.PlanModeOverrideSet,
		UpdatedBy:           strings.TrimSpace(surface.ActorUserID),
		UpdatedAt:           s.now(),
	})
	if !ok {
		return
	}
	s.root.BotCapabilitySettings[key] = record
}

func (s *Service) surfaceCanWriteBotCapabilitySettings(surface *state.SurfaceConsoleRecord) bool {
	if surface == nil || strings.TrimSpace(surface.GatewayID) == "" {
		return false
	}
	if surface.Platform != "" && surface.Platform != "feishu" {
		return false
	}
	return surfaceFeishuRoomID(surface) == ""
}

func (s *Service) rejectBotCapabilityMutationInReadOnlySurface(surface *state.SurfaceConsoleRecord, action control.Action) []eventcontract.Event {
	if surface == nil || surfaceFeishuRoomID(surface) == "" || !isBotCapabilitySettingsAction(action.Kind) {
		return nil
	}
	text := "此设置请在和机器人的私聊中修改。群聊里只保留当前群会话设置。"
	if commandCardOwnsInlineResult(action) {
		return s.inlineCommandCardEvents(surface, action, control.FeishuCatalogConfigView{
			Sealed:     true,
			StatusKind: "error",
			StatusText: text,
		})
	}
	return notice(surface, "bot_capability_private_required", text)
}

func isBotCapabilitySettingsAction(kind control.ActionKind) bool {
	switch kind {
	case control.ActionModeCommand,
		control.ActionCodexProviderCommand,
		control.ActionClaudeProfileCommand,
		control.ActionModelCommand,
		control.ActionReasoningCommand,
		control.ActionAccessCommand,
		control.ActionPlanCommand:
		return true
	default:
		return false
	}
}
