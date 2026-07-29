package orchestrator

import (
	"sort"

	"github.com/kxn/codex-remote-feishu/internal/core/state"
)

func (s *Service) MaterializeFeishuRoomPrimaryState(records []state.FeishuRoomPrimaryRecord) {
	if s == nil || s.root == nil {
		return
	}
	if s.root.FeishuRoomContexts == nil {
		s.root.FeishuRoomContexts = map[string]*state.FeishuRoomContextRecord{}
	}
	for _, record := range records {
		normalized, ok := state.NormalizeFeishuRoomPrimaryRecord(record)
		if !ok {
			continue
		}
		room := s.root.FeishuRoomContexts[normalized.RoomID]
		if room == nil {
			room = &state.FeishuRoomContextRecord{
				RoomID:            normalized.RoomID,
				ChatID:            normalized.ChatID,
				GatewayIDs:        map[string]bool{},
				SurfaceSessionIDs: map[string]bool{},
			}
			s.root.FeishuRoomContexts[normalized.RoomID] = room
		}
		room.RoomID = normalized.RoomID
		room.ChatID = normalized.ChatID
		room.PrimaryGatewayID = normalized.PrimaryGatewayID
		room.PrimaryUpdatedBy = normalized.PrimaryUpdatedBy
		room.PrimaryUpdatedAt = normalized.PrimaryUpdatedAt
	}
}

func (s *Service) FeishuRoomPrimaryState() []state.FeishuRoomPrimaryRecord {
	if s == nil || s.root == nil || len(s.root.FeishuRoomContexts) == 0 {
		return nil
	}
	keys := make([]string, 0, len(s.root.FeishuRoomContexts))
	for key := range s.root.FeishuRoomContexts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	records := make([]state.FeishuRoomPrimaryRecord, 0, len(keys))
	for _, key := range keys {
		room := s.root.FeishuRoomContexts[key]
		if room == nil || room.PrimaryGatewayID == "" {
			continue
		}
		record, ok := state.FeishuRoomPrimaryRecordFromContext(room)
		if !ok {
			continue
		}
		records = append(records, record)
	}
	return records
}

func (s *Service) FeishuRoomPrimaryGateway(chatID string) string {
	if s == nil || s.root == nil {
		return ""
	}
	room := s.root.FeishuRoomContexts[state.FeishuRoomPrimaryKey(chatID)]
	if room == nil {
		return ""
	}
	return room.PrimaryGatewayID
}
