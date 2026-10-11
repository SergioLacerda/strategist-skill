package telemetry

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"

	collectorlogs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
	"google.golang.org/grpc"
)

type logsReceiver struct {
	collectorlogs.UnimplementedLogsServiceServer
	mu   sync.Mutex
	reqs []*collectorlogs.ExportLogsServiceRequest
}

func (r *logsReceiver) Export(_ context.Context, req *collectorlogs.ExportLogsServiceRequest) (*collectorlogs.ExportLogsServiceResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reqs = append(r.reqs, req)
	return &collectorlogs.ExportLogsServiceResponse{}, nil
}

func (r *logsReceiver) attributes() map[string]*commonpb.AnyValue {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := map[string]*commonpb.AnyValue{}
	for _, req := range r.reqs {
		for _, rec := range logRecords(req) {
			for _, kv := range rec.GetAttributes() {
				out[kv.GetKey()] = kv.GetValue()
			}
		}
	}
	return out
}

func logRecords(req *collectorlogs.ExportLogsServiceRequest) []*logspb.LogRecord {
	var out []*logspb.LogRecord
	for _, rl := range req.GetResourceLogs() {
		for _, sl := range rl.GetScopeLogs() {
			out = append(out, sl.GetLogRecords()...)
		}
	}
	return out
}

// TestInit_ExportsSlogRecordsOverOTLPLogs speaks the real OTLP/gRPC protocol to
// an in-process receiver. Before the log pipeline existed, the slog bridge
// wrote to a no-op provider and every exported mission event was dropped.
func TestInit_ExportsSlogRecordsOverOTLPLogs(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer()
	recv := &logsReceiver{}
	collectorlogs.RegisterLogsServiceServer(srv, recv)
	go func() { _ = srv.Serve(lis) }() //nolint:errcheck // stopped by GracefulStop below
	t.Cleanup(srv.Stop)

	shutdown, err := Init(Config{Endpoint: lis.Addr().String(), ServiceName: "strategist-test", Insecure: true})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	slog.Info("usage recorded", "event.name", EventUsageRecorded, AttrUsageTokensInput, int64(1200))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := shutdown(ctx); err != nil {
		t.Fatalf("shutdown (flush): %v", err)
	}

	got := recv.attributes()
	name, ok := got["event.name"]
	if !ok || name.GetStringValue() != EventUsageRecorded {
		t.Fatalf("the receiver did not get event.name=%q; attributes: %v", EventUsageRecorded, got)
	}
	if v, ok := got[AttrUsageTokensInput]; !ok || v.GetIntValue() != 1200 {
		t.Fatalf("the token count did not arrive as an integer attribute: %v", got[AttrUsageTokensInput])
	}
}
