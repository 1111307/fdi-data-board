package biz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-kratos/kratos/v2/log"

	"fdi_data_board/internal/conf"
)

const defaultWebhookURL = "https://open.feishu.cn/open-apis/bot/v2/hook/c3f49363-67bb-421f-966c-a7bcf132c104"

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

		if alert.ValueString != "" {
			rows = append(rows, []feishuEl{{Tag: "text", Text: "Value: " + alert.ValueString}})
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

		if alert.StartsAt != "" {
			rows = append(rows, []feishuEl{{Tag: "text", Text: "触发时间：" + alert.StartsAt}})
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
