package orchestrator

import (
	"strings"

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
