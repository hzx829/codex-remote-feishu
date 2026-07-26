package daemon

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kxn/codex-remote-feishu/internal/app/daemon/surfaceresume"
	"github.com/kxn/codex-remote-feishu/internal/core/agentproto"
	"github.com/kxn/codex-remote-feishu/internal/core/control"
	relayruntime "github.com/kxn/codex-remote-feishu/internal/runtime"
)

func TestFeishuGroupOnDemandTextStartsHeadlessAndDefersMessage(t *testing.T) {
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
	var captured relayruntime.HeadlessLaunchOptions
	app.startHeadless = func(opts relayruntime.HeadlessLaunchOptions) (int, error) {
		captured = opts
		return 4321, nil
	}

	app.HandleAction(context.Background(), control.Action{
		Kind:             control.ActionTextMessage,
		GatewayID:        "app-1",
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		ChatID:           "oc_room",
		ActorUserID:      "ou_user",
		MessageID:        "om-on-demand-1",
		Text:             "继续刚才的任务",
	})

	snapshot := app.service.SurfaceSnapshot("feishu:app-1:chat:oc_room")
	if snapshot == nil || snapshot.PendingHeadless.InstanceID == "" {
		t.Fatalf("expected on-demand group text to start pending headless recovery, got %#v", snapshot)
	}
	if captured.InstanceID != snapshot.PendingHeadless.InstanceID || captured.WorkDir != workspaceDir {
		t.Fatalf("unexpected on-demand headless launch options: captured=%#v pending=%#v", captured, snapshot.PendingHeadless)
	}
	if len(app.service.PendingRemoteTurns()) != 0 {
		t.Fatalf("expected original message to wait for recovery before dispatch, got %#v", app.service.PendingRemoteTurns())
	}
}

func TestFeishuGroupOnDemandLaunchFailureClearsContinuationAndRepliesOnce(t *testing.T) {
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
		return 0, errors.New("boom")
	}

	app.HandleAction(context.Background(), control.Action{
		Kind:             control.ActionTextMessage,
		GatewayID:        "app-1",
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		ChatID:           "oc_room",
		ActorUserID:      "ou_user",
		MessageID:        "om-on-demand-1",
		Text:             "继续刚才的任务",
	})

	if continuation := app.surfaceResumeRuntime.groupOnDemandContinuations["feishu:app-1:chat:oc_room"]; continuation != nil {
		t.Fatalf("expected launch failure to clear continuation, got %#v", continuation)
	}
	gateway := app.gateway.(*recordingGateway)
	if len(gateway.operations) != 1 {
		t.Fatalf("expected one on-demand failure notice, got %#v", gateway.operations)
	}
	if gateway.operations[0].CardTitle != "恢复失败" {
		t.Fatalf("expected restore failure notice, got %#v", gateway.operations[0])
	}
}

func TestFeishuGroupOnDemandTimeoutClearsContinuationAndRepliesOnce(t *testing.T) {
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
		return 4321, nil
	}

	app.HandleAction(context.Background(), control.Action{
		Kind:             control.ActionTextMessage,
		GatewayID:        "app-1",
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		ChatID:           "oc_room",
		ActorUserID:      "ou_user",
		MessageID:        "om-on-demand-1",
		Text:             "继续刚才的任务",
	})
	pending := app.service.SurfaceSnapshot("feishu:app-1:chat:oc_room").PendingHeadless
	if pending.InstanceID == "" {
		t.Fatalf("expected pending headless before timeout")
	}
	gateway := app.gateway.(*recordingGateway)
	gateway.operations = nil

	app.onTick(context.Background(), pending.ExpiresAt.Add(time.Second))

	if continuation := app.surfaceResumeRuntime.groupOnDemandContinuations["feishu:app-1:chat:oc_room"]; continuation != nil {
		t.Fatalf("expected timeout to clear continuation, got %#v", continuation)
	}
	if len(gateway.operations) != 1 {
		t.Fatalf("expected one timeout notice, got %#v", gateway.operations)
	}
	if gateway.operations[0].CardTitle != "恢复失败" {
		t.Fatalf("expected restore timeout notice, got %#v", gateway.operations[0])
	}
}

func TestFeishuGroupOnDemandImmediateFailureRepliesOnce(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	putSurfaceResumeStateForTest(t, stateDir, surfaceresume.Entry{
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_user",
		ProductMode:      "normal",
		Backend:          "codex",
		ResumeThreadID:   "thread-missing",
		ResumeHeadless:   true,
	})
	app := newRestoreHintTestApp(stateDir)
	app.startHeadless = func(relayruntime.HeadlessLaunchOptions) (int, error) {
		t.Fatal("missing target must fail before headless launch")
		return 0, nil
	}

	app.HandleAction(context.Background(), control.Action{
		Kind:             control.ActionTextMessage,
		GatewayID:        "app-1",
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		ChatID:           "oc_room",
		ActorUserID:      "ou_user",
		MessageID:        "om-on-demand-missing-1",
		Text:             "继续刚才的任务",
	})

	if continuation := app.surfaceResumeRuntime.groupOnDemandContinuations["feishu:app-1:chat:oc_room"]; continuation != nil {
		t.Fatalf("expected immediate failure to avoid continuation, got %#v", continuation)
	}
	gateway := app.gateway.(*recordingGateway)
	if len(gateway.operations) != 1 {
		t.Fatalf("expected one immediate failure notice, got %#v", gateway.operations)
	}
	if gateway.operations[0].CardTitle != "恢复失败" {
		t.Fatalf("expected restore failure notice, got %#v", gateway.operations[0])
	}
}

func TestFeishuGroupOnDemandReplayDispatchesOriginalTextAfterHeadlessConnect(t *testing.T) {
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
		return 4321, nil
	}
	var sent []agentproto.Command
	app.sendAgentCommand = func(_ string, command agentproto.Command) error {
		sent = append(sent, command)
		return nil
	}

	app.HandleAction(context.Background(), control.Action{
		Kind:             control.ActionTextMessage,
		GatewayID:        "app-1",
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		ChatID:           "oc_room",
		ActorUserID:      "ou_user",
		MessageID:        "om-on-demand-1",
		Text:             "继续刚才的任务",
	})
	pending := app.service.SurfaceSnapshot("feishu:app-1:chat:oc_room").PendingHeadless
	if pending.InstanceID == "" {
		t.Fatalf("expected pending headless before replay")
	}

	app.onHello(context.Background(), agentproto.Hello{
		Instance: agentproto.InstanceHello{
			InstanceID:    pending.InstanceID,
			DisplayName:   "headless",
			WorkspaceRoot: workspaceDir,
			WorkspaceKey:  workspaceDir,
			ShortName:     "headless",
			Source:        "headless",
			Managed:       true,
			PID:           4321,
		},
	})

	prompts := make([]agentproto.Command, 0)
	for _, command := range sent {
		if command.Kind == agentproto.CommandPromptSend {
			prompts = append(prompts, command)
		}
	}
	if len(prompts) != 1 {
		t.Fatalf("expected replay to dispatch original text once, got commands=%#v prompts=%#v", sent, prompts)
	}
	if prompts[0].Origin.MessageID != "om-on-demand-1" {
		t.Fatalf("expected replayed command to keep original message id, got %#v", prompts[0].Origin)
	}
	if continuation := app.surfaceResumeRuntime.groupOnDemandContinuations["feishu:app-1:chat:oc_room"]; continuation != nil {
		t.Fatalf("expected replay to clear continuation, got %#v", continuation)
	}
}

func TestFeishuGroupOnDemandTextWithFilesReturnsRecoveryPrompt(t *testing.T) {
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
		ResumeThreadCWD:    workspaceDir,
		ResumeWorkspaceKey: workspaceDir,
		ResumeHeadless:     true,
	})
	app := newRestoreHintTestApp(stateDir)
	app.startHeadless = func(relayruntime.HeadlessLaunchOptions) (int, error) {
		t.Fatal("text with files must not start on-demand headless recovery")
		return 0, nil
	}

	app.HandleAction(context.Background(), control.Action{
		Kind:             control.ActionTextMessage,
		GatewayID:        "app-1",
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		ChatID:           "oc_room",
		ActorUserID:      "ou_user",
		MessageID:        "om-on-demand-file-1",
		Text:             "看这个文件",
		Files: []control.ActionFileAttachment{{
			SourceMessageID: "om-file-1",
			LocalPath:       "/tmp/file.txt",
			FileName:        "file.txt",
		}},
	})

	if snapshot := app.service.SurfaceSnapshot("feishu:app-1:chat:oc_room"); snapshot == nil || snapshot.PendingHeadless.InstanceID != "" {
		t.Fatalf("expected text with files to avoid pending headless, got %#v", snapshot)
	}
	if continuation := app.surfaceResumeRuntime.groupOnDemandContinuations["feishu:app-1:chat:oc_room"]; continuation != nil {
		t.Fatalf("expected no continuation for text with files, got %#v", continuation)
	}
	gateway := app.gateway.(*recordingGateway)
	if len(gateway.operations) != 1 || gateway.operations[0].CardTitle != "请先恢复群上下文" {
		t.Fatalf("expected one file recovery prompt, got %#v", gateway.operations)
	}
}

func TestFeishuGroupOnDemandVSCodeReturnsUnsupportedPrompt(t *testing.T) {
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
	app.startHeadless = func(relayruntime.HeadlessLaunchOptions) (int, error) {
		t.Fatal("VS Code group on-demand recovery must not start headless")
		return 0, nil
	}

	app.HandleAction(context.Background(), control.Action{
		Kind:             control.ActionTextMessage,
		GatewayID:        "app-1",
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		ChatID:           "oc_room",
		ActorUserID:      "ou_user",
		MessageID:        "om-on-demand-vscode-1",
		Text:             "继续刚才的任务",
	})

	if continuation := app.surfaceResumeRuntime.groupOnDemandContinuations["feishu:app-1:chat:oc_room"]; continuation != nil {
		t.Fatalf("expected no VS Code continuation, got %#v", continuation)
	}
	gateway := app.gateway.(*recordingGateway)
	if len(gateway.operations) != 1 || gateway.operations[0].CardTitle != "当前群暂不能自动恢复 VS Code" {
		t.Fatalf("expected one VS Code unsupported prompt, got %#v", gateway.operations)
	}
}
