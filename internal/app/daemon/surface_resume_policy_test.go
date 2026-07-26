package daemon

import (
	"context"
	"testing"
	"time"

	"github.com/kxn/codex-remote-feishu/internal/app/daemon/surfaceresume"
	relayruntime "github.com/kxn/codex-remote-feishu/internal/runtime"
)

func TestFeishuGroupSurfaceResumeStateMaterializesButSkipsBackgroundRecovery(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	workspaceDir := t.TempDir()
	putSurfaceResumeStateForTest(t, stateDir, surfaceresume.Entry{
		SurfaceSessionID:   "feishu:app-1:chat:oc_room",
		GatewayID:          "app-1",
		ChatID:             "oc_room",
		ActorUserID:        "ou_user",
		ProductMode:        "normal",
		Backend:            "codex",
		ResumeThreadID:     "thread-1",
		ResumeThreadTitle:  "修复登录流程",
		ResumeThreadCWD:    workspaceDir,
		ResumeWorkspaceKey: workspaceDir,
		ResumeRouteMode:    "pinned",
		ResumeHeadless:     true,
	})
	app := newRestoreHintTestApp(stateDir)
	app.startHeadless = func(relayruntime.HeadlessLaunchOptions) (int, error) {
		t.Fatal("group background recovery must not start headless")
		return 0, nil
	}

	snapshot := app.service.SurfaceSnapshot("feishu:app-1:chat:oc_room")
	if snapshot == nil {
		t.Fatal("expected group surface to materialize from resume state")
	}
	if _, ok := app.surfaceResumeRuntime.recovery["feishu:app-1:chat:oc_room"]; ok {
		t.Fatalf("expected group surface to stay out of background recovery map, got %#v", app.surfaceResumeRuntime.recovery)
	}

	app.onTick(context.Background(), time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC))

	gateway := app.gateway.(*recordingGateway)
	if len(gateway.operations) != 0 {
		t.Fatalf("expected group background recovery tick to stay silent, got %#v", gateway.operations)
	}
	if snapshot := app.service.SurfaceSnapshot("feishu:app-1:chat:oc_room"); snapshot == nil || snapshot.PendingHeadless.InstanceID != "" {
		t.Fatalf("expected group surface to remain detached without pending headless, got %#v", snapshot)
	}
}

func TestFeishuUserSurfaceResumeStateStillEntersBackgroundRecovery(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	workspaceDir := t.TempDir()
	putSurfaceResumeStateForTest(t, stateDir, surfaceresume.Entry{
		SurfaceSessionID:   "feishu:app-1:user:ou_user",
		GatewayID:          "app-1",
		ChatID:             "oc_p2p",
		ActorUserID:        "ou_user",
		ProductMode:        "normal",
		Backend:            "codex",
		ResumeThreadID:     "thread-1",
		ResumeThreadTitle:  "修复登录流程",
		ResumeThreadCWD:    workspaceDir,
		ResumeWorkspaceKey: workspaceDir,
		ResumeRouteMode:    "pinned",
		ResumeHeadless:     true,
	})
	app := newRestoreHintTestApp(stateDir)

	if _, ok := app.surfaceResumeRuntime.recovery["feishu:app-1:user:ou_user"]; !ok {
		t.Fatalf("expected Feishu p2p surface to enter background recovery map, got %#v", app.surfaceResumeRuntime.recovery)
	}
}

func TestFeishuGroupVSCodeResumeDoesNotEmitDetachedPromptOnStartup(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	putSurfaceResumeStateForTest(t, stateDir, surfaceresume.Entry{
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_user",
		ProductMode:      "vscode",
		Backend:          "vscode",
		ResumeInstanceID: "inst-vscode-1",
	})
	app := newRestoreHintTestApp(stateDir)

	if events := app.maybePromptDetachedVSCodeSurfacesLocked(); len(events) != 0 {
		t.Fatalf("expected group VS Code resume prompt scan to stay silent, got %#v", events)
	}
}
