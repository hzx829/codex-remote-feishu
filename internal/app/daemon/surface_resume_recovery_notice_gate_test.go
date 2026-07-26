package daemon

import (
	"context"
	"testing"
	"time"

	"github.com/kxn/codex-remote-feishu/internal/app/daemon/surfaceresume"
	"github.com/kxn/codex-remote-feishu/internal/core/control"
	"github.com/kxn/codex-remote-feishu/internal/core/eventcontract"
	"github.com/kxn/codex-remote-feishu/internal/core/orchestrator"
	relayruntime "github.com/kxn/codex-remote-feishu/internal/runtime"
)

func TestDaemonPendingHeadlessRestoreTimeoutSuppressesRepeatedNotice(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	workspaceDir := t.TempDir()
	putSurfaceResumeStateForTest(t, stateDir, surfaceresume.Entry{
		SurfaceSessionID:   "surface-1",
		GatewayID:          "app-1",
		ChatID:             "chat-1",
		ActorUserID:        "user-1",
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
	now := time.Date(2026, 7, 26, 9, 0, 0, 0, time.UTC)
	app.startHeadless = func(relayruntime.HeadlessLaunchOptions) (int, error) {
		return 4321, nil
	}
	app.onTick(context.Background(), now)

	snapshot := app.service.SurfaceSnapshot("surface-1")
	if snapshot == nil || snapshot.PendingHeadless.InstanceID == "" {
		t.Fatalf("expected initial recovery tick to create pending headless, got %#v", snapshot)
	}
	gateway := app.gateway.(*recordingGateway)
	gateway.operations = nil

	if displayCode, emit := app.recordSurfaceResumeFailureLocked("surface-1", "headless_restore_start_timeout", now.Add(time.Second)); !emit || displayCode != "headless_restore_start_timeout" {
		t.Fatalf("expected first restore timeout to be recorded as emitted, display=%q emit=%t", displayCode, emit)
	}

	app.onTick(context.Background(), now.Add(time.Minute))

	if len(gateway.operations) != 0 {
		t.Fatalf("expected repeated pending headless restore timeout to stay silent, got %#v", gateway.operations)
	}
	if snapshot := app.service.SurfaceSnapshot("surface-1"); snapshot == nil || snapshot.PendingHeadless.InstanceID != "" {
		t.Fatalf("expected expired pending headless to be consumed, got %#v", snapshot)
	}
}

func TestDaemonUngatedRestoreOutcomeGateSuppressesRepeatedConnectFailureNotice(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	putSurfaceResumeStateForTest(t, stateDir, surfaceresume.Entry{
		SurfaceSessionID:   "surface-1",
		GatewayID:          "app-1",
		ChatID:             "chat-1",
		ActorUserID:        "user-1",
		ProductMode:        "normal",
		Backend:            "codex",
		ResumeThreadID:     "thread-1",
		ResumeThreadTitle:  "修复登录流程",
		ResumeThreadCWD:    "/data/dl/droid",
		ResumeWorkspaceKey: "/data/dl/droid",
		ResumeRouteMode:    "pinned",
		ResumeHeadless:     true,
	})
	app := newRestoreHintTestApp(stateDir)
	now := time.Date(2026, 7, 26, 9, 30, 0, 0, time.UTC)
	if displayCode, emit := app.recordSurfaceResumeFailureLocked("surface-1", "headless_restore_workspace_busy", now); !emit || displayCode != "headless_restore_workspace_busy" {
		t.Fatalf("expected first workspace-busy restore failure to be recorded as emitted, display=%q emit=%t", displayCode, emit)
	}

	filtered := app.gateUngatedManagedHeadlessResumeOutcomeEventsLocked([]eventcontract.Event{
		{
			Kind:             eventcontract.KindDaemonCommand,
			SurfaceSessionID: "surface-1",
			DaemonCommand: &control.DaemonCommand{
				Kind:       control.DaemonCommandKillHeadless,
				InstanceID: "inst-headless-1",
			},
		},
		{
			Kind:             eventcontract.KindNotice,
			SurfaceSessionID: "surface-1",
			Notice:           orchestrator.NoticeForHeadlessRestoreFailure("workspace_busy"),
		},
	}, now.Add(time.Second))

	if len(filtered) != 1 || filtered[0].DaemonCommand == nil {
		t.Fatalf("expected repeated workspace-busy restore notice to be suppressed while preserving daemon command, got %#v", filtered)
	}
}
