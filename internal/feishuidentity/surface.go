package feishuidentity

import "strings"

const (
	PlatformFeishu  = "feishu"
	ScopeKindUser   = "user"
	ScopeKindChat   = "chat"
	ScopeKindThread = "thread"
)

type SurfaceRef struct {
	Platform  string
	GatewayID string
	ScopeKind string
	ScopeID   string
}

func ParseSurfaceRef(surfaceID string) (SurfaceRef, bool) {
	parts := strings.Split(strings.TrimSpace(surfaceID), ":")
	if len(parts) != 4 || parts[0] != PlatformFeishu {
		return SurfaceRef{}, false
	}
	ref := SurfaceRef{
		Platform:  parts[0],
		GatewayID: strings.TrimSpace(parts[1]),
		ScopeKind: strings.TrimSpace(parts[2]),
		ScopeID:   strings.TrimSpace(parts[3]),
	}
	if !ref.valid() {
		return SurfaceRef{}, false
	}
	return ref, true
}

func (r SurfaceRef) SurfaceID() string {
	r.Platform = strings.TrimSpace(r.Platform)
	r.GatewayID = strings.TrimSpace(r.GatewayID)
	r.ScopeKind = strings.TrimSpace(r.ScopeKind)
	r.ScopeID = strings.TrimSpace(r.ScopeID)
	if !r.valid() {
		return ""
	}
	return strings.Join([]string{PlatformFeishu, r.GatewayID, r.ScopeKind, r.ScopeID}, ":")
}

func (r SurfaceRef) IsUser() bool {
	return r.valid() && strings.TrimSpace(r.ScopeKind) == ScopeKindUser
}

func (r SurfaceRef) IsChat() bool {
	return r.valid() && strings.TrimSpace(r.ScopeKind) == ScopeKindChat
}

func (r SurfaceRef) IsThread() bool {
	return r.valid() && strings.TrimSpace(r.ScopeKind) == ScopeKindThread
}

func (r SurfaceRef) valid() bool {
	if strings.TrimSpace(r.Platform) != PlatformFeishu {
		return false
	}
	if strings.TrimSpace(r.GatewayID) == "" || strings.TrimSpace(r.ScopeID) == "" {
		return false
	}
	switch strings.TrimSpace(r.ScopeKind) {
	case ScopeKindUser, ScopeKindChat, ScopeKindThread:
		return true
	default:
		return false
	}
}
