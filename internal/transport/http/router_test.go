package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wyw14/cry-071/internal/application"
	"github.com/wyw14/cry-071/internal/config"
	"github.com/wyw14/cry-071/internal/platform/files"
	"github.com/wyw14/cry-071/internal/platform/identifier"
	"github.com/wyw14/cry-071/internal/platform/notification"
	"github.com/wyw14/cry-071/internal/repository/memory"
	"github.com/wyw14/cry-071/internal/service"
	httpapi "github.com/wyw14/cry-071/internal/transport/http"
	"go.uber.org/zap"
)

func testRouter(t *testing.T) http.Handler {
	t.Helper()
	now := time.Date(2026, 8, 22, 8, 0, 0, 0, time.UTC)
	store := memory.NewStoreWithDemo(now)
	tokens, _ := service.NewTokenCodec("http-test-pepper-value")
	objects, err := files.NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	services, err := application.NewServices(application.Dependencies{Transactions: store, Repositories: store, Clock: httpFixedTime{at: now}, IDs: &identifier.Sequence{}, Tokens: tokens, Notifications: notification.NewLocalSink(), Objects: objects, Redactor: service.Redactor{}, Duplicates: service.NewDuplicateMatcher(store)})
	if err != nil {
		t.Fatal(err)
	}
	return httpapi.NewRouter(httpapi.RouterDependencies{Services: services, Config: config.Config{Environment: "test", RequestTimeout: time.Second, AllowedOrigins: []string{"http://localhost:5173"}}, Logger: zap.NewNop(), Ready: func(context.Context) error { return nil }})
}

type httpFixedTime struct{ at time.Time }

func (f httpFixedTime) Now() time.Time { return f.at }

func TestSubmitAndQueryFeedbackHTTP(t *testing.T) {
	router := testRouter(t)
	body := map[string]any{"area_id": "area-central-park", "facility_category_id": "lighting", "subject_code": "safety", "priority": "high", "title": "滨河入口照明持续闪烁", "description": "入口处照明持续闪烁并影响夜间行人辨认台阶", "location": "中心公园滨河入口", "submitter_name": "周先生", "submitter_contact": "13900139000"}
	encoded, _ := json.Marshal(body)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewReader(encoded))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("submit status=%d body=%s", response.Code, response.Body.String())
	}
	var envelope struct {
		Data struct {
			QueryToken string `json:"query_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data.QueryToken == "" {
		t.Fatal("response did not contain query token")
	}
	query := httptest.NewRequest(http.MethodGet, "/api/v1/feedback/query/"+envelope.Data.QueryToken, nil)
	queryResponse := httptest.NewRecorder()
	router.ServeHTTP(queryResponse, query)
	if queryResponse.Code != http.StatusOK {
		t.Fatalf("query status=%d body=%s", queryResponse.Code, queryResponse.Body.String())
	}
}

func TestQueueRejectsUnknownSortWithStableError(t *testing.T) {
	router := testRouter(t)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/queue?sort=private_field", nil)
	request.Header.Set("X-Actor-ID", "manager")
	request.Header.Set("X-Actor-Role", "manager")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if response.Header().Get("X-Request-ID") == "" {
		t.Fatal("request id header missing")
	}
	if payload["error"] == nil {
		t.Fatal("stable error object missing")
	}
}
