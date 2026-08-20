package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	dashboard_api "fdi_data_board/api/dashboard"
)

// LlmRepo 大模型流式网关(data 层实现,Anthropic Messages 协议)
type LlmRepo interface {
	Enabled() bool
	ChatStream(ctx context.Context, systemPrompt, userPrompt string, onDelta func(string)) error
}

// AiDashboardUseCase 看板 AI 总结用例:复用 FO/DO 用例的聚合结果(口径与页面完全一致),
// 拼装快照后交给大模型生成解读;未配置 LLM 网关时退化为本地统计模式。
type AiDashboardUseCase struct {
	fo  *FoDashboardUseCase
	do  *DoDashboardUseCase
	llm LlmRepo
}

func NewAiDashboardUseCase(fo *FoDashboardUseCase, do *DoDashboardUseCase, llm LlmRepo) *AiDashboardUseCase {
	return &AiDashboardUseCase{fo: fo, do: do, llm: llm}
}

// aiDashboardSnapshot 交给大模型的数据快照(全部来自现有聚合接口,失败的分项置空跳过)
type aiDashboardSnapshot struct {
	QueryRange      *queryRange               `json:"query_range,omitempty"`
	FffOverview     *dashboard_api.FoFffOverviewResponse   `json:"fff_overview,omitempty"`
	RunningOverview *dashboard_api.FoRunningOverviewResponse `json:"running_overview,omitempty"`
	FffFailReason   []*dashboard_api.DoFailReasonItem       `json:"fff_fail_reason,omitempty"`
	CloseReason     []*dashboard_api.CloseReasonItem        `json:"close_reason,omitempty"`
	StageTrend      *dashboard_api.StageTrendResponse      `json:"stage_trend,omitempty"`
	MemTop          []*dashboard_api.DoEventTopItem         `json:"mem_top,omitempty"`
	DiskTop         []*dashboard_api.DoEventTopItem         `json:"disk_top,omitempty"`
	QuotaTop        []*dashboard_api.DoEventTopItem         `json:"quota_top,omitempty"`
	FdrQuality      *dashboard_api.DoFdrQualityResponse     `json:"fdr_quality,omitempty"`
	FclQuality      *dashboard_api.DoFclQualityResponse     `json:"fcl_quality,omitempty"`
}

type queryRange struct {
	StartDt     string   `json:"start_dt,omitempty"`
	EndDt       string   `json:"end_dt,omitempty"`
	EventNames  []string `json:"event_names,omitempty"`
	ProjectName string   `json:"project_name,omitempty"`
	CarTypes    []string `json:"car_types,omitempty"`
}

// buildSnapshot 拉取专项分析所需的全部聚合数据,单项失败不阻断整体
func (uc *AiDashboardUseCase) buildSnapshot(ctx context.Context, req *dashboard_api.AiSummaryRequest) *aiDashboardSnapshot {
	snap := &aiDashboardSnapshot{
		QueryRange: &queryRange{
			StartDt:     req.StartDt,
			EndDt:       req.EndDt,
			EventNames:  splitEventNames(req.EventNames),
			ProjectName: req.ProjectName,
			CarTypes:    splitEventNames(req.CarTypes),
		},
	}

	triggerReq := &dashboard_api.FffTriggerRequest{
		FilterName: req.FilterName, EventNames: req.EventNames, ProjectName: req.ProjectName,
		CarTypes: req.CarTypes, StartDt: req.StartDt, EndDt: req.EndDt,
	}
	runningReq := &dashboard_api.FffRunningRequest{
		FilterName: req.FilterName, EventNames: req.EventNames, ProjectName: req.ProjectName,
		CarTypes: req.CarTypes, StartDt: req.StartDt, EndDt: req.EndDt,
	}
	closeReq := &dashboard_api.CloseReasonRequest{
		FilterName: req.FilterName, EventNames: req.EventNames, ProjectName: req.ProjectName,
		CarTypes: req.CarTypes, StartDt: req.StartDt, EndDt: req.EndDt,
	}
	trendReq := &dashboard_api.StageTrendRequest{
		FilterName: req.FilterName, EventNames: req.EventNames, ProjectName: req.ProjectName,
		CarTypes: req.CarTypes, StartDt: req.StartDt, EndDt: req.EndDt,
	}
	doReq := &dashboard_api.DoCoolTopRequest{
		FilterName: req.FilterName, EventNames: req.EventNames, ProjectName: req.ProjectName,
		CarTypes: req.CarTypes, StartDt: req.StartDt, EndDt: req.EndDt,
	}

	if v, err := uc.fo.GetFffOverview(ctx, triggerReq); err == nil {
		snap.FffOverview = v
	}
	if v, err := uc.fo.GetRunningOverview(ctx, runningReq); err == nil {
		snap.RunningOverview = v
	}
	if v, err := uc.fo.GetFffFailReason(ctx, triggerReq); err == nil {
		snap.FffFailReason = v.List
	}
	if v, err := uc.fo.GetCloseReason(ctx, closeReq); err == nil {
		snap.CloseReason = v.List
	}
	if v, err := uc.fo.GetStageTrend(ctx, trendReq); err == nil {
		snap.StageTrend = v
	}
	if v, err := uc.do.GetMemTop(ctx, doReq); err == nil {
		snap.MemTop = v.List
	}
	if v, err := uc.do.GetDiskTop(ctx, doReq); err == nil {
		snap.DiskTop = v.List
	}
	if v, err := uc.do.GetQuotaTop(ctx, doReq); err == nil {
		snap.QuotaTop = v.List
	}
	if v, err := uc.do.GetFdrQuality(ctx, doReq); err == nil {
		snap.FdrQuality = v
	}
	if v, err := uc.do.GetFclQuality(ctx, doReq); err == nil {
		snap.FclQuality = v
	}
	return snap
}

// StreamSummary 生成流式总结:emit 被逐段调用(LLM 增量或本地分段),返回错误表示整体失败
func (uc *AiDashboardUseCase) StreamSummary(ctx context.Context, req *dashboard_api.AiSummaryRequest, emit func(string) error) error {
	snap := uc.buildSnapshot(ctx, req)

	if uc.llm.Enabled() {
		userPrompt, err := buildAiUserPrompt(snap)
		if err != nil {
			return err
		}
		return uc.llm.ChatStream(ctx, AiSystemPrompt, userPrompt, func(delta string) {
			_ = emit(delta)
		})
	}

	// 本地统计模式:未配置 LLM 网关,输出确定性统计摘要(内容与页面口径一致)
	for _, chunk := range strings.SplitAfter(localSummary(snap), "\n") {
		if err := emit(chunk); err != nil {
			return err
		}
		time.Sleep(15 * time.Millisecond) // 轻微节流,前端有流式体验
	}
	return nil
}

// AiSystemPrompt 系统提示词:角色 + 口径词典 + 输出纪律(数字必须来自给定数据)
const AiSystemPrompt = `你是数采链路看板的资深数据分析师。用户会给你一段看板数据快照(JSON),请生成中文 markdown 分析总结。

## 业务口径(必须遵守)
- 链路三阶段:FFF=筛选器触发、FDR=落盘(录制数据写盘)、FCL=上传(云上传)。
- 三阶段为独立口径:各阶段分别统计成功/失败,后一阶段不以前一阶段成功为门槛;不要把三阶段数字相加或相除得出"全链路成功率"。
- FFF 失败原因 cooldown 表示冷却丢弃(短时间重复触发被丢弃),不是故障。
- success_rate 等比率字段已是百分数;P95 字段是分位值,不要再聚合。
- 数据中的 list 类字段是 Top 榜单,不代表全量。

## 输出要求
1. 所有数字必须严格来自给定快照,禁止编造、推算或引入快照外的数字;快照缺失的分项直接跳过,不要臆测。
2. 结构:## 总体概览 → ## FFF 触发 → ## FDR 落盘 → ## FCL 上传 → ## 建议关注。每节 2-5 条要点,引用具体数值与占比。
3. 亮点用 ✅,风险用 ⚠️,需要人工确认的用 🔍。
4. 末尾加一行:> 以上解读由 AI 生成,数字来自看板聚合数据,口径以看板为准。
5. 语言精炼,不要输出与数据无关的寒暄。`

func buildAiUserPrompt(snap *aiDashboardSnapshot) (string, error) {
	raw, err := json.Marshal(snap)
	if err != nil {
		return "", fmt.Errorf("marshal snapshot: %w", err)
	}
	return "以下是当前查询范围的看板数据快照(JSON),请按系统要求输出分析总结:\n" + string(raw), nil
}

// localSummary 本地确定性摘要(LLM 未配置时的兜底,口径与页面一致)
func localSummary(snap *aiDashboardSnapshot) string {
	var b strings.Builder
	b.WriteString("> 统计模式(未配置 LLM 网关,以下为规则生成的摘要)\n\n")

	if r := snap.QueryRange; r != nil {
		b.WriteString(fmt.Sprintf("- 查询范围:%s ~ %s", orDash(r.StartDt), orDash(r.EndDt)))
		if len(r.EventNames) > 0 {
			b.WriteString(",事件:" + strings.Join(r.EventNames, ","))
		}
		if r.ProjectName != "" {
			b.WriteString(",项目:" + r.ProjectName)
		}
		if len(r.CarTypes) > 0 {
			b.WriteString(",车型:" + strings.Join(r.CarTypes, ","))
		}
		b.WriteString("\n\n")
	}

	if o := snap.FffOverview; o != nil {
		b.WriteString("## FFF 触发\n")
		b.WriteString(fmt.Sprintf("- 触发总数 %s,成功率 %.1f%%,失败 %s\n",
			fmtInt(o.TriggerTotal), o.TriggerSuccessRate, fmtInt(o.TriggerFailed)))
		if len(snap.FffFailReason) > 0 {
			var total int64
			for _, it := range snap.FffFailReason {
				total += it.Value
			}
			b.WriteString("- 失败原因 Top:")
			for i, it := range snap.FffFailReason {
				if i >= 3 {
					break
				}
				b.WriteString(fmt.Sprintf(" %s(%s, %.1f%%)", strings.TrimPrefix(it.Name, "FFF-"), fmtInt(it.Value), pct(it.Value, total)))
			}
			b.WriteString("\n")
		}
	}

	if q := snap.FdrQuality; q != nil {
		b.WriteString("\n## FDR 落盘\n")
		b.WriteString(fmt.Sprintf("- 落盘 %s,成功 %s;耗时 P95 %.0fms,磁盘 P95 %.0fMB\n",
			fmtInt(q.FdrTotal), fmtInt(q.FdrSuccess), q.TimeCostMsP95, q.TdMbP95))
		if len(snap.MemTop) > 0 {
			b.WriteString(fmt.Sprintf("- 内存不足 Top1:%s(%s 次)\n", snap.MemTop[0].EventName, fmtInt(snap.MemTop[0].Count)))
		}
		if len(snap.DiskTop) > 0 {
			b.WriteString(fmt.Sprintf("- 磁盘不足 Top1:%s(%s 次)\n", snap.DiskTop[0].EventName, fmtInt(snap.DiskTop[0].Count)))
		}
	}

	if q := snap.FclQuality; q != nil {
		b.WriteString("\n## FCL 上传\n")
		b.WriteString(fmt.Sprintf("- 上传总数 %s,Bag 大小 P95 %.2f GB\n", fmtInt(q.UploadTotal), q.BagSizeP95/1024/1024/1024))
		if len(snap.QuotaTop) > 0 {
			b.WriteString(fmt.Sprintf("- Quota 超限 Top1:%s(%s 次)\n", snap.QuotaTop[0].EventName, fmtInt(snap.QuotaTop[0].Count)))
		}
	}

	b.WriteString("\n> 以上数字来自看板聚合数据,口径以看板为准\n")
	return b.String()
}

func fmtInt(v int64) string {
	s := fmt.Sprintf("%d", v)
	if len(s) <= 4 {
		return s
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	return strings.Join(parts, ",")
}

func pct(num, den int64) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den) * 100
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
