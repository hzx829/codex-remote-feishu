package feishu

import (
	"testing"

	gatewaypkg "github.com/kxn/codex-remote-feishu/internal/adapter/feishu/gateway"
	"github.com/kxn/codex-remote-feishu/internal/feishuidentity"
)

func TestSurfaceIDForInboundUsesThreadScopeForGroupTopic(t *testing.T) {
	got := gatewaypkg.SurfaceIDForInbound("app-1", "oc_xxx", "group", "omt_topic_1", "user-1")
	if got != "feishu:app-1:thread:omt_topic_1" {
		t.Fatalf("unexpected group topic surface id: %q", got)
	}
}

func TestParseSurfaceRefAcceptsTopic(t *testing.T) {
	ref, ok := feishuidentity.ParseSurfaceRef("feishu:app-1:thread:omt_1")
	if !ok {
		t.Fatal("expected topic surface id to parse")
	}
	if ref.GatewayID != "app-1" || ref.ScopeKind != feishuidentity.ScopeKindThread || ref.ScopeID != "omt_1" {
		t.Fatalf("unexpected topic surface ref: %#v", ref)
	}
}
