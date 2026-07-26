package daemon

import (
	"strings"

	"github.com/kxn/codex-remote-feishu/internal/app/daemon/surfaceresume"
	"github.com/kxn/codex-remote-feishu/internal/core/state"
)

func surfaceResumeEntryAllowsBackgroundRecovery(entry surfaceresume.Entry) bool {
	return !surfaceResumeEntryIsFeishuGroup(entry)
}

func surfaceAllowsDaemonLifecycleNotice(surface *state.SurfaceConsoleRecord) bool {
	return !surfaceIsFeishuGroup(strings.TrimSpace(surfaceIDFromSurface(surface)))
}

func surfaceResumeEntryIsFeishuGroup(entry surfaceresume.Entry) bool {
	return surfaceIsFeishuGroup(strings.TrimSpace(entry.SurfaceSessionID))
}

func surfaceIDFromSurface(surface *state.SurfaceConsoleRecord) string {
	if surface == nil {
		return ""
	}
	return surface.SurfaceSessionID
}

func surfaceIsFeishuGroup(surfaceID string) bool {
	ref, ok := parseFeishuSurfaceID(surfaceID)
	return ok && ref.scopeKind == "chat" && ref.scopeID != ""
}

type feishuSurfaceRef struct {
	gatewayID string
	scopeKind string
	scopeID   string
}

func parseFeishuSurfaceID(surfaceID string) (feishuSurfaceRef, bool) {
	parts := strings.Split(strings.TrimSpace(surfaceID), ":")
	if len(parts) != 4 || parts[0] != "feishu" {
		return feishuSurfaceRef{}, false
	}
	ref := feishuSurfaceRef{
		gatewayID: strings.TrimSpace(parts[1]),
		scopeKind: strings.TrimSpace(parts[2]),
		scopeID:   strings.TrimSpace(parts[3]),
	}
	if ref.gatewayID == "" || ref.scopeKind == "" || ref.scopeID == "" {
		return feishuSurfaceRef{}, false
	}
	return ref, true
}
