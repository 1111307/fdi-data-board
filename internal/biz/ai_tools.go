package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/param"

	dashboard_api "fdi_data_board/api/dashboard"
)

// aiToolDef 工具定义:Anthropic tool schema + 进程内执行适配器(直调 biz 用例,无 HTTP)
type aiToolDef struct {
	def  anthropic.ToolParam
	exec func(ctx context.Context, args map[string]any) (string, error)
}

// aiToolRegistry 白名单工具注册表;clarify 为保留工具(不执行,触发人工确认暂停)
type aiToolRegistry struct {
	tools map[string]*aiToolDef
	order []string
}

const aiClarifyToolName = "clarify"
const aiSubmitPlanToolName = "submit_plan"

func newAiToolRegistry(fo *FoDashboardUseCase, do *DoDashboardUseCase) *aiToolRegistry {
	r := &aiToolRegistry{tools: map[string]*aiToolDef{}}

	add := func(def anthropic.ToolParam, exec func(context.Context, map[string]any) (string, error)) {
		r.tools[def.Name] = &aiToolDef{def: def, exec: exec}
		r.order = append(r.order, def.Name)
	}

	// ---- 保留工具:clarify(参数/意图不确定时向用户求证) ----
	add(anthropic.ToolParam{
		Name:        aiClarifyToolName,
		Description: param.NewOpt("当无法确定调用哪个工具、或关键参数(如 stage/kind/日期范围/事件)缺失或有歧义时,调用此工具向用户提问。payload.kind: missing_param(缺参数,含 param 名与 candidates 候选)/ambiguous_tool(工具歧义,含 options 候选与各自理由)/confirm_params(参数齐了,展示最终参数让用户确认)。question 为给用户看的一句中文提问。"),
		InputSchema: aiSchema(map[string]any{
			"kind":       map[string]any{"type": "string", "enum": []string{"missing_param", "ambiguous_tool", "confirm_params"}, "description": "确认类型"},
			"question":   map[string]any{"type": "string", "description": "向用户提出的问题,一句话"},
			"param":      map[string]any{"type": "string", "description": "kind=missing_param 时缺的参数名"},
			"candidates": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "kind=missing_param 时的候选值"},
			"options": map[string]any{"type": "array", "items": map[string]any{
				"type": "object", "properties": map[string]any{
					"tool":   map[string]any{"type": "string"},
					"reason": map[string]any{"type": "string"},
				},
			}, "description": "kind=ambiguous_tool 时的候选工具与理由"},
			"args": map[string]any{"type": "object", "description": "kind=confirm_params 时的最终参数预览"},
		}, "kind", "question"),
	}, nil)

	// ---- 保留工具:submit_plan(计划外显,复杂问题先出查询计划让用户审查) ----
	add(anthropic.ToolParam{
		Name:        aiSubmitPlanToolName,
		Description: param.NewOpt("当问题需要【两步及以上】数据查询(对比、多阶段、多维度交叉)时,必须先调用此工具提交查询计划,经用户确认后才能开始执行。简单单步查询(只调一个工具即可回答)无需计划,直接调数据工具。steps 每步含:tool(要调的工具名)、args(该步参数)、purpose(这一步查什么、为什么需要,中文一句话)。summary 用中文一句话概括整个计划的思路与口径(时间范围/阶段/对比方式)。用户确认后按 steps 顺序执行;用户拒绝时你会收到反馈,据此重新规划。"),
		InputSchema: aiSchema(map[string]any{
			"summary": map[string]any{"type": "string", "description": "计划思路一句话:查什么、什么口径、怎么得出结论"},
			"steps": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"tool":    map[string]any{"type": "string", "description": "工具名,如 get_fail_reason"},
						"args":    map[string]any{"type": "object", "description": "该步参数"},
						"purpose": map[string]any{"type": "string", "description": "这一步的目的,中文"},
					},
					"required": []string{"tool", "args", "purpose"},
				},
				"description": "按执行顺序排列的查询步骤"},
		}, "summary", "steps"),
	}, nil)

	// ---- 数据工具:全部直调既有用例,口径与看板页面一致 ----
	commonProps := func() map[string]any {
		return map[string]any{
			"start_dt":     map[string]any{"type": "string", "description": "开始日期 YYYY-MM-DD,缺省为近7天"},
			"end_dt":       map[string]any{"type": "string", "description": "结束日期 YYYY-MM-DD,缺省为近7天"},
			"event_names":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "事件名过滤,用户点名才传"},
			"project_name": map[string]any{"type": "string", "description": "项目名过滤"},
			"car_types":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "车型过滤"},
		}
	}

	add(anthropic.ToolParam{
		Name:        "get_fail_reason",
		Description: param.NewOpt("查询某阶段的失败原因分布(低基数 detail_tag 聚合,Top 榜单)。适用:失败原因/为什么失败/失败分布。stage=fff 查筛选器触发失败,fdr 查落盘失败,fcl 查上传失败。"),
		InputSchema: aiSchemaWith(commonProps(), map[string]any{
			"stage": map[string]any{"type": "string", "enum": []string{"fff", "fdr", "fcl"}, "description": "链路阶段,必填"},
		}, "stage"),
	}, execStageQuery(fo, do, "fail_reason"))

	add(anthropic.ToolParam{
		Name:        "get_stage_trend",
		Description: param.NewOpt("查询某阶段按日的成功/失败趋势序列。适用:趋势/每日变化/环比。stage=fff/fdr/fcl。"),
		InputSchema: aiSchemaWith(commonProps(), map[string]any{
			"stage": map[string]any{"type": "string", "enum": []string{"fff", "fdr", "fcl"}, "description": "链路阶段,必填"},
		}, "stage"),
	}, execStageQuery(fo, do, "trend"))

	add(anthropic.ToolParam{
		Name:        "get_overview",
		Description: param.NewOpt("查询整体概览:FFF 触发总数/成功率/失败数,FDR 落盘质量,FCL 上传质量。适用:概览/整体情况/成功率。"),
		InputSchema: aiSchema(commonProps()),
	}, execStageQuery(fo, do, "overview"))

	add(anthropic.ToolParam{
		Name:        "get_top",
		Description: param.NewOpt("查询 Top 榜单。kind=trigger 触发次数 Top 事件/mem 内存不足 Top/disk 磁盘不足 Top/quota 配额超限 Top/close 关闭次数 Top 筛选器。"),
		InputSchema: aiSchemaWith(commonProps(), map[string]any{
			"kind": map[string]any{"type": "string", "enum": []string{"trigger", "mem", "disk", "quota", "close"}, "description": "榜单类型,必填"},
		}, "kind"),
	}, execStageQuery(fo, do, "top"))

	add(anthropic.ToolParam{
		Name:        "get_quality",
		Description: param.NewOpt("查询阶段质量 P95 指标。stage=fdr:落盘耗时/磁盘/内存 P95、碎片率;fcl:Bag 大小 P95、上传量。"),
		InputSchema: aiSchemaWith(commonProps(), map[string]any{
			"stage": map[string]any{"type": "string", "enum": []string{"fdr", "fcl"}, "description": "链路阶段,必填"},
		}, "stage"),
	}, execStageQuery(fo, do, "quality"))

	// ---- 明细下钻工具:事件级明细表(汇总口径查不到细节时的降级/下钻通道) ----
	// 明细统一上限 100 条,note 提示模型告知用户"只支持查 100 条"
	add(anthropic.ToolParam{
		Name:        "get_detail",
		Description: param.NewOpt("查询事件级明细(每行=一次真实事件,亿级大表)。★慢查询纪律:必须带 event_names/anonymous_ids/start_dt(≤7天) 至少一项有索引条件才可直接查;只带 status/uuid 等无索引字段或裸查,必须先 clarify 向用户确认并建议缩小范围。kind=trigger 触发明细/uuid 全链路明细/running 运行明细/close 关闭明细/fdr 落盘明细/fcl 上传明细。明细只保留近几天;最多 limit 行(默认20,最大100),超出需告知用户只支持查前100条。"),
		InputSchema: aiSchemaWith(map[string]any{
			"start_dt":      map[string]any{"type": "string", "description": "开始日期 YYYY-MM-DD,缺省近3个月"},
			"end_dt":        map[string]any{"type": "string", "description": "结束日期 YYYY-MM-DD,缺省近3个月"},
			"event_names":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "事件名过滤"},
			"project_name":  map[string]any{"type": "string", "description": "项目过滤"},
			"car_types":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "车型过滤"},
			"limit":         map[string]any{"type": "integer", "description": "返回行数上限,默认 20,最大 100"},
			"anonymous_ids": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "按匿名 ID(车辆标识)过滤"},
			"uuid":          map[string]any{"type": "string", "description": "按链路 UUID 精确匹配(trigger/fdr/fcl/uuid)"},
			"status":        map[string]any{"type": "string", "enum": []string{"success", "fail", "discard", "waiting"}, "description": "按状态过滤(trigger/fdr/fcl/running)"},
			"fff_status":    map[string]any{"type": "string", "description": "按 FFF 阶段状态过滤(仅 kind=uuid)"},
			"fdr_status":    map[string]any{"type": "string", "description": "按 FDR 阶段状态过滤(仅 kind=uuid)"},
			"fcl_status":    map[string]any{"type": "string", "description": "按 FCL 阶段状态过滤(仅 kind=uuid)"},
		}, map[string]any{
			"kind": map[string]any{"type": "string", "enum": []string{"trigger", "uuid", "running", "close", "fdr", "fcl"}, "description": "明细类型"},
		}, "kind"),
	}, func(ctx context.Context, args map[string]any) (string, error) {
		start, end := defaultDateRange(args)
		events := argStringSlice(args, "event_names")
		cars := argStringSlice(args, "car_types")
		project := argString(args, "project_name")
		eventStr := strings.Join(events, ",")
		carStr := strings.Join(cars, ",")
		anonStr := strings.Join(argStringSlice(args, "anonymous_ids"), ",")
		limit := argLimit(args, 20, 100)

		kind := argString(args, "kind")
		var list any
		var total int64
		switch kind {
		case "trigger":
			res, err := fo.ListFffTrigger(ctx, &dashboard_api.FffTriggerRequest{
				EventNames: eventStr, ProjectName: project, CarTypes: carStr,
				StartDt: start, EndDt: end, Page: 1, PageSize: limit, AnonymousIds: anonStr,
				Uuid: argString(args, "uuid"), Status: argString(args, "status"),
			})
			if err != nil {
				return "", err
			}
			list, total = res.List, res.Total
		case "uuid":
			res, err := fo.ListUuidDetail(ctx, &dashboard_api.UuidDetailRequest{
				EventNames: eventStr, ProjectName: project, CarTypes: carStr,
				StartDt: start, EndDt: end, Page: 1, PageSize: limit, AnonymousIds: anonStr,
				Uuid: argString(args, "uuid"),
				FffStatus: argString(args, "fff_status"),
				FdrStatus: argString(args, "fdr_status"),
				FclStatus: argString(args, "fcl_status"),
			})
			if err != nil {
				return "", err
			}
			list, total = res.List, res.Total
		case "running":
			res, err := fo.ListFffRunning(ctx, &dashboard_api.FffRunningRequest{
				EventNames: eventStr, ProjectName: project, CarTypes: carStr,
				StartDt: start, EndDt: end, Page: 1, PageSize: limit, AnonymousIds: anonStr,
			})
			if err != nil {
				return "", err
			}
			list, total = res.List, res.Total
		case "close":
			res, err := fo.ListFffClose(ctx, &dashboard_api.FffCloseRequest{
				ProjectName: project, CarTypes: carStr,
				StartDt: start, EndDt: end, Page: 1, PageSize: limit, AnonymousIds: anonStr,
			})
			if err != nil {
				return "", err
			}
			list, total = res.List, res.Total
		case "fdr":
			res, err := fo.ListFdrTrigger(ctx, &dashboard_api.FdrTriggerRequest{
				EventNames: eventStr, ProjectName: project, CarTypes: carStr,
				StartDt: start, EndDt: end, Page: 1, PageSize: limit, AnonymousIds: anonStr,
				Uuid: argString(args, "uuid"), Status: argString(args, "status"),
			})
			if err != nil {
				return "", err
			}
			list, total = res.List, res.Total
		case "fcl":
			res, err := fo.ListFclTrigger(ctx, &dashboard_api.FclTriggerRequest{
				EventNames: eventStr, ProjectName: project, CarTypes: carStr,
				StartDt: start, EndDt: end, Page: 1, PageSize: limit, AnonymousIds: anonStr,
				Uuid: argString(args, "uuid"), Status: argString(args, "status"),
			})
			if err != nil {
				return "", err
			}
			list, total = res.List, res.Total
		default:
			return "", fmt.Errorf("kind 必须为 trigger/uuid/running/close/fdr/fcl")
		}
		note := fmt.Sprintf("共 %d 条,仅返回前 %d 行;明细查询最多支持 100 条,更多请到看板明细页导出", total, limit)
		if total > 100 {
			note = fmt.Sprintf("共 %d 条,仅返回前 %d 行;⚠️ 明细查询最多只支持 100 条,请在回答中明确告知用户该限制", total, limit)
		}
		return marshalToolResult(map[string]any{
			"note":  note,
			"total": total,
			"rows":  list,
		})
	})

	// ---- 聚合类补充:此前未接入的 biz 聚合方法,统一薄适配 ----
	common := func() map[string]any {
		return map[string]any{
			"start_dt":     map[string]any{"type": "string", "description": "开始日期 YYYY-MM-DD,缺省近3个月"},
			"end_dt":       map[string]any{"type": "string", "description": "结束日期 YYYY-MM-DD,缺省近3个月"},
			"event_names":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "事件名过滤"},
			"project_name": map[string]any{"type": "string", "description": "项目过滤"},
			"car_types":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "车型过滤"},
		}
	}
	add(anthropic.ToolParam{
		Name:        "get_funnel",
		Description: param.NewOpt("查询数采全链路漏斗:FFF触发→FDR落盘→FCL上传各级数量与整体成功率。适用:全链路转化/整体成功率/漏斗。"),
		InputSchema: aiSchema(common()),
	}, execStageQuery(fo, do, "funnel"))

	add(anthropic.ToolParam{
		Name:        "get_running_overview",
		Description: param.NewOpt("查询筛选器运行概览:运行总数/运行车辆数/开关次数/运行成败。适用:运行情况/多少车在跑/开关占比。"),
		InputSchema: aiSchema(common()),
	}, execStageQuery(fo, do, "running_overview"))

	add(anthropic.ToolParam{
		Name:        "get_running_trend",
		Description: param.NewOpt("查询某筛选器的运行活跃车辆趋势(按日,需指定筛选器)。适用:某筛选器活跃车辆变化。"),
		InputSchema: aiSchemaWith(common(), map[string]any{
			"filter_name": map[string]any{"type": "string", "description": "筛选器名称,必填"},
		}, "filter_name"),
	}, execStageQuery(fo, do, "running_trend"))

	add(anthropic.ToolParam{
		Name:        "get_sw_version",
		Description: param.NewOpt("查询软件版本分布(各版本事件量/车辆数)。适用:版本对比/各版本数据量。"),
		InputSchema: aiSchema(common()),
	}, execStageQuery(fo, do, "sw_version"))

	add(anthropic.ToolParam{
		Name:        "get_project_car",
		Description: param.NewOpt("查询项目×车型矩阵(各组合车辆数与事件量)。适用:项目车型分布/哪个项目车型最多。"),
		InputSchema: aiSchema(common()),
	}, execStageQuery(fo, do, "project_car"))

	add(anthropic.ToolParam{
		Name:        "get_project_event",
		Description: param.NewOpt("查询项目×事件交叉统计。适用:某项目下各事件量/事件按项目分布。"),
		InputSchema: aiSchema(common()),
	}, execStageQuery(fo, do, "project_event"))

	add(anthropic.ToolParam{
		Name:        "get_overview_events",
		Description: param.NewOpt("查询事件横向对比(每个事件的车辆数/触发数/三阶段成功率)。适用:哪些事件量最大/事件级成功率对比。"),
		InputSchema: aiSchema(common()),
	}, execStageQuery(fo, do, "overview"))

	add(anthropic.ToolParam{
		Name:        "get_trend",
		Description: param.NewOpt("查询数据量趋势(运行/触发/落盘/上传多指标按日)。适用:整体数据量变化/多指标趋势。"),
		InputSchema: aiSchema(common()),
	}, execStageQuery(fo, do, "trend"))

	add(anthropic.ToolParam{
		Name:        "get_fdr_fragment",
		Description: param.NewOpt("查询 FDR 碎片率(原始值为千分比)。适用:碎片率/存储碎片情况。"),
		InputSchema: aiSchema(common()),
	}, execStageQuery(fo, do, "fragment"))

	add(anthropic.ToolParam{
		Name:        "get_net_speed",
		Description: param.NewOpt("查询网络速率统计。适用:网速/上传速度情况。"),
		InputSchema: aiSchema(common()),
	}, execStageQuery(fo, do, "net_speed"))

	add(anthropic.ToolParam{
		Name:        "get_fcl_bw",
		Description: param.NewOpt("查询 FCL 上传带宽(按日均值/峰值)。适用:上传带宽/带宽趋势。"),
		InputSchema: aiSchema(common()),
	}, execStageQuery(fo, do, "fcl_bw"))

	add(anthropic.ToolParam{
		Name:        "get_top_vehicles",
		Description: param.NewOpt("查询数据量 Top 车辆。适用:哪台车数据最多/最活跃车辆。"),
		InputSchema: aiSchema(common()),
	}, execStageQuery(fo, do, "top_vehicles"))

	add(anthropic.ToolParam{
		Name:        "get_anomaly_vehicles",
		Description: param.NewOpt("查询异常车辆(触发多/成功率低等)。适用:异常车辆/哪些车有问题。"),
		InputSchema: aiSchema(common()),
	}, execStageQuery(fo, do, "anomaly_vehicles"))

	add(anthropic.ToolParam{
		Name:        "get_active_trend",
		Description: param.NewOpt("查询活跃车辆趋势(按日去重车辆数)。适用:活跃车辆数变化。"),
		InputSchema: aiSchema(common()),
	}, execStageQuery(fo, do, "active_trend"))

	add(anthropic.ToolParam{
		Name:        "get_cool_top",
		Description: param.NewOpt("查询冷却丢弃 Top 筛选器(被冷却最多的筛选器)。适用:哪个筛选器冷却最多。"),
		InputSchema: aiSchema(common()),
	}, execStageQuery(fo, do, "cool_top"))

	add(anthropic.ToolParam{
		Name:        "get_dimensions",
		Description: param.NewOpt("查询可选维度枚举:事件名/项目/车型/筛选器列表。适用:用户问有哪些可选值,或 clarify 前需要候选列表。"),
		InputSchema: aiSchema(map[string]any{}, ""),
	}, func(ctx context.Context, _ map[string]any) (string, error) {
		res, err := fo.GetDimensions(ctx, "")
		if err != nil {
			return "", err
		}
		// 截断防 prompt 膨胀
		out := map[string]any{
			"event_names": truncate(res.EventNames, 50),
			"projects":    truncate(res.ProjectNames, 50),
			"car_types":   truncate(res.CarTypes, 50),
			"filters":     truncate(res.FilterNames, 50),
		}
		return marshalToolResult(out)
	})

	return r
}

// Params 导出给 Anthropic 请求的 tools 定义(保持注册顺序)
func (r *aiToolRegistry) Params() []anthropic.ToolUnionParam {
	out := make([]anthropic.ToolUnionParam, 0, len(r.order))
	for _, name := range r.order {
		def := r.tools[name].def
		out = append(out, anthropic.ToolUnionParam{OfTool: &def})
	}
	return out
}

// Exec 执行白名单工具,返回 tool_result 内容(JSON 字符串)
func (r *aiToolRegistry) Exec(ctx context.Context, name string, args map[string]any) (string, error) {
	t, ok := r.tools[name]
	if !ok || t.exec == nil {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return t.exec(ctx, args)
}

// ---- 参数提取工具 ----

func argString(args map[string]any, key string) string {
	if v, ok := args[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func argStringSlice(args map[string]any, key string) []string {
	raw, ok := args[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}

// defaultDateRange 补齐缺省日期:近 7 天(含今天)
// enumNotes 枚举值 → 白话说明(工具返回时内嵌,模型解释数据时直接引用,
// 不依赖 system prompt 背字典——知识在使用点出现)
var enumNotes = map[string]string{
	// FFF 失败原因
	"cooldown":         "冷却丢弃(短时重复触发被系统主动丢弃,非故障)",
	"acquire_data":     "采集数据失败",
	"trigger_maximum":  "触发次数达上限",
	"drm_quota":        "DRM 配额限制",
	"switch_off":       "筛选器被关闭",
	// FDR 失败原因
	"max_files_exceeded":   "文件数超限",
	"mem_pool_water_line":  "内存池水位过高",
	"event_not_recognized": "事件未识别",
	"full_gc":              "Full GC 卡顿",
	"bag_invalid":          "数据包无效",
	"disk_overrun":         "磁盘超限",
	"unauthorized":         "未授权",
	// FCL 失败原因
	"quota_exceeded":             "云盘配额超限",
	"unexpected_bag_upload_query": "异常上传查询",
	"bag_not_exist":              "数据包不存在",
	"tls_error":                  "TLS 连接错误",
	"meta_file_lost":             "元数据丢失",
	"create_socket_failed":       "建连失败",
	"event_in_blacklist":         "事件被拉黑",
	"http_request_failed":        "HTTP 请求失败",
	// 通用
	"success": "成功", "failed": "失败", "discard": "丢弃", "other": "其他",
	// 关闭原因
	"out of memory":            "内存不足关闭",
	"not enough memory":        "内存不足关闭",
	"close operator for crash": "程序崩溃关闭",
}

// FFF / FDR_FCL 两组枚举白话(按 stage 提取子集,返回时内嵌)
var fffEnumNotes = func() map[string]string {
	keys := []string{"cooldown", "acquire_data", "trigger_maximum", "drm_quota", "switch_off", "other"}
	out := make(map[string]string, len(keys))
	for _, k := range keys {
		out[k] = enumNotes[k]
	}
	return out
}()

var fdrFclEnumNotes = func() map[string]string {
	keys := []string{"max_files_exceeded", "mem_pool_water_line", "event_not_recognized", "full_gc", "bag_invalid", "disk_overrun", "unauthorized", "quota_exceeded", "unexpected_bag_upload_query", "bag_not_exist", "tls_error", "meta_file_lost", "create_socket_failed", "event_in_blacklist", "http_request_failed", "other"}
	out := make(map[string]string, len(keys))
	for _, k := range keys {
		out[k] = enumNotes[k]
	}
	return out
}()

// argLimit 取整型参数并夹在 [def, max] 区间
func argLimit(args map[string]any, def, max int) int {
	v := def
	if f, ok := args["limit"].(float64); ok && f > 0 {
		v = int(f)
	}
	if v > max {
		v = max
	}
	return v
}

// defaultDateRange 汇总工具默认时间范围:近 3 个月(90 天)。
// 之前默认近 7 天,用户问"有哪些车型/项目"这类全局盘点问题时容易漏掉
// 近期不活跃的维度值——扩到 3 个月,由模型在回答中标注实际范围。
func defaultDateRange(args map[string]any) (string, string) {
	start, end := argString(args, "start_dt"), argString(args, "end_dt")
	if start == "" || end == "" {
		today := time.Now()
		start = today.AddDate(0, 0, -89).Format("2006-01-02")
		end = today.Format("2006-01-02")
	}
	return start, end
}

// execStageQuery 生成"公共查询参数 → 既有用例方法"的适配器
func execStageQuery(fo *FoDashboardUseCase, do *DoDashboardUseCase, kind string) func(context.Context, map[string]any) (string, error) {
	return func(ctx context.Context, args map[string]any) (string, error) {
		start, end := defaultDateRange(args)
		events := argStringSlice(args, "event_names")
		cars := argStringSlice(args, "car_types")
		project := argString(args, "project_name")
		eventStr := strings.Join(events, ",")
		carStr := strings.Join(cars, ",")

		triggerReq := &dashboard_api.FffTriggerRequest{EventNames: eventStr, ProjectName: project, CarTypes: carStr, StartDt: start, EndDt: end}
		trendReq := &dashboard_api.StageTrendRequest{EventNames: eventStr, ProjectName: project, CarTypes: carStr, StartDt: start, EndDt: end}
		doReq := &dashboard_api.DoCoolTopRequest{EventNames: eventStr, ProjectName: project, CarTypes: carStr, StartDt: start, EndDt: end}

		switch kind {
		case "fail_reason":
			switch argString(args, "stage") {
			case "fff":
				res, err := fo.GetFffFailReason(ctx, triggerReq)
				if err != nil {
					return "", err
				}
				return marshalToolResult(map[string]any{"list": truncateItems(res.List, 15), "enum_notes": fffEnumNotes, "note": "name 形如 FFF-<reason>"})
			case "fdr", "fcl":
				res, err := do.GetFailReason(ctx, &dashboard_api.DoFailReasonRequest{EventNames: eventStr, ProjectName: project, CarTypes: carStr, StartDt: start, EndDt: end})
				if err != nil {
					return "", err
				}
				return marshalToolResult(map[string]any{"list": truncateItems(res.List, 15), "enum_notes": fdrFclEnumNotes, "note": "仅含 " + argString(args, "stage") + " 阶段条目"})
			default:
				return "", fmt.Errorf("stage 必须为 fff/fdr/fcl")
			}
		case "trend":
			res, err := fo.GetStageTrend(ctx, trendReq)
			if err != nil {
				return "", err
			}
			stage := argString(args, "stage")
			series := res.Fff
			if stage == "fdr" {
				series = res.Fdr
			} else if stage == "fcl" {
				series = res.Fcl
			}
			// 派生指标由工具算好(total/daily 环比/各系列 7 日合计与占比),
			// 否则模型在 thinking 里手算亿级加法,极慢且易断流
			return marshalToolResult(buildTrendDerived(res.Dates, series))
		case "overview":
			fo_, err := fo.GetFffOverview(ctx, triggerReq)
			if err != nil {
				return "", err
			}
			fdrQ, err1 := do.GetFdrQuality(ctx, doReq)
			fclQ, err2 := do.GetFclQuality(ctx, doReq)
			out := map[string]any{"fff_overview": fo_}
			if err1 == nil {
				out["fdr_quality"] = fdrQ
			}
			if err2 == nil {
				out["fcl_quality"] = fclQ
			}
			return marshalToolResult(out)
		case "top":
			switch argString(args, "kind") {
			case "trigger":
				res, err := do.GetTriggerRank(ctx, doReq)
				if err != nil {
					return "", err
				}
				return marshalToolResult(map[string]any{"list": truncateItems(res.List, 10)})
			case "mem":
				res, err := do.GetMemTop(ctx, doReq)
				if err != nil {
					return "", err
				}
				return marshalToolResult(map[string]any{"list": truncateItems(res.List, 10)})
			case "disk":
				res, err := do.GetDiskTop(ctx, doReq)
				if err != nil {
					return "", err
				}
				return marshalToolResult(map[string]any{"list": truncateItems(res.List, 10)})
			case "quota":
				res, err := do.GetQuotaTop(ctx, doReq)
				if err != nil {
					return "", err
				}
				return marshalToolResult(map[string]any{"list": truncateItems(res.List, 10)})
			case "close":
				res, err := do.GetCloseTop(ctx, doReq)
				if err != nil {
					return "", err
				}
				return marshalToolResult(map[string]any{"list": truncateItems(res.List, 10), "note": "关闭次数 Top 筛选器"})
			default:
				return "", fmt.Errorf("kind 必须为 trigger/mem/disk/quota/close")
			}
		case "quality":
			switch argString(args, "stage") {
			case "fdr":
				res, err := do.GetFdrQuality(ctx, doReq)
				if err != nil {
					return "", err
				}
				return marshalToolResult(map[string]any{"fdr_quality": res})
			case "fcl":
				res, err := do.GetFclQuality(ctx, doReq)
				if err != nil {
					return "", err
				}
				return marshalToolResult(map[string]any{"fcl_quality": res})
			default:
				return "", fmt.Errorf("stage 必须为 fdr 或 fcl")
			}
		case "funnel":
			res, err := fo.GetFunnel(ctx, &dashboard_api.FunnelRequest{
				EventNames: eventStr, ProjectName: project, CarTypes: carStr, StartDt: start, EndDt: end,
			})
			if err != nil {
				return "", err
			}
			return marshalToolResult(res)
		case "running_overview":
			res, err := fo.GetRunningOverview(ctx, &dashboard_api.FffRunningRequest{
				EventNames: eventStr, ProjectName: project, CarTypes: carStr, StartDt: start, EndDt: end,
			})
			if err != nil {
				return "", err
			}
			return marshalToolResult(res)
		case "running_trend":
			res, err := fo.GetFffRunningTrend(ctx, &dashboard_api.FffRunningTrendRequest{
				FilterName: argString(args, "filter_name"), ProjectName: project, CarTypes: carStr, StartDt: start, EndDt: end,
			})
			if err != nil {
				return "", err
			}
			return marshalToolResult(res)
		case "sw_version":
			res, err := do.GetSwVersion(ctx, doReq)
			if err != nil {
				return "", err
			}
			return marshalToolResult(map[string]any{"list": truncateItems(res.List, 15)})
		case "project_car":
			res, err := do.GetProjectCar(ctx, doReq)
			if err != nil {
				return "", err
			}
			return marshalToolResult(res)
		case "project_event":
			res, err := do.GetProjectEvent(ctx, doReq)
			if err != nil {
				return "", err
			}
			return marshalToolResult(map[string]any{"list": truncateItems(res.List, 15)})
		case "overview_events": // 事件横向对比
			res, err := do.GetOverview(ctx, &dashboard_api.DoOverviewRequest{
				EventNames: eventStr, ProjectName: project, CarTypes: carStr, StartDt: start, EndDt: end,
			})
			if err != nil {
				return "", err
			}
			return marshalToolResult(map[string]any{"list": truncateItems(res.List, 15)})
		case "trend_multi": // 数据量多指标趋势
			res, err := do.GetTrend(ctx, &dashboard_api.DoTrendRequest{
				EventNames: eventStr, ProjectName: project, CarTypes: carStr, StartDt: start, EndDt: end,
			})
			if err != nil {
				return "", err
			}
			return marshalToolResult(res)
		case "fragment":
			res, err := do.GetFdrFragment(ctx, doReq)
			if err != nil {
				return "", err
			}
			return marshalToolResult(res)
		case "net_speed":
			res, err := do.GetNetSpeed(ctx, doReq)
			if err != nil {
				return "", err
			}
			return marshalToolResult(res)
		case "fcl_bw":
			res, err := do.GetFclBw(ctx, doReq)
			if err != nil {
				return "", err
			}
			return marshalToolResult(res)
		case "top_vehicles":
			res, err := do.GetTopVehicles(ctx, &dashboard_api.DoTopVehicleRequest{
				EventNames: eventStr, ProjectName: project, CarTypes: carStr, StartDt: start, EndDt: end,
			})
			if err != nil {
				return "", err
			}
			return marshalToolResult(map[string]any{"list": truncateItems(res.List, 15)})
		case "anomaly_vehicles":
			res, err := do.GetAnomalyVehicles(ctx, &dashboard_api.DoAnomalyRequest{
				EventNames: eventStr, ProjectName: project, CarTypes: carStr, StartDt: start, EndDt: end,
			})
			if err != nil {
				return "", err
			}
			return marshalToolResult(map[string]any{"list": truncateItems(res.List, 15)})
		case "active_trend":
			res, err := do.GetActiveTrend(ctx, &dashboard_api.DoTopVehicleRequest{
				EventNames: eventStr, ProjectName: project, CarTypes: carStr, StartDt: start, EndDt: end,
			})
			if err != nil {
				return "", err
			}
			return marshalToolResult(res)
		case "cool_top":
			res, err := do.GetCoolTop(ctx, doReq)
			if err != nil {
				return "", err
			}
			return marshalToolResult(map[string]any{"list": truncateItems(res.List, 10), "note": "冷却丢弃 Top 筛选器"})
		}
		return "", fmt.Errorf("unknown query kind: %s", kind)
	}
}

// buildTrendDerived 趋势结果 + 派生指标:
// daily_total(逐日总量)、daily_total_pct_change(环比%,首日 null)、
// totals(各系列合计)、totals_pct(占比%)、7 日合计行。
// 目的:模型零算术,直接引用;series 原始数据保留(画图用)。
func buildTrendDerived(dates []string, series []*dashboard_api.StageTrendSeries) map[string]any {
	out := map[string]any{"dates": dates, "series": series}
	if len(dates) == 0 {
		return out
	}
	n := len(dates)

	// 逐日总量与环比
	totalByDay := make([]int64, n)
	for _, s := range series {
		for i, v := range s.Data {
			if i < n {
				totalByDay[i] += v
			}
		}
	}
	totalPct := make([]*float64, n) // 首日 null
	for i := 1; i < n; i++ {
		if totalByDay[i-1] > 0 {
			p := float64(totalByDay[i]-totalByDay[i-1]) / float64(totalByDay[i-1]) * 100
			totalPct[i] = &p
		}
	}
	out["daily_total"] = totalByDay
	out["daily_total_pct_change"] = totalPct

	// 各系列合计与占比(占全部系列总和)
	type seriesTotal struct {
		Name string   `json:"name"`
		Total int64   `json:"total"`
		Pct  *float64 `json:"pct_of_all"`
	}
	var grand int64
	totals := make([]seriesTotal, 0, len(series))
	for _, s := range series {
		var t int64
		for _, v := range s.Data {
			t += v
		}
		totals = append(totals, seriesTotal{Name: s.Name, Total: t})
		grand += t
	}
	if grand > 0 {
		for i := range totals {
			p := float64(totals[i].Total) / float64(grand) * 100
			totals[i].Pct = &p
		}
	}
	out["series_totals"] = totals
	out["grand_total"] = grand
	// series name 白话(内嵌到返回,模型解释趋势时不依赖 system prompt 背字典)
	if sn := seriesNotes(series); len(sn) > 0 {
		out["series_notes"] = sn
	}
	return out
}

// seriesNotes 从 series name 提取枚举白话子集
func seriesNotes(series []*dashboard_api.StageTrendSeries) map[string]string {
	sn := make(map[string]string)
	for _, s := range series {
		if note, ok := enumNotes[s.Name]; ok {
			sn[s.Name] = note
		}
	}
	return sn
}



func aiSchema(props map[string]any, required ...string) anthropic.ToolInputSchemaParam {
	req := []string{}
	for _, r := range required {
		if r != "" {
			req = append(req, r)
		}
	}
	return anthropic.ToolInputSchemaParam{Type: "object", Properties: props, Required: req}
}

func aiSchemaWith(base, extra map[string]any, required ...string) anthropic.ToolInputSchemaParam {
	for k, v := range extra {
		base[k] = v
	}
	return aiSchema(base, required...)
}

func marshalToolResult(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func truncate(list []string, n int) []string {
	if len(list) > n {
		return list[:n]
	}
	return list
}

// truncateItems 泛型截断任意指针切片(Top 榜单防膨胀)
func truncateItems[T any](list []T, n int) []T {
	if len(list) > n {
		return list[:n]
	}
	return list
}
