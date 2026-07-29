package control

import (
	"testing"

	"github.com/kxn/codex-remote-feishu/internal/core/agentproto"
)

func TestNormalizeCatalogContextDefaultsToCodexDetached(t *testing.T) {
	ctx := NormalizeCatalogContext(CatalogContext{})
	if ctx.Backend != agentproto.BackendCodex {
		t.Fatalf("Backend = %q, want %q", ctx.Backend, agentproto.BackendCodex)
	}
	if ctx.ProductMode != "normal" {
		t.Fatalf("ProductMode = %q, want normal", ctx.ProductMode)
	}
	if ctx.MenuStage != string(FeishuCommandMenuStageDetached) {
		t.Fatalf("MenuStage = %q, want %q", ctx.MenuStage, FeishuCommandMenuStageDetached)
	}
	if ctx.AttachedKind != string(CatalogAttachedKindDetached) {
		t.Fatalf("AttachedKind = %q, want %q", ctx.AttachedKind, CatalogAttachedKindDetached)
	}
	if !ctx.Capabilities.ThreadsRefresh || !ctx.Capabilities.TurnSteer || !ctx.Capabilities.RequestRespond || !ctx.Capabilities.ResumeByThreadID || !ctx.Capabilities.VSCodeMode {
		t.Fatalf("expected codex default capabilities, got %#v", ctx.Capabilities)
	}
	if ctx.Capabilities.SessionCatalog || ctx.Capabilities.RequiresCWDForResume {
		t.Fatalf("unexpected codex-only capabilities: %#v", ctx.Capabilities)
	}
	if ctx.SurfaceScopeKind != string(CatalogSurfaceScopeKindUser) {
		t.Fatalf("SurfaceScopeKind = %q, want %q", ctx.SurfaceScopeKind, CatalogSurfaceScopeKindUser)
	}
	if ctx.PrimaryBotState != string(CatalogPrimaryBotStateUnknown) {
		t.Fatalf("PrimaryBotState = %q, want %q", ctx.PrimaryBotState, CatalogPrimaryBotStateUnknown)
	}
	if ctx.PrimaryPermissionState != string(CatalogPrimaryPermissionStateUnknown) {
		t.Fatalf("PrimaryPermissionState = %q, want %q", ctx.PrimaryPermissionState, CatalogPrimaryPermissionStateUnknown)
	}
}

func TestNormalizeCatalogContextNormalizesPrimaryProjectionState(t *testing.T) {
	ctx := NormalizeCatalogContext(CatalogContext{
		SurfaceScopeKind:       " CHAT ",
		PrimaryBotState:        " Current ",
		PrimaryPermissionState: " Granted ",
	})
	if ctx.SurfaceScopeKind != string(CatalogSurfaceScopeKindChat) {
		t.Fatalf("SurfaceScopeKind = %q, want %q", ctx.SurfaceScopeKind, CatalogSurfaceScopeKindChat)
	}
	if ctx.PrimaryBotState != string(CatalogPrimaryBotStateCurrent) {
		t.Fatalf("PrimaryBotState = %q, want %q", ctx.PrimaryBotState, CatalogPrimaryBotStateCurrent)
	}
	if ctx.PrimaryPermissionState != string(CatalogPrimaryPermissionStateGranted) {
		t.Fatalf("PrimaryPermissionState = %q, want %q", ctx.PrimaryPermissionState, CatalogPrimaryPermissionStateGranted)
	}
}

func TestNormalizeCatalogContextRejectsUnknownPrimaryProjectionState(t *testing.T) {
	ctx := NormalizeCatalogContext(CatalogContext{
		SurfaceScopeKind:       "channel",
		PrimaryBotState:        "self",
		PrimaryPermissionState: "ok",
	})
	if ctx.SurfaceScopeKind != string(CatalogSurfaceScopeKindUser) {
		t.Fatalf("SurfaceScopeKind = %q, want %q", ctx.SurfaceScopeKind, CatalogSurfaceScopeKindUser)
	}
	if ctx.PrimaryBotState != string(CatalogPrimaryBotStateUnknown) {
		t.Fatalf("PrimaryBotState = %q, want %q", ctx.PrimaryBotState, CatalogPrimaryBotStateUnknown)
	}
	if ctx.PrimaryPermissionState != string(CatalogPrimaryPermissionStateUnknown) {
		t.Fatalf("PrimaryPermissionState = %q, want %q", ctx.PrimaryPermissionState, CatalogPrimaryPermissionStateUnknown)
	}
}
