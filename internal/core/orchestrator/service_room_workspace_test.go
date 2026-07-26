package orchestrator

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/kxn/codex-remote-feishu/internal/core/agentproto"
	"github.com/kxn/codex-remote-feishu/internal/core/control"
	"github.com/kxn/codex-remote-feishu/internal/core/eventcontract"
	"github.com/kxn/codex-remote-feishu/internal/core/state"
)

type fakeChatAdminAuthorizer struct {
	allowed bool
	calls   []ChatAdminAuthorizationRequest
}

func (f *fakeChatAdminAuthorizer) AuthorizeChatAdmin(_ context.Context, req ChatAdminAuthorizationRequest) ChatAdminAuthorizationDecision {
	f.calls = append(f.calls, req)
	return ChatAdminAuthorizationDecision{Allowed: f.allowed, Reason: "test"}
}

func TestRoomWorkspaceBindingRecordsFirstGroupWorkspaceAttach(t *testing.T) {
	svc := newRoomWorkspaceTestService(t)

	events := svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_owner",
		WorkspaceKey:     "/data/dl/droid",
	})

	if noticeCode(events, "workspace_attached") == "" {
		t.Fatalf("expected workspace attach notice, got %#v", events)
	}
	room := svc.root.FeishuRoomContexts["feishu:chat:oc_room"]
	if room == nil {
		t.Fatal("expected room context")
	}
	if room.WorkspaceKey != "/data/dl/droid" {
		t.Fatalf("room workspace = %q, want /data/dl/droid", room.WorkspaceKey)
	}
	if room.WorkspaceUpdatedBy != "ou_owner" {
		t.Fatalf("room workspace updater = %q, want ou_owner", room.WorkspaceUpdatedBy)
	}
	if room.WorkspaceResetGeneration != 0 {
		t.Fatalf("first binding should not reset room, generation=%d", room.WorkspaceResetGeneration)
	}
}

func TestRoomWorkspaceBindingLetsSameRoomSurfaceInheritWorkspace(t *testing.T) {
	svc := newRoomWorkspaceTestService(t)
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_owner",
		WorkspaceKey:     "/data/dl/droid",
	})

	events := svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-2:chat:oc_room",
		GatewayID:        "app-2",
		ChatID:           "oc_room",
		ActorUserID:      "ou_member",
		WorkspaceKey:     "/data/dl/droid",
	})

	second := svc.root.Surfaces["feishu:app-2:chat:oc_room"]
	if second.AttachedInstanceID != "inst-droid-b" || second.ClaimedWorkspaceKey != "/data/dl/droid" {
		t.Fatalf("expected second same-room surface to attach same workspace independently, got %#v", second)
	}
	if second.SelectedThreadID != "" {
		t.Fatalf("same-room workspace inheritance must not share selected thread, got %q", second.SelectedThreadID)
	}
	if noticeCode(events, "workspace_attached") == "" {
		t.Fatalf("expected second attach notice, got %#v", events)
	}
}

func TestRoomWorkspaceBindingDefaultsTargetPickerForNewSameRoomSurface(t *testing.T) {
	svc := newRoomWorkspaceTestService(t)
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_owner",
		WorkspaceKey:     "/data/dl/web",
	})

	events := svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionShowThreads,
		SurfaceSessionID: "feishu:app-2:chat:oc_room",
		GatewayID:        "app-2",
		ChatID:           "oc_room",
		ActorUserID:      "ou_member",
	})

	view := singleTargetPickerEvent(t, events)
	if view.SelectedWorkspaceKey != "/data/dl/web" {
		t.Fatalf("selected workspace = %q, want room binding /data/dl/web", view.SelectedWorkspaceKey)
	}
}

func TestRoomWorkspaceBindingRecordsGroupAttachInstance(t *testing.T) {
	svc := newRoomWorkspaceTestService(t)

	events := svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachInstance,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_owner",
		InstanceID:       "inst-droid-a",
	})

	if noticeCode(events, "attached") == "" {
		t.Fatalf("expected attach notice, got %#v", events)
	}
	room := svc.root.FeishuRoomContexts["feishu:chat:oc_room"]
	if room == nil || room.WorkspaceKey != "/data/dl/droid" {
		t.Fatalf("expected room workspace binding after attach instance, got %#v", room)
	}
}

func TestRoomWorkspaceSwitchRejectsNonAdminWithoutReset(t *testing.T) {
	authorizer := &fakeChatAdminAuthorizer{allowed: false}
	svc := newRoomWorkspaceTestService(t)
	svc.config.ChatAdminAuthorizer = authorizer
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_owner",
		WorkspaceKey:     "/data/dl/droid",
	})

	events := svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_member",
		WorkspaceKey:     "/data/dl/web",
	})

	if len(authorizer.calls) != 1 {
		t.Fatalf("admin authorizer calls = %d, want 1", len(authorizer.calls))
	}
	if authorizer.calls[0].GatewayID != "app-1" || authorizer.calls[0].ChatID != "oc_room" || authorizer.calls[0].ActorOpenID != "ou_member" {
		t.Fatalf("unexpected admin check request: %#v", authorizer.calls[0])
	}
	if code := noticeCode(events, "room_workspace_admin_required"); code == "" {
		t.Fatalf("expected admin-required notice, got %#v", events)
	}
	room := svc.root.FeishuRoomContexts["feishu:chat:oc_room"]
	if room.WorkspaceKey != "/data/dl/droid" || room.WorkspaceResetGeneration != 0 {
		t.Fatalf("non-admin switch should not mutate room binding, got %#v", room)
	}
	surface := svc.root.Surfaces["feishu:app-1:chat:oc_room"]
	if surface.ClaimedWorkspaceKey != "/data/dl/droid" || surface.AttachedInstanceID != "inst-droid-a" {
		t.Fatalf("non-admin switch should keep surface route, got %#v", surface)
	}
}

func TestRoomWorkspaceSwitchRejectsWhileSameRoomSurfaceRunning(t *testing.T) {
	authorizer := &fakeChatAdminAuthorizer{allowed: true}
	svc := newRoomWorkspaceTestService(t)
	svc.config.ChatAdminAuthorizer = authorizer
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_owner",
		WorkspaceKey:     "/data/dl/droid",
	})
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-2:chat:oc_room",
		GatewayID:        "app-2",
		ChatID:           "oc_room",
		ActorUserID:      "ou_member",
		WorkspaceKey:     "/data/dl/droid",
	})
	running := svc.root.Surfaces["feishu:app-2:chat:oc_room"]
	running.ActiveQueueItemID = "queue-running"
	running.QueueItems["queue-running"] = &state.QueueItemRecord{ID: "queue-running", Status: state.QueueItemRunning}

	events := svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_owner",
		WorkspaceKey:     "/data/dl/web",
	})

	if len(authorizer.calls) != 0 {
		t.Fatalf("active-running switch should fail before admin API call, got %#v", authorizer.calls)
	}
	if noticeCode(events, "room_workspace_busy") == "" {
		t.Fatalf("expected room workspace busy notice, got %#v", events)
	}
	room := svc.root.FeishuRoomContexts["feishu:chat:oc_room"]
	if room.WorkspaceKey != "/data/dl/droid" || room.WorkspaceResetGeneration != 0 {
		t.Fatalf("busy switch should not mutate room binding, got %#v", room)
	}
}

func TestRoomWorkspaceSwitchDoesNotResetSiblingsWhenCurrentSurfaceCannotLeave(t *testing.T) {
	authorizer := &fakeChatAdminAuthorizer{allowed: true}
	svc := newRoomWorkspaceTestService(t)
	svc.config.ChatAdminAuthorizer = authorizer
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_owner",
		WorkspaceKey:     "/data/dl/droid",
	})
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-2:chat:oc_room",
		GatewayID:        "app-2",
		ChatID:           "oc_room",
		ActorUserID:      "ou_member",
		WorkspaceKey:     "/data/dl/droid",
	})
	current := svc.root.Surfaces["feishu:app-1:chat:oc_room"]
	current.QueueItems["queue-1"] = &state.QueueItemRecord{ID: "queue-1", Status: state.QueueItemQueued}
	current.QueuedQueueItemIDs = []string{"queue-1"}

	events := svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_owner",
		WorkspaceKey:     "/data/dl/web",
	})

	if noticeCode(events, "thread_switch_queued") == "" {
		t.Fatalf("expected current-surface queue blocker, got %#v", events)
	}
	if len(authorizer.calls) != 0 {
		t.Fatalf("current-surface blocker should fail before admin API call, got %#v", authorizer.calls)
	}
	room := svc.root.FeishuRoomContexts["feishu:chat:oc_room"]
	if room.WorkspaceKey != "/data/dl/droid" || room.WorkspaceResetGeneration != 0 {
		t.Fatalf("current-surface blocker should not mutate room binding, got %#v", room)
	}
	second := svc.root.Surfaces["feishu:app-2:chat:oc_room"]
	if second.AttachedInstanceID == "" || second.ClaimedWorkspaceKey == "" {
		t.Fatalf("current-surface blocker should not reset sibling surface, got %#v", second)
	}
}

func TestRoomWorkspaceSwitchDoesNotResetSiblingsWhenTargetWorkspaceBusy(t *testing.T) {
	authorizer := &fakeChatAdminAuthorizer{allowed: true}
	svc := newRoomWorkspaceTestService(t)
	svc.config.ChatAdminAuthorizer = authorizer
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_owner",
		WorkspaceKey:     "/data/dl/droid",
	})
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-2:chat:oc_room",
		GatewayID:        "app-2",
		ChatID:           "oc_room",
		ActorUserID:      "ou_member",
		WorkspaceKey:     "/data/dl/droid",
	})
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-1:user:ou_other",
		GatewayID:        "app-1",
		ChatID:           "ou_other",
		ActorUserID:      "ou_other",
		WorkspaceKey:     "/data/dl/web",
	})

	events := svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_owner",
		WorkspaceKey:     "/data/dl/web",
	})

	if noticeCode(events, "workspace_busy") == "" {
		t.Fatalf("expected target workspace busy notice, got %#v", events)
	}
	if len(authorizer.calls) != 0 {
		t.Fatalf("target busy should fail before admin API call, got %#v", authorizer.calls)
	}
	room := svc.root.FeishuRoomContexts["feishu:chat:oc_room"]
	if room.WorkspaceKey != "/data/dl/droid" || room.WorkspaceResetGeneration != 0 {
		t.Fatalf("target busy should not mutate room binding, got %#v", room)
	}
	second := svc.root.Surfaces["feishu:app-2:chat:oc_room"]
	if second.AttachedInstanceID == "" || second.ClaimedWorkspaceKey == "" {
		t.Fatalf("target busy should not reset sibling surface, got %#v", second)
	}
}

func TestRoomWorkspaceFreshWorkspaceCreateRejectsNonAdminBeforePendingHeadless(t *testing.T) {
	authorizer := &fakeChatAdminAuthorizer{allowed: false}
	svc := newRoomWorkspaceTestService(t)
	svc.config.ChatAdminAuthorizer = authorizer
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_owner",
		WorkspaceKey:     "/data/dl/droid",
	})
	surface := svc.root.Surfaces["feishu:app-1:chat:oc_room"]
	surface.ActorUserID = "ou_member"

	events := svc.startFreshWorkspaceHeadless(surface, "/data/dl/new")

	if len(authorizer.calls) != 1 {
		t.Fatalf("admin authorizer calls = %d, want 1", len(authorizer.calls))
	}
	if noticeCode(events, "room_workspace_admin_required") == "" {
		t.Fatalf("expected admin-required notice, got %#v", events)
	}
	if surface.PendingHeadless != nil {
		t.Fatalf("non-admin workspace create must not start pending headless, got %#v", surface.PendingHeadless)
	}
	room := svc.root.FeishuRoomContexts["feishu:chat:oc_room"]
	if room.WorkspaceKey != "/data/dl/droid" || room.WorkspaceResetGeneration != 0 {
		t.Fatalf("non-admin workspace create should keep room binding, got %#v", room)
	}
}

func TestRoomWorkspaceSwitchByAdminResetsSameRoomSurfaces(t *testing.T) {
	authorizer := &fakeChatAdminAuthorizer{allowed: true}
	svc := newRoomWorkspaceTestService(t)
	svc.config.ChatAdminAuthorizer = authorizer
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_owner",
		WorkspaceKey:     "/data/dl/droid",
	})
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-2:chat:oc_room",
		GatewayID:        "app-2",
		ChatID:           "oc_room",
		ActorUserID:      "ou_member",
		WorkspaceKey:     "/data/dl/droid",
	})
	second := svc.root.Surfaces["feishu:app-2:chat:oc_room"]
	second.SelectedThreadID = "thread-droid-b"
	second.RouteMode = state.RouteModePinned
	second.QueueItems["queue-1"] = &state.QueueItemRecord{ID: "queue-1", Status: state.QueueItemQueued}
	second.QueuedQueueItemIDs = []string{"queue-1"}
	second.StagedImages["img-1"] = &state.StagedImageRecord{ImageID: "img-1", State: state.ImageStaged}

	events := svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_owner",
		WorkspaceKey:     "/data/dl/web",
	})

	if noticeCode(events, "workspace_switched") == "" {
		t.Fatalf("expected workspace switched notice, got %#v", events)
	}
	room := svc.root.FeishuRoomContexts["feishu:chat:oc_room"]
	if room.WorkspaceKey != "/data/dl/web" || room.WorkspaceResetGeneration != 1 {
		t.Fatalf("expected room to switch to web with generation 1, got %#v", room)
	}
	if second.AttachedInstanceID != "" || second.SelectedThreadID != "" || second.ClaimedWorkspaceKey != "/data/dl/web" {
		t.Fatalf("expected sibling surface route to reset, got %#v", second)
	}
	if second.ActiveQueueItemID != "" || len(second.QueueItems) != 0 || len(second.QueuedQueueItemIDs) != 0 {
		t.Fatalf("expected sibling queue to reset, got active=%q items=%#v order=%#v", second.ActiveQueueItemID, second.QueueItems, second.QueuedQueueItemIDs)
	}
	if len(second.StagedImages) != 0 || len(second.PendingRequests) != 0 || second.ReviewSession != nil {
		t.Fatalf("expected sibling overlays to reset, got images=%#v requests=%#v review=%#v", second.StagedImages, second.PendingRequests, second.ReviewSession)
	}
}

func TestRoomActiveLockBlocksSecondSameRoomSurfaceDispatch(t *testing.T) {
	svc := newRoomWorkspaceTestService(t)
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_owner",
		WorkspaceKey:     "/data/dl/droid",
	})
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-2:chat:oc_room",
		GatewayID:        "app-2",
		ChatID:           "oc_room",
		ActorUserID:      "ou_member",
		WorkspaceKey:     "/data/dl/droid",
	})
	running := svc.root.Surfaces["feishu:app-1:chat:oc_room"]
	running.ActiveQueueItemID = "queue-running"
	running.QueueItems["queue-running"] = &state.QueueItemRecord{ID: "queue-running", Status: state.QueueItemRunning}

	events := svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionTextMessage,
		SurfaceSessionID: "feishu:app-2:chat:oc_room",
		GatewayID:        "app-2",
		ChatID:           "oc_room",
		ActorUserID:      "ou_member",
		MessageID:        "msg-2",
		Text:             "继续处理",
	})

	if hasAgentCommand(events) {
		t.Fatalf("same-room active lock should block agent dispatch, got %#v", events)
	}
	if noticeCode(events, "room_workspace_active") == "" {
		t.Fatalf("expected room active notice, got %#v", events)
	}
	second := svc.root.Surfaces["feishu:app-2:chat:oc_room"]
	if second.ActiveQueueItemID != "" {
		t.Fatalf("blocked surface should not gain active queue item, got %q", second.ActiveQueueItemID)
	}
	if len(second.QueuedQueueItemIDs) != 1 {
		t.Fatalf("blocked surface should keep queued item for later retry, got %#v", second.QueuedQueueItemIDs)
	}
}

func TestRoomActiveLockDoesNotBlockDifferentRoomDispatch(t *testing.T) {
	svc := newRoomWorkspaceTestService(t)
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_owner",
		WorkspaceKey:     "/data/dl/droid",
	})
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-1:chat:oc_other",
		GatewayID:        "app-1",
		ChatID:           "oc_other",
		ActorUserID:      "ou_other",
		WorkspaceKey:     "/data/dl/web",
	})
	running := svc.root.Surfaces["feishu:app-1:chat:oc_room"]
	running.ActiveQueueItemID = "queue-running"
	running.QueueItems["queue-running"] = &state.QueueItemRecord{ID: "queue-running", Status: state.QueueItemRunning}

	events := svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionTextMessage,
		SurfaceSessionID: "feishu:app-1:chat:oc_other",
		GatewayID:        "app-1",
		ChatID:           "oc_other",
		ActorUserID:      "ou_other",
		MessageID:        "msg-other",
		Text:             "另一个群继续处理",
	})

	if !hasAgentCommand(events) {
		t.Fatalf("different room should dispatch normally, got %#v", events)
	}
}

func TestRoomActiveLockReleaseAllowsSameRoomSurfaceDispatch(t *testing.T) {
	svc := newRoomWorkspaceTestService(t)
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_owner",
		WorkspaceKey:     "/data/dl/droid",
	})
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-2:chat:oc_room",
		GatewayID:        "app-2",
		ChatID:           "oc_room",
		ActorUserID:      "ou_member",
		WorkspaceKey:     "/data/dl/droid",
	})
	firstEvents := svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionTextMessage,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_owner",
		MessageID:        "msg-1",
		Text:             "先处理",
	})
	if !hasAgentCommand(firstEvents) {
		t.Fatalf("first surface should dispatch, got %#v", firstEvents)
	}
	first := svc.root.Surfaces["feishu:app-1:chat:oc_room"]
	activeID := first.ActiveQueueItemID
	if activeID == "" {
		t.Fatal("expected first surface to have active queue item")
	}
	svc.clearSurfaceActiveQueueItem(first, activeID)

	secondEvents := svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionTextMessage,
		SurfaceSessionID: "feishu:app-2:chat:oc_room",
		GatewayID:        "app-2",
		ChatID:           "oc_room",
		ActorUserID:      "ou_member",
		MessageID:        "msg-2",
		Text:             "再处理",
	})

	if !hasAgentCommand(secondEvents) {
		t.Fatalf("same-room surface should dispatch after lock release, got %#v", secondEvents)
	}
}

func TestRoomActiveLockBlocksAutoContinueDispatch(t *testing.T) {
	svc := newRoomWorkspaceTestService(t)
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_owner",
		WorkspaceKey:     "/data/dl/droid",
	})
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-2:chat:oc_room",
		GatewayID:        "app-2",
		ChatID:           "oc_room",
		ActorUserID:      "ou_member",
		WorkspaceKey:     "/data/dl/droid",
	})
	running := svc.root.Surfaces["feishu:app-1:chat:oc_room"]
	running.ActiveQueueItemID = "queue-running"
	running.QueueItems["queue-running"] = &state.QueueItemRecord{ID: "queue-running", Status: state.QueueItemRunning}
	second := svc.root.Surfaces["feishu:app-2:chat:oc_room"]
	second.AutoContinue.Enabled = true
	second.AutoContinue.Episode = &state.PendingAutoContinueEpisodeRecord{
		EpisodeID:    "autocontinue-1",
		InstanceID:   "inst-droid-b",
		State:        state.AutoContinueEpisodeScheduled,
		PendingDueAt: svc.now(),
	}

	events := svc.maybeDispatchPendingAutoContinue(second, svc.now())

	if hasAgentCommand(events) {
		t.Fatalf("same-room active lock should block auto-continue dispatch, got %#v", events)
	}
	if noticeCode(events, "room_workspace_active") == "" {
		t.Fatalf("expected room active notice, got %#v", events)
	}
	if second.ActiveQueueItemID != "" {
		t.Fatalf("blocked auto-continue should not create active queue item, got %q", second.ActiveQueueItemID)
	}
	if second.AutoContinue.Episode == nil || second.AutoContinue.Episode.State != state.AutoContinueEpisodeScheduled {
		t.Fatalf("blocked auto-continue should remain scheduled, got %#v", second.AutoContinue.Episode)
	}

	repeated := svc.maybeDispatchPendingAutoContinue(second, svc.now())
	if noticeCode(repeated, "room_workspace_active") != "" {
		t.Fatalf("repeated blocked auto-continue tick should not resend room active notice, got %#v", repeated)
	}
}

func TestRoomActiveLockBlocksAutoWhipWithoutConsumingPending(t *testing.T) {
	svc := newRoomWorkspaceTestService(t)
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_owner",
		WorkspaceKey:     "/data/dl/droid",
	})
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-2:chat:oc_room",
		GatewayID:        "app-2",
		ChatID:           "oc_room",
		ActorUserID:      "ou_member",
		WorkspaceKey:     "/data/dl/droid",
	})
	running := svc.root.Surfaces["feishu:app-1:chat:oc_room"]
	running.ActiveQueueItemID = "queue-running"
	running.QueueItems["queue-running"] = &state.QueueItemRecord{ID: "queue-running", Status: state.QueueItemRunning}
	second := svc.root.Surfaces["feishu:app-2:chat:oc_room"]
	second.AutoWhip.Enabled = true
	second.AutoWhip.PendingReason = state.AutoWhipReasonIncompleteStop
	second.AutoWhip.PendingDueAt = svc.now()
	second.AutoWhip.ConsecutiveCount = 1
	second.AutoWhip.PendingReplyToMessageID = "msg-root"

	events := svc.maybeDispatchPendingAutoWhip(second, svc.now())

	if hasAgentCommand(events) {
		t.Fatalf("same-room active lock should block auto-whip dispatch, got %#v", events)
	}
	if noticeCode(events, "room_workspace_active") == "" {
		t.Fatalf("expected room active notice, got %#v", events)
	}
	if second.AutoWhip.PendingReason != state.AutoWhipReasonIncompleteStop || second.AutoWhip.PendingDueAt.IsZero() {
		t.Fatalf("blocked auto-whip should keep pending retry, got %#v", second.AutoWhip)
	}
	if len(second.QueuedQueueItemIDs) != 0 {
		t.Fatalf("blocked auto-whip should not enqueue a queue item, got %#v", second.QueuedQueueItemIDs)
	}

	repeated := svc.maybeDispatchPendingAutoWhip(second, svc.now())
	if noticeCode(repeated, "room_workspace_active") != "" {
		t.Fatalf("repeated blocked auto-whip tick should not resend room active notice, got %#v", repeated)
	}
}

func TestRoomActiveLockRefreshesOnTurnStarted(t *testing.T) {
	svc := newRoomWorkspaceTestService(t)
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_owner",
		WorkspaceKey:     "/data/dl/droid",
	})
	events := svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionTextMessage,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_owner",
		MessageID:        "msg-1",
		Text:             "先处理",
	})
	if !hasAgentCommand(events) {
		t.Fatalf("expected initial dispatch, got %#v", events)
	}

	svc.ApplyAgentEvent("inst-droid-a", agentproto.Event{
		Kind:     agentproto.EventTurnStarted,
		ThreadID: "thread-droid-a",
		TurnID:   "turn-1",
		Initiator: agentproto.Initiator{
			Kind:             agentproto.InitiatorRemoteSurface,
			SurfaceSessionID: "feishu:app-1:chat:oc_room",
		},
	})

	room := svc.root.FeishuRoomContexts["feishu:chat:oc_room"]
	if room == nil || room.ActiveLock == nil {
		t.Fatalf("expected active lock, got %#v", room)
	}
	if room.ActiveLock.ThreadID != "thread-droid-a" || room.ActiveLock.TurnID != "turn-1" || room.ActiveLock.Reason != "running" {
		t.Fatalf("active lock = %#v, want turn/thread running lock", room.ActiveLock)
	}
}

func TestRoomActiveLockStaleRecordDoesNotBlockDispatch(t *testing.T) {
	svc := newRoomWorkspaceTestService(t)
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_owner",
		WorkspaceKey:     "/data/dl/droid",
	})
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-2:chat:oc_room",
		GatewayID:        "app-2",
		ChatID:           "oc_room",
		ActorUserID:      "ou_member",
		WorkspaceKey:     "/data/dl/droid",
	})
	room := svc.root.FeishuRoomContexts["feishu:chat:oc_room"]
	room.ActiveLock = &state.FeishuRoomActiveLockRecord{
		SurfaceSessionID: "feishu:app-missing:chat:oc_room",
		InstanceID:       "inst-missing",
		QueueItemID:      "queue-missing",
		Reason:           "running",
		UpdatedAt:        svc.now(),
	}

	events := svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionTextMessage,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_owner",
		MessageID:        "msg-1",
		Text:             "继续处理",
	})

	if !hasAgentCommand(events) {
		t.Fatalf("stale active lock should not block dispatch, got %#v", events)
	}
	if room.ActiveLock == nil || room.ActiveLock.SurfaceSessionID != "feishu:app-1:chat:oc_room" {
		t.Fatalf("expected stale lock to be replaced by current dispatch lock, got %#v", room.ActiveLock)
	}
}

func TestRoomActiveLockRoomWorkspaceResetClearsStaleRecord(t *testing.T) {
	authorizer := &fakeChatAdminAuthorizer{allowed: true}
	svc := newRoomWorkspaceTestService(t)
	svc.config.ChatAdminAuthorizer = authorizer
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_owner",
		WorkspaceKey:     "/data/dl/droid",
	})
	room := svc.root.FeishuRoomContexts["feishu:chat:oc_room"]
	room.ActiveLock = &state.FeishuRoomActiveLockRecord{
		SurfaceSessionID: "feishu:app-missing:chat:oc_room",
		InstanceID:       "inst-missing",
		QueueItemID:      "queue-missing",
		Reason:           "running",
		UpdatedAt:        svc.now(),
	}

	events := svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_owner",
		WorkspaceKey:     "/data/dl/web",
	})

	if noticeCode(events, "workspace_switched") == "" {
		t.Fatalf("expected workspace switch to complete, got %#v", events)
	}
	if room.ActiveLock != nil {
		t.Fatalf("room workspace reset should clear stale active lock, got %#v", room.ActiveLock)
	}
}

func TestPrivateWorkspaceAttachDoesNotCreateRoomBindingOrAdminCheck(t *testing.T) {
	authorizer := &fakeChatAdminAuthorizer{allowed: false}
	svc := newRoomWorkspaceTestService(t)
	svc.config.ChatAdminAuthorizer = authorizer

	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "feishu:app-1:user:ou_owner",
		GatewayID:        "app-1",
		ChatID:           "ou_owner",
		ActorUserID:      "ou_owner",
		WorkspaceKey:     "/data/dl/droid",
	})

	if len(authorizer.calls) != 0 {
		t.Fatalf("private attach should not call admin authorizer, got %#v", authorizer.calls)
	}
	if len(svc.root.FeishuRoomContexts) != 0 {
		t.Fatalf("private attach should not create room binding, got %#v", svc.root.FeishuRoomContexts)
	}
}

func newRoomWorkspaceTestService(t *testing.T) *Service {
	t.Helper()
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	svc := newServiceForTest(&now)
	for _, inst := range []*state.InstanceRecord{
		{
			InstanceID:    "inst-droid-a",
			DisplayName:   "droid-a",
			WorkspaceRoot: "/data/dl/droid",
			WorkspaceKey:  "/data/dl/droid",
			Online:        true,
			Threads: map[string]*state.ThreadRecord{
				"thread-droid-a": {ThreadID: "thread-droid-a", Name: "A", CWD: "/data/dl/droid", Loaded: true},
			},
		},
		{
			InstanceID:    "inst-droid-b",
			DisplayName:   "droid-b",
			WorkspaceRoot: "/data/dl/droid",
			WorkspaceKey:  "/data/dl/droid",
			Online:        true,
			Threads: map[string]*state.ThreadRecord{
				"thread-droid-b": {ThreadID: "thread-droid-b", Name: "B", CWD: "/data/dl/droid", Loaded: true},
			},
		},
		{
			InstanceID:    "inst-web",
			DisplayName:   "web",
			WorkspaceRoot: "/data/dl/web",
			WorkspaceKey:  "/data/dl/web",
			Online:        true,
			Threads: map[string]*state.ThreadRecord{
				"thread-web": {ThreadID: "thread-web", Name: "Web", CWD: "/data/dl/web", Loaded: true},
			},
		},
	} {
		svc.UpsertInstance(inst)
	}
	return svc
}

func noticeCode(events []eventcontract.Event, code string) string {
	for _, event := range events {
		if event.Notice != nil && event.Notice.Code == code {
			return event.Notice.Code
		}
	}
	return ""
}

func noticeTextContains(events []eventcontract.Event, code, text string) bool {
	for _, event := range events {
		if event.Notice != nil && event.Notice.Code == code && strings.Contains(event.Notice.Text, text) {
			return true
		}
	}
	return false
}

func hasAgentCommand(events []eventcontract.Event) bool {
	for _, event := range events {
		if event.Kind == eventcontract.KindAgentCommand {
			return true
		}
	}
	return false
}
