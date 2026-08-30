package main

// 临时复现工具:不经 HTTP/鉴权,直接调用 AiDashboardUseCase.StreamSummary,
// 区分快照构建耗时与 LLM 耗时并打印真实错误。排查 /dashboard/v1/ai/summary 500 用完即删。

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/env"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"

	dashboard_api "fdi_data_board/api/dashboard"
	"fdi_data_board/internal/biz"
	"fdi_data_board/internal/conf"
	"fdi_data_board/internal/data"
)

var (
	t0         time.Time
	llmStart   time.Time
	promptLen  int
	deltaCount int
	deltaBytes int
	thinkCount int
	thinkBytes int
)

func main() {
	confPath := os.Args[1]
	if confPath == "" {
		confPath = "configs/config-dev.yaml"
	}
	t0 = time.Now()

	c := config.New(config.WithSource(file.NewSource(confPath), env.NewSource()))
	if err := c.Load(); err != nil {
		panic(err)
	}
	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp, "caller", log.DefaultCaller)

	dataData, cleanup, err := data.NewData(bc.Data, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	foUc := biz.NewFoDashboardUseCase(data.NewFoDashboardRepo(dataData), data.NewRunningFilterNameResolver(dataData))
	doUc := biz.NewDoDashboardUseCase(data.NewDoDashboardRepo(dataData))
	llmRepo := data.NewLlmRepo(dataData)
	timedLlm := &timedLlmRepo{inner: llmRepo}
	aiUc := biz.NewAiDashboardUseCase(foUc, doUc, timedLlm)

	fmt.Printf("LLM enabled: %v\n", llmRepo.Enabled())

	req := &dashboard_api.AiSummaryRequest{
		StartDt: "2026-06-01",
		EndDt:   "2026-06-03",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 360*time.Second) // 与生产 LLM_TIMEOUT 对齐
	defer cancel()

	start := time.Now()
	err = aiUc.StreamSummary(ctx, req, func(kind, text string) error {
		if kind == "thinking" {
			thinkCount++
			thinkBytes += len(text)
			return nil
		}
		deltaCount++
		deltaBytes += len(text)
		return nil
	})
	fmt.Printf("\n=== 总耗时: %s, thinking 段数: %d(%d字节), delta 段数: %d, 输出字节数: %d\n",
		time.Since(start), thinkCount, thinkBytes, deltaCount, deltaBytes)
	if !llmStart.IsZero() {
		fmt.Printf("=== 快照构建耗时: %s, LLM 阶段耗时: %s, prompt 长度: %d bytes\n",
			llmStart.Sub(t0), time.Since(llmStart), promptLen)
	} else {
		fmt.Printf("=== LLM 从未被调用, 前置阶段耗时: %s\n", time.Since(t0))
	}
	if err != nil {
		fmt.Printf("=== 真实错误: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("=== 成功")
}

// timedLlmRepo 包装 biz.LlmRepo,记录调用时机、prompt 长度与耗时
type timedLlmRepo struct {
	inner biz.LlmRepo
}

func (t *timedLlmRepo) Enabled() bool    { return t.inner.Enabled() }
func (t *timedLlmRepo) Model() string    { return t.inner.Model() }
func (t *timedLlmRepo) MaxTokens() int64 { return t.inner.MaxTokens() }

func (t *timedLlmRepo) ChatStreamEx(ctx context.Context, params anthropic.MessageNewParams, onEvent func(anthropic.MessageStreamEventUnion)) (*anthropic.Message, error) {
	return t.inner.ChatStreamEx(ctx, params, onEvent)
}

func (t *timedLlmRepo) ChatStream(ctx context.Context, systemPrompt, userPrompt string, onDelta func(string), onThinking func(string)) error {
	llmStart = time.Now()
	promptLen = len(userPrompt)
	_ = os.WriteFile("/tmp/ai_system_prompt.txt", []byte(systemPrompt), 0644)
	_ = os.WriteFile("/tmp/ai_user_prompt.txt", []byte(userPrompt), 0644)
	err := t.inner.ChatStream(ctx, systemPrompt, userPrompt, onDelta, onThinking)
	fmt.Printf(">>> ChatStream 返回: err=%v, 耗时=%s, prompt=%d bytes, thinking=%d段/%d字节, delta=%d段/%d字节\n",
		err, time.Since(llmStart), promptLen, thinkCount, thinkBytes, deltaCount, deltaBytes)
	return err
}
