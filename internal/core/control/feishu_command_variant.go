package control

import "strings"

func FeishuCommandVariantIDForContext(commandID string, ctx CatalogContext) string {
	commandID = strings.TrimSpace(commandID)
	if commandID == "" {
		return ""
	}
	normalized := NormalizeCatalogContext(ctx)
	if commandID == FeishuCommandPrimary {
		return commandID + "." + string(normalized.Backend) + "." + normalized.ProductMode + "." +
			normalized.SurfaceScopeKind + "." + normalized.PrimaryBotState + "." + normalized.PrimaryPermissionState
	}
	return commandID + "." + string(normalized.Backend) + "." + normalized.ProductMode
}

func feishuCommandVariantIDForContext(commandID string, ctx CatalogContext) string {
	return FeishuCommandVariantIDForContext(commandID, ctx)
}
