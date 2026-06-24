package runtime

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/proxy-go/proxy-go/internal/models"
	"github.com/proxy-go/proxy-go/internal/testutil"
)

func TestApplyFailureSavesFailedStatus(t *testing.T) {
	db := testutil.NewDB(t)
	svc := &Service{
		DB:    db,
		Cfg:   testutil.NewConfig(t),
		Nginx: &fakeApplier{err: errors.New("nginx failed")},
		Xray:  &fakeApplier{},
	}

	err := svc.Apply(context.Background())
	if err == nil {
		t.Fatalf("expected apply error")
	}
	var setting models.SystemSetting
	db.First(&setting, 1)
	if setting.RuntimeConfigStatus != "failed" {
		t.Fatalf("unexpected setting after failure: %#v", setting)
	}
}

func TestApplySuccessSavesSuccessStatus(t *testing.T) {
	db := testutil.NewDB(t)
	svc := &Service{
		DB:    db,
		Cfg:   testutil.NewConfig(t),
		Nginx: &fakeApplier{},
		Xray:  &fakeApplier{},
	}

	if err := svc.Apply(context.Background()); err != nil {
		t.Fatalf("apply: %v", err)
	}
	var setting models.SystemSetting
	db.First(&setting, 1)
	if setting.RuntimeConfigStatus != "success" {
		t.Fatalf("unexpected setting after success: %#v", setting)
	}
}

func TestProcessControlsDelegateToRuntimeProcesses(t *testing.T) {
	db := testutil.NewDB(t)
	nginx := &fakeApplier{}
	xray := &fakeApplier{}
	svc := &Service{
		DB:    db,
		Cfg:   testutil.NewConfig(t),
		Nginx: nginx,
		Xray:  xray,
	}

	if err := svc.StartNginx(context.Background()); err != nil {
		t.Fatalf("start nginx: %v", err)
	}
	if err := svc.StopNginx(context.Background()); err != nil {
		t.Fatalf("stop nginx: %v", err)
	}
	if err := svc.RestartNginx(context.Background()); err != nil {
		t.Fatalf("restart nginx: %v", err)
	}
	if err := svc.StartXray(context.Background()); err != nil {
		t.Fatalf("start xray: %v", err)
	}
	if err := svc.StopXray(context.Background()); err != nil {
		t.Fatalf("stop xray: %v", err)
	}
	if err := svc.RestartXray(context.Background()); err != nil {
		t.Fatalf("restart xray: %v", err)
	}

	if nginx.starts != 1 || nginx.stops != 1 || nginx.restarts != 1 {
		t.Fatalf("unexpected nginx calls: %#v", nginx)
	}
	if xray.starts != 1 || xray.stops != 1 || xray.restarts != 1 {
		t.Fatalf("unexpected xray calls: %#v", xray)
	}
}

func TestXrayLogsReturnProcessDetails(t *testing.T) {
	svc := &Service{
		Cfg:  testutil.NewConfig(t),
		Xray: &fakeApplier{logs: []string{"stderr-detail", "process exited: signal: killed"}},
	}

	got := svc.XrayLogs()
	want := LogSummary{Logs: []string{"stderr-detail", "process exited: signal: killed"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected xray logs: got %#v want %#v", got, want)
	}
}

type fakeApplier struct {
	err      error
	starts   int
	stops    int
	restarts int
	logs     []string
}

func (f *fakeApplier) Apply(ctx context.Context) error {
	return f.err
}

func (f *fakeApplier) Reload(ctx context.Context) error {
	return f.err
}

func (f *fakeApplier) Start(ctx context.Context) error {
	f.starts++
	return f.err
}

func (f *fakeApplier) Stop(ctx context.Context) error {
	f.stops++
	return f.err
}

func (f *fakeApplier) Restart(ctx context.Context) error {
	f.restarts++
	return f.err
}

func (f *fakeApplier) Status() any {
	return map[string]any{"running": true}
}

func (f *fakeApplier) Logs() []string {
	return f.logs
}
