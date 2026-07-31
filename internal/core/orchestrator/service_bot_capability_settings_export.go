package orchestrator

import (
	"sort"

	"github.com/kxn/codex-remote-feishu/internal/core/state"
)

func (s *Service) MaterializeBotCapabilitySettings(records []state.BotCapabilitySettingsRecord) {
	if s == nil || s.root == nil {
		return
	}
	s.root.BotCapabilitySettings = map[string]state.BotCapabilitySettingsRecord{}
	for _, record := range records {
		normalized, ok := state.NormalizeBotCapabilitySettingsRecord(record)
		if !ok {
			continue
		}
		key := state.BotCapabilitySettingsKey(normalized.GatewayID)
		if key == "" {
			continue
		}
		s.root.BotCapabilitySettings[key] = normalized
	}
	for _, record := range s.root.BotCapabilitySettings {
		s.projectBotCapabilitySettingsToGatewaySurfaces(nil, record)
	}
}

func (s *Service) BotCapabilitySettings() []state.BotCapabilitySettingsRecord {
	if s == nil || s.root == nil || len(s.root.BotCapabilitySettings) == 0 {
		return nil
	}
	keys := make([]string, 0, len(s.root.BotCapabilitySettings))
	for key := range s.root.BotCapabilitySettings {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	records := make([]state.BotCapabilitySettingsRecord, 0, len(keys))
	for _, key := range keys {
		record, ok := state.NormalizeBotCapabilitySettingsRecord(s.root.BotCapabilitySettings[key])
		if !ok || state.BotCapabilitySettingsKey(record.GatewayID) != key {
			continue
		}
		records = append(records, record)
	}
	return records
}
