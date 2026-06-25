package runtime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/proxy-go/proxy-go/internal/models"
	"github.com/proxy-go/proxy-go/internal/testutil"
)

func TestApplyFailureSavesFailedStatus(t *testing.T) {
	db := testutil.NewDB(t)
	svc := &Service{
		DB:      db,
		Cfg:     testutil.NewConfig(t),
		Nginx:   &fakeApplier{},
		SingBox: &fakeApplier{err: errors.New("sing-box failed")},
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
		DB:      db,
		Cfg:     testutil.NewConfig(t),
		Nginx:   &fakeApplier{name: "nginx", calls: &calls},
		SingBox: &fakeApplier{name: "sing-box", calls: &calls},
	}

	if err := svc.Apply(context.Background()); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !reflect.DeepEqual(calls, []string{"sing-box.apply", "nginx.apply"}) {
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
	singBox := &fakeApplier{}
	svc := &Service{
		DB:      db,
		Cfg:     testutil.NewConfig(t),
		Nginx:   nginx,
		SingBox: singBox,
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
	if err := svc.StartSingBox(context.Background()); err != nil {
		t.Fatalf("start sing-box: %v", err)
	}
	if err := svc.StopSingBox(context.Background()); err != nil {
		t.Fatalf("stop sing-box: %v", err)
	}
	if err := svc.RestartSingBox(context.Background()); err != nil {
		t.Fatalf("restart sing-box: %v", err)
	}

	if nginx.starts != 1 || nginx.stops != 1 || nginx.restarts != 1 {
		t.Fatalf("unexpected nginx calls: %#v", nginx)
	}
	if singBox.starts != 1 || singBox.stops != 1 || singBox.restarts != 1 {
		t.Fatalf("unexpected sing-box calls: %#v", singBox)
	}
}

func TestSingBoxLogsReturnProcessDetails(t *testing.T) {
	svc := &Service{
		Cfg:     testutil.NewConfig(t),
		SingBox: &fakeApplier{logs: []string{"stderr-detail", "process exited: signal: killed"}},
	}

	got := svc.SingBoxLogs()
	want := LogSummary{Logs: []string{"stderr-detail", "process exited: signal: killed"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected sing-box logs: got %#v want %#v", got, want)
	}
}

func TestNginxConfigReadsRenderedConfig(t *testing.T) {
	cfg := testutil.NewConfig(t)
	if err := os.MkdirAll(cfg.Paths.NginxConfDir, 0755); err != nil {
		t.Fatalf("create nginx dir: %v", err)
	}
	path := filepath.Join(cfg.Paths.NginxConfDir, "nginx.conf")
	if err := os.WriteFile(path, []byte("stream { apple.com 127.0.0.1:31001; }"), 0644); err != nil {
		t.Fatalf("write nginx config: %v", err)
	}
	svc := &Service{Cfg: cfg}

	got, err := svc.NginxConfig()
	if err != nil {
		t.Fatalf("nginx config: %v", err)
	}
	if got.Path != path || got.Content != "stream { apple.com 127.0.0.1:31001; }" {
		t.Fatalf("unexpected nginx config snapshot: %#v", got)
	}
}

func TestStatusReportsNginxPublicPortsAndSingBoxLocalInbounds(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := testutil.NewConfig(t)
	domain := models.Domain{Domain: "proxy.example.com", Status: "enabled"}
	if err := db.Create(&domain).Error; err != nil {
		t.Fatalf("create domain: %v", err)
	}
	if err := db.Create(&models.ProxyInbound{
		DomainID:   domain.ID,
		Name:       "main",
		Template:   "vless-reality-vision",
		ListenAddr: "127.0.0.1",
		ListenPort: 31001,
		RouteSNI:   "apple.com",
		Enabled:    true,
	}).Error; err != nil {
		t.Fatalf("create inbound: %v", err)
	}
	svc := &Service{
		DB:      db,
		Cfg:     cfg,
		Nginx:   &fakeApplier{},
		SingBox: &fakeApplier{},
	}

	status, err := svc.Status()
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.NginxPublicHTTPPort != 80 || status.NginxPublicHTTPSPort != 443 {
		t.Fatalf("unexpected nginx public ports: %#v", status)
	}
	if status.SingBoxInboundListen != "127.0.0.1:31001" {
		t.Fatalf("unexpected sing-box inbound listen: %#v", status)
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
