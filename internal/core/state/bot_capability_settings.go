package state

import (
	"strings"

	"github.com/kxn/codex-remote-feishu/internal/core/agentproto"
)

const (
	SurfaceCapabilitySettingsSourceSurface = "surface"
	SurfaceCapabilitySettingsSourceBot     = "bot"
)

func BotCapabilitySettingsKey(gatewayID string) string {
	gatewayID = strings.TrimSpace(gatewayID)
	if gatewayID == "" {
		return ""
	}
	return "feishu:gateway:" + gatewayID
}

func NormalizeBotCapabilitySettingsRecord(record BotCapabilitySettingsRecord) (BotCapabilitySettingsRecord, bool) {
	record.GatewayID = strings.TrimSpace(record.GatewayID)
	if record.GatewayID == "" {
		return BotCapabilitySettingsRecord{}, false
	}
	contract := NormalizeSurfaceBackendContract(SurfaceBackendContract{
		ProductMode:     record.ProductMode,
		Backend:         record.Backend,
		CodexProviderID: strings.TrimSpace(record.CodexProviderID),
		ClaudeProfileID: strings.TrimSpace(record.ClaudeProfileID),
	})
	record.ProductMode = contract.ProductMode
	record.Backend = contract.Backend
	record.CodexProviderID = contract.CodexProviderID
	record.ClaudeProfileID = contract.ClaudeProfileID
	record.PromptOverride = NormalizeModelConfigRecord(record.PromptOverride)
	record.PlanMode = NormalizePlanModeSetting(record.PlanMode)
	record.UpdatedBy = strings.TrimSpace(record.UpdatedBy)
	if !record.UpdatedAt.IsZero() {
		record.UpdatedAt = record.UpdatedAt.UTC()
	}
	return record, true
}

func NormalizeModelConfigRecord(record ModelConfigRecord) ModelConfigRecord {
	record.Model = strings.TrimSpace(record.Model)
	record.ReasoningEffort = NormalizeClaudeReasoningEffort(record.ReasoningEffort)
	record.AccessMode = agentproto.NormalizeAccessMode(record.AccessMode)
	return record
}

func BotCapabilitySettingsContract(record BotCapabilitySettingsRecord) SurfaceBackendContract {
	normalized, ok := NormalizeBotCapabilitySettingsRecord(record)
	if !ok {
		return HeadlessCodexSurfaceBackendContract("")
	}
	return NormalizeSurfaceBackendContract(SurfaceBackendContract{
		ProductMode:     normalized.ProductMode,
		Backend:         normalized.Backend,
		CodexProviderID: normalized.CodexProviderID,
		ClaudeProfileID: normalized.ClaudeProfileID,
	})
}

func EffectiveSurfaceCapabilitySettings(root *Root, surface *SurfaceConsoleRecord) SurfaceCapabilitySettings {
	if record, ok := surfaceBotCapabilitySettings(root, surface); ok {
		return SurfaceCapabilitySettings{
			Contract:            BotCapabilitySettingsContract(record),
			PromptOverride:      NormalizeModelConfigRecord(record.PromptOverride),
			PlanMode:            NormalizePlanModeSetting(record.PlanMode),
			PlanModeOverrideSet: record.PlanModeOverrideSet,
			Source:              SurfaceCapabilitySettingsSourceBot,
		}
	}
	if surface == nil {
		return SurfaceCapabilitySettings{
			Contract: HeadlessCodexSurfaceBackendContract(""),
			Source:   SurfaceCapabilitySettingsSourceSurface,
		}
	}
	return SurfaceCapabilitySettings{
		Contract:            SurfaceDesiredBackendContract(surface),
		PromptOverride:      NormalizeModelConfigRecord(surface.PromptOverride),
		PlanMode:            NormalizePlanModeSetting(surface.PlanMode),
		PlanModeOverrideSet: surface.PlanModeOverrideSet,
		Source:              SurfaceCapabilitySettingsSourceSurface,
	}
}

func surfaceBotCapabilitySettings(root *Root, surface *SurfaceConsoleRecord) (BotCapabilitySettingsRecord, bool) {
	if root == nil || surface == nil || !surfaceUsesBotCapabilitySettings(surface) {
		return BotCapabilitySettingsRecord{}, false
	}
	key := BotCapabilitySettingsKey(surface.GatewayID)
	if key == "" {
		return BotCapabilitySettingsRecord{}, false
	}
	record, ok := root.BotCapabilitySettings[key]
	if !ok {
		return BotCapabilitySettingsRecord{}, false
	}
	return NormalizeBotCapabilitySettingsRecord(record)
}

func surfaceUsesBotCapabilitySettings(surface *SurfaceConsoleRecord) bool {
	if surface == nil || strings.TrimSpace(surface.ChatID) == "" {
		return false
	}
	if surface.Platform != "" && surface.Platform != "feishu" {
		return false
	}
	parts := strings.Split(strings.TrimSpace(surface.SurfaceSessionID), ":")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "chat" && strings.TrimSpace(parts[i+1]) != "" {
			return true
		}
	}
	return false
}
