package biz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	"fdi_data_board/internal/conf"
)

const defaultWebhookURL = "https://open.feishu.cn/open-apis/bot/v2/hook/c3f49363-67bb-421f-966c-a7bcf132c104"

var shanghaiLoc = func() *time.Location {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	return loc
}()

// GrafanaAlertPayload Grafana webhook 告警结构（兼容 Grafana 9+ unified alerting 格式）
type GrafanaAlertPayload struct {
	Title   string        `json:"title"`
	Message string        `json:"message"`
	State   string        `json:"state"`
	Alerts  []AlertDetail `json:"alerts"`
}

type AlertDetail struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     string            `json:"startsAt"`
	EndsAt       string            `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	SilenceURL   string            `json:"silenceURL"`
	DashboardURL string            `json:"dashboardURL"`
	PanelURL     string            `json:"panelURL"`
	ValueString  string            `json:"valueString"`
}

// AlertUseCase 告警业务逻辑
type AlertUseCase struct {
	data *conf.Data
}

func NewAlertUseCase(data *conf.Data) *AlertUseCase {
	return &AlertUseCase{data: data}
}

func (uc *AlertUseCase) webhookURL() string {
	if uc.data.Feishu != nil && uc.data.Feishu.WebhookUrl != "" {
		return uc.data.Feishu.WebhookUrl
	}
	return defaultWebhookURL
}

// feishuWebhookPayload 飞书自定义机器人 webhook 消息格式
type feishuWebhookPayload struct {
	MsgType string            `json:"msg_type"`
	Content feishuPostContent `json:"content"`
}

type feishuPostContent struct {
	Post feishuPostLang `json:"post"`
}

type feishuPostLang struct {
	ZhCn feishuPostBody `json:"zh_cn"`
}

type feishuPostBody struct {
	Title   string       `json:"title"`
	Content [][]feishuEl `json:"content"`
}

type feishuEl struct {
	Tag  string `json:"tag"`
	Text string `json:"text,omitempty"`
}

// SendGrafanaAlert 将 Grafana 告警通过自定义 webhook 机器人发送到飞书群
func (uc *AlertUseCase) SendGrafanaAlert(ctx context.Context, payload *GrafanaAlertPayload) error {
	webhookURL := uc.webhookURL()
	log.Infof("SendGrafanaAlert: url=%q title=%q", webhookURL, payload.Title)

	body := uc.buildWebhookPayload(payload)
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal feishu payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send feishu webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("feishu webhook returned status %d", resp.StatusCode)
	}
	log.Infof("SendGrafanaAlert: sent successfully")
	return nil
}

func (uc *AlertUseCase) buildWebhookPayload(payload *GrafanaAlertPayload) feishuWebhookPayload {
	stateIcon := "🔴"
	if strings.ToLower(payload.State) == "ok" || strings.ToLower(payload.State) == "resolved" {
		stateIcon = "🟢"
	}

	var rows [][]feishuEl

	rows = append(rows, []feishuEl{{Tag: "text", Text: fmt.Sprintf("状态：%s %s", stateIcon, payload.State)}})

	if payload.Message != "" {
		rows = append(rows, []feishuEl{{Tag: "text", Text: fmt.Sprintf("描述：%s", payload.Message)}})
	}

	for i, alert := range payload.Alerts {
		if i >= 5 {
			rows = append(rows, []feishuEl{{Tag: "text", Text: fmt.Sprintf("... 共 %d 条告警", len(payload.Alerts))}})
			break
		}
		alertName := alert.Labels["alertname"]
		if alertName == "" {
			alertName = fmt.Sprintf("Alert #%d", i+1)
		}

		rows = append(rows, []feishuEl{{Tag: "text", Text: fmt.Sprintf("[%s] %s", alertName, alert.Status)}})

		// 如果 message 不为空，说明 Grafana 已经把详情都带上了，不用再重复输出各字段
		if payload.Message == "" {
			if alert.ValueString != "" {
				rows = append(rows, []feishuEl{{Tag: "text", Text: "Value:"}})
				for _, line := range formatValueString(alert.ValueString) {
					rows = append(rows, []feishuEl{{Tag: "text", Text: "  " + line}})
				}
			}
			if len(alert.Labels) > 0 {
				rows = append(rows, []feishuEl{{Tag: "text", Text: "Labels:"}})
				for k, v := range alert.Labels {
					rows = append(rows, []feishuEl{{Tag: "text", Text: fmt.Sprintf(" - %s = %s", k, v)}})
				}
			}
			if len(alert.Annotations) > 0 {
				rows = append(rows, []feishuEl{{Tag: "text", Text: "Annotations:"}})
				for k, v := range alert.Annotations {
					rows = append(rows, []feishuEl{{Tag: "text", Text: fmt.Sprintf(" - %s = %s", k, v)}})
				}
			}
			// formatStartsAt 将 UTC 时间转为 Asia/Shanghai
			if alert.StartsAt != "" {
				rows = append(rows, []feishuEl{{Tag: "text", Text: "触发时间：" + formatStartsAt(alert.StartsAt)}})
			}
			if alert.GeneratorURL != "" {
				rows = append(rows, []feishuEl{{Tag: "text", Text: "Source: " + alert.GeneratorURL}})
			}
			if alert.SilenceURL != "" {
				rows = append(rows, []feishuEl{{Tag: "text", Text: "Silence: " + alert.SilenceURL}})
			}
			if alert.DashboardURL != "" {
				rows = append(rows, []feishuEl{{Tag: "text", Text: "Dashboard: " + alert.DashboardURL}})
			}
			if alert.PanelURL != "" {
				rows = append(rows, []feishuEl{{Tag: "text", Text: "Panel: " + alert.PanelURL}})
			}
		} else {
			if alert.StartsAt != "" {
				rows = append(rows, []feishuEl{{Tag: "text", Text: "触发时间：" + formatStartsAt(alert.StartsAt)}})
			}
		}
	}

	return feishuWebhookPayload{
		MsgType: "post",
		Content: feishuPostContent{
			Post: feishuPostLang{
				ZhCn: feishuPostBody{
					Title:   fmt.Sprintf("[Grafana告警] %s", payload.Title),
					Content: rows,
				},
			},
		},
	}
}

// formatStartsAt 将 Grafana UTC 时间字符串转为 Asia/Shanghai 本地时间
func formatStartsAt(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts // 解析失败原样返回
	}
	return t.In(shanghaiLoc).Format("2006-01-02 15:04:05")
}

var valueRegex = regexp.MustCompile(`\{([^}]*)\}\s+value=([\d.eE+-]+)`)

// formatValueString 将 Grafana 原始 valueString 拆成可读的行
// 输入: [ var='B0' metric='Value' labels={key1=val1, key2=val2} value=123 ], [...]
// 输出: ["pod=data-collection-normal-788b67c849-f7stb, value=2", ...]
func formatValueString(raw string) []string {
	matches := valueRegex.FindAllStringSubmatch(raw, -1)
	if len(matches) == 0 {
		return []string{raw} // 正则匹配失败时原样返回
	}

	var lines []string
	for _, m := range matches {
		labelsStr := m[1] // key1=val1, key2=val2
		value := m[2]

		parts := strings.Split(labelsStr, ",")
		podName := ""
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(p, "pod=") {
				podName = p
				break
			}
		}
		if podName == "" {
			// 没有 pod 字段时取第一个非空的 label
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					podName = p
					break
				}
			}
		}
		lines = append(lines, podName+", value="+value)
	}
	return lines
}
