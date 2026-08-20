package data

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"fdi_data_board/internal/conf"
)

// newMockAnthropicGateway 模拟 Anthropic Messages 流式网关:
// 依次下发 thinking 块(应被过滤)与 text 块(应逐段回调),夹杂 ping 事件
func newMockAnthropicGateway(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		frames := []string{
			`event: message_start` + "\n" + `data: {"type":"message_start","message":{}}`,
			`event: ping` + "\n" + `data: {"type":"ping"}`,
			`event: content_block_start` + "\n" + `data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}`,
			`event: content_block_delta` + "\n" + `data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"内心推理,不应下发"}}`,
			`event: content_block_stop` + "\n" + `data: {"type":"content_block_stop","index":0}`,
			`event: content_block_start` + "\n" + `data: {"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}`,
			`event: content_block_delta` + "\n" + `data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"你好"}}`,
			`event: content_block_delta` + "\n" + `data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":","}}`,
			`event: content_block_delta` + "\n" + `data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"看板"}}`,
			`event: content_block_stop` + "\n" + `data: {"type":"content_block_stop","index":1}`,
			`event: message_delta` + "\n" + `data: {"type":"message_delta","delta":{"stop_reason":"end_turn"}}`,
			`event: message_stop` + "\n" + `data: {"type":"message_stop"}`,
		}
		for _, frame := range frames {
			_, _ = w.Write([]byte(frame + "\n\n"))
			flusher.Flush()
		}
	}))
}

func newTestLlmRepo(serverURL string) *LlmRepo {
	client := anthropic.NewClient(
		option.WithBaseURL(serverURL),
		option.WithAPIKey("test-key"),
	)
	return &LlmRepo{data: &Data{
		conf: &conf.Data{Llm: &conf.Data_LLM{
			BaseUrl:   serverURL,
			ApiKey:    "test-key",
			Model:     "kimi-k3",
			MaxTokens: 1024,
		}},
		llmClient: &client,
	}}
}

func TestLlmRepoChatStreamParsesTextDeltaAndSkipsThinking(t *testing.T) {
	server := newMockAnthropicGateway(t)
	defer server.Close()

	repo := newTestLlmRepo(server.URL)
	if !repo.Enabled() {
		t.Fatal("Enabled() = false, want true")
	}

	var got []string
	if err := repo.ChatStream(context.Background(), "system", "user", func(text string) {
		got = append(got, text)
	}); err != nil {
		t.Fatalf("ChatStream error: %v", err)
	}

	want := strings.Join([]string{"你好", ",", "看板"}, "")
	if strings.Join(got, "") != want {
		t.Fatalf("deltas = %q, want joined %q", got, want)
	}
	for _, d := range got {
		if strings.Contains(d, "内心推理") {
			t.Fatalf("thinking delta leaked: %q", d)
		}
	}
}

func TestLlmRepoDisabledWhenClientMissing(t *testing.T) {
	repo := NewLlmRepo(&Data{conf: &conf.Data{}})
	if repo.Enabled() {
		t.Fatal("Enabled() = true, want false when llmClient is nil")
	}
	if err := repo.ChatStream(context.Background(), "s", "u", func(string) {}); err == nil {
		t.Fatal("ChatStream should error when disabled")
	}
}
