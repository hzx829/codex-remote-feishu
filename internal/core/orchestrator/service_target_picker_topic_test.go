package orchestrator

import (
	"strings"
	"testing"
	"time"

	"github.com/kxn/codex-remote-feishu/internal/core/control"
	"github.com/kxn/codex-remote-feishu/internal/core/state"
)

func TestTopicListPickerBindsWorkspaceWithoutSessionDropdown(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	svc := newServiceForTest(&now)
	svc.UpsertInstance(&state.InstanceRecord{
		InstanceID:    "inst-web",
		DisplayName:   "web",
		WorkspaceRoot: "/data/dl/web",
		WorkspaceKey:  "/data/dl/web",
		ShortName:     "web",
		Online:        true,
		Threads: map[string]*state.ThreadRecord{
			"thread-web": {ThreadID: "thread-web", Name: "整理样式", CWD: "/data/dl/web", Loaded: true},
		},
	})

	view := singleTargetPickerEvent(t, svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionListInstances,
		SurfaceSessionID: "feishu:app-1:thread:omt_topic_1",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_user",
	}))
	if !view.HideSessionSelect || view.ShowSessionSelect {
		t.Fatalf("expected topic /list to hide the session dropdown, got %#v", view)
	}
	if view.SelectedSessionValue != targetPickerNewThreadValue || len(view.SessionOptions) != 1 {
		t.Fatalf("expected topic /list to bind a workspace for a new Codex session, got %#v", view)
	}
	if view.ConfirmLabel != "绑定工作区" || view.Title != "绑定话题工作区" || !view.CanConfirm {
		t.Fatalf("unexpected topic workspace binding presentation: %#v", view)
	}
}

func TestTargetPickerSessionOptionCarriesReadableDetails(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	svc := newServiceForTest(&now)
	svc.UpsertInstance(&state.InstanceRecord{
		InstanceID:    "inst-web",
		DisplayName:   "web",
		WorkspaceRoot: "/data/dl/web",
		WorkspaceKey:  "/data/dl/web",
		ShortName:     "web",
		Online:        true,
		Threads: map[string]*state.ThreadRecord{
			"thread-web": {
				ThreadID:                "thread-web",
				Name:                    "整理样式",
				CWD:                     "/data/dl/web",
				ExplicitModel:           "gpt-5.4",
				ExplicitReasoningEffort: "high",
				LastUsedAt:              now.Add(-time.Minute),
				Loaded:                  true,
			},
		},
	})

	view := singleTargetPickerEvent(t, svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionShowThreads,
		SurfaceSessionID: "feishu:app-1:thread:omt_topic_1",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_user",
	}))
	option, ok := targetPickerSessionOption(view, targetPickerThreadValue("thread-web"))
	if !ok {
		t.Fatalf("expected existing session option, got %#v", view.SessionOptions)
	}
	details := strings.Join(option.DetailLines, "\n")
	if len(option.DetailLines) != 4 {
		t.Fatalf("expected four session detail lines, got %#v", option)
	}
	for _, want := range []string{"gpt-5.4", "high", "/data/dl/web", "thread-web"} {
		if !strings.Contains(details, want) {
			t.Fatalf("session details missing %q: %#v", want, option)
		}
	}
}
