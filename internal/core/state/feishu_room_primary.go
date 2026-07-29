package state

import "strings"

func FeishuRoomPrimaryKey(roomOrChatID string) string {
	value := strings.TrimSpace(roomOrChatID)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "feishu:chat:") {
		return value
	}
	return "feishu:chat:" + value
}

func NormalizeFeishuRoomPrimaryRecord(record FeishuRoomPrimaryRecord) (FeishuRoomPrimaryRecord, bool) {
	record.RoomID = FeishuRoomPrimaryKey(record.RoomID)
	record.ChatID = strings.TrimSpace(record.ChatID)
	if record.RoomID == "" && record.ChatID != "" {
		record.RoomID = FeishuRoomPrimaryKey(record.ChatID)
	}
	if record.ChatID == "" && strings.HasPrefix(record.RoomID, "feishu:chat:") {
		record.ChatID = strings.TrimPrefix(record.RoomID, "feishu:chat:")
	}
	if record.RoomID == "" || record.ChatID == "" {
		return FeishuRoomPrimaryRecord{}, false
	}
	record.PrimaryGatewayID = strings.TrimSpace(record.PrimaryGatewayID)
	record.PrimaryUpdatedBy = strings.TrimSpace(record.PrimaryUpdatedBy)
	if !record.PrimaryUpdatedAt.IsZero() {
		record.PrimaryUpdatedAt = record.PrimaryUpdatedAt.UTC()
	}
	return record, true
}

func FeishuRoomPrimaryRecordFromContext(room *FeishuRoomContextRecord) (FeishuRoomPrimaryRecord, bool) {
	if room == nil {
		return FeishuRoomPrimaryRecord{}, false
	}
	return NormalizeFeishuRoomPrimaryRecord(FeishuRoomPrimaryRecord{
		RoomID:           room.RoomID,
		ChatID:           room.ChatID,
		PrimaryGatewayID: room.PrimaryGatewayID,
		PrimaryUpdatedBy: room.PrimaryUpdatedBy,
		PrimaryUpdatedAt: room.PrimaryUpdatedAt,
	})
}
