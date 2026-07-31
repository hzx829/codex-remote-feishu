package orchestrator

import (
	"testing"
	"time"

	"github.com/kxn/codex-remote-feishu/internal/core/agentproto"
	"github.com/kxn/codex-remote-feishu/internal/core/control"
	"github.com/kxn/codex-remote-feishu/internal/core/state"
)

func TestTargetPickerGroupsGlobalCodexSnapshotByThreadCWD(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	svc := newServiceForTest(&now)
	svc.UpsertInstance(&state.InstanceRecord{
		InstanceID:    "inst-catalog",
		DisplayName:   "catalog",
		WorkspaceRoot: "/runtime/catalog",
		WorkspaceKey:  "/runtime/catalog",
		ShortName:     "catalog",
		Source:        "headless",
		Managed:       true,
		Online:        true,
		Threads:       map[string]*state.ThreadRecord{},
	})
	svc.SetPersistedThreadCatalog(&fakePersistedThreadCatalog{
		recent: []state.ThreadRecord{
			{ThreadID: "thread-droid", Name: "Fix login", CWD: "/data/dl/droid", Loaded: true, LastUsedAt: now.Add(-time.Minute)},
			{ThreadID: "thread-web", Name: "Polish styles", CWD: "/data/dl/web", Loaded: true, LastUsedAt: now.Add(-2 * time.Minute)},
		},
		recentWorkspaces: map[string]time.Time{
			"/data/dl/droid": now.Add(-time.Minute),
			"/data/dl/web":   now.Add(-2 * time.Minute),
		},
	})
	svc.ApplyAgentEvent("inst-catalog", agentproto.Event{
		Kind: agentproto.EventThreadsSnapshot,
		Threads: []agentproto.ThreadSnapshotRecord{
			{ThreadID: "thread-droid", Name: "Fix login", CWD: "/data/dl/droid", Loaded: true, ListOrder: 1},
			{ThreadID: "thread-web", Name: "Polish styles", CWD: "/data/dl/web", Loaded: true, ListOrder: 2},
		},
	})

	initial := singleTargetPickerEvent(t, svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionListInstances,
		SurfaceSessionID: "surface-1",
		ChatID:           "chat-1",
		ActorUserID:      "user-1",
	}))
	if _, ok := targetPickerWorkspaceOption(initial, "/data/dl/droid"); !ok {
		t.Fatalf("expected droid workspace from thread cwd, got %#v", initial.WorkspaceOptions)
	}
	if _, ok := targetPickerWorkspaceOption(initial, "/data/dl/web"); !ok {
		t.Fatalf("expected web workspace from thread cwd, got %#v", initial.WorkspaceOptions)
	}

	view := singleTargetPickerEvent(t, svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionTargetPickerSelectWorkspace,
		SurfaceSessionID: "surface-1",
		ChatID:           "chat-1",
		ActorUserID:      "user-1",
		PickerID:         initial.PickerID,
		WorkspaceKey:     "/data/dl/web",
	}))
	if view.SelectedWorkspaceKey != "/data/dl/web" {
		t.Fatalf("expected workspace selection to remain on web, got %#v", view)
	}
	if _, ok := targetPickerSessionOption(view, targetPickerThreadValue("thread-web")); !ok {
		t.Fatalf("expected web session after workspace switch, got %#v", view.SessionOptions)
	}
	if _, ok := targetPickerSessionOption(view, targetPickerThreadValue("thread-droid")); ok {
		t.Fatalf("expected droid session to be excluded from web workspace, got %#v", view.SessionOptions)
	}
}
