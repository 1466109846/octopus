package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/gin-gonic/gin"
)

// TestCreateScheduledProbeRejectsInvalidKind 创建接口必须在 API 处拒绝非空未知种类。
//
// 这是"未知种类静默回落成 monitor"这条旧行为的最后一道闸暴露在 HTTP 上的形态:
// 前端若传来一个写错的 kind, 应当拿到 400 而不是一条被偷偷改成监控的任务。
//
// op.validateScheduledProbe 把种类校验排在最前(先于渠道/模型存在性检查), 因此这条拒绝路径
// 不依赖任何缓存或数据库, 直接由 API 返回 400 —— 正好把断言钉在"种类"这一道。
// 这里把处理器直接挂到不带鉴权中间件的引擎上, 让断言纯粹落在种类校验这一道,
// 不被鉴权中间件挡在前面(见 log_test.go 的同样做法)。
func TestCreateScheduledProbeRejectsInvalidKind(t *testing.T) {
	engine := gin.New()
	engine.POST("/api/v1/scheduled-probe/create", createScheduledProbe)

	payload, err := json.Marshal(model.ScheduledProbeRequest{
		Name:            "p",
		Kind:            "bogus",
		Targets:         []model.ScheduledProbeTargetRequest{{ChannelID: 2, ModelName: "m-a"}},
		IntervalMinutes: 10,
	})
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scheduled-probe/create", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid kind: status=%d want=%d (body=%s)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

// TestCreateScheduledProbeRejectsInvalidKindForKeepalive 保活种类走的是 /create 的同一份提交校验,
// 一个写错的 kind(如 liveness)同样要在 API 处被拒绝, 不会静默落到保活调度。
func TestCreateScheduledProbeRejectsInvalidKeepaliveKind(t *testing.T) {
	engine := gin.New()
	engine.POST("/api/v1/scheduled-probe/create", createScheduledProbe)

	payload, err := json.Marshal(model.ScheduledProbeRequest{
		Name:            "p",
		Kind:            "liveness",
		Targets:         []model.ScheduledProbeTargetRequest{{ChannelID: 2, ModelName: "m-a"}},
		IntervalMinutes: 10,
	})
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scheduled-probe/create", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid keepalive kind: status=%d want=%d (body=%s)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
