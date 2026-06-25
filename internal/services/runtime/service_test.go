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
		Nginx: &fakeApplier{},
		Xray:  &fakeApplier{err: errors.New("xray failed")},
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
	var calls []string
	svc := &Service{
		DB:    db,
		Cfg:   testutil.NewConfig(t),
		Nginx: &fakeApplier{name: "nginx", calls: &calls},
		Xray:  &fakeApplier{name: "xray", calls: &calls},
	}

	if err := svc.Apply(context.Background()); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !reflect.DeepEqual(calls, []string{"xray.apply", "nginx.apply"}) {
		t.Fatalf("unexpected apply order: %#v", calls)
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

func TestStatusReportsNginxPublicPortsAndXrayLocalInbound(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := testutil.NewConfig(t)
	domain := models.Domain{Domain: "proxy.example.com", Status: "enabled"}
	if err := db.Create(&domain).Error; err != nil {
		t.Fatalf("create domain: %v", err)
	}
	if err := db.Create(&models.ProxyInbound{
		DomainID:   domain.ID,
		Name:       "main",
		Template:   "vless-xhttp",
		ListenAddr: "127.0.0.1",
		ListenPort: 31001,
		Enabled:    true,
	}).Error; err != nil {
		t.Fatalf("create inbound: %v", err)
	}
	svc := &Service{
		DB:    db,
		Cfg:   cfg,
		Nginx: &fakeApplier{},
		Xray:  &fakeApplier{},
	}

	status, err := svc.Status()
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.NginxPublicHTTPPort != 80 || status.NginxPublicHTTPSPort != 443 {
		t.Fatalf("unexpected nginx public ports: %#v", status)
	}
	if status.XrayInboundListen != "127.0.0.1:31001" {
		t.Fatalf("unexpected xray inbound listen: %#v", status)
	}
}

type fakeApplier struct {
	err      error
	name     string
	calls    *[]string
	starts   int
	stops    int
	restarts int
	logs     []string
}

func (f *fakeApplier) Apply(ctx context.Context) error {
	f.record("apply")
	return f.err
}

func (f *fakeApplier) Reload(ctx context.Context) error {
	return f.err
}

func (f *fakeApplier) Start(ctx context.Context) error {
	f.record("start")
	f.starts++
	return f.err
}

func (f *fakeApplier) Stop(ctx context.Context) error {
	f.record("stop")
	f.stops++
	return f.err
}

func (f *fakeApplier) Restart(ctx context.Context) error {
	f.record("restart")
	f.restarts++
	return f.err
}

func (f *fakeApplier) record(action string) {
	if f.calls == nil || f.name == "" {
		return
	}
	*f.calls = append(*f.calls, f.name+"."+action)
}

func (f *fakeApplier) Status() any {
	return map[string]any{"running": true}
}

func (f *fakeApplier) Logs() []string {
	return f.logs
}
