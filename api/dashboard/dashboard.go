package dashboard_api

import "encoding/json"

// BaseResponse 统一响应基础结构，实现 api.HttpResponse 接口
type BaseResponse struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
}

func (r *BaseResponse) GetCode() int32     { return r.Code }
func (r *BaseResponse) GetMessage() string { return r.Message }

// FffRunningItem 筛选器运行明细单条记录，字段与 dwd_cfdi_basic_fff_running 列对齐
type FffRunningItem struct {
	Dt              string `json:"dt"`
	FilterName      string `json:"filter_name"`
	AnonymousId     string `json:"anonymous_id"`
	TimestampUtc    string `json:"timestamp_utc"`
	CreateAt        string `json:"create_at"`
	CollectType     string `json:"collect_type"`
	SwVersion       string `json:"sw_version"`
	ProjectName     string `json:"project_name"`
	CarType         string `json:"car_type"`
	VehicleSource   string `json:"vehicle_source"`
	SwitchOn        bool   `json:"switch_on"`
	Version         string `json:"version"`
	OnAutopilot     bool   `json:"on_autopilot"`
	FunctionMode    string `json:"function_mode"`
	Status          string `json:"status"`
	FdiProjectName  string `json:"fdi_project_name"`
	ProjectCarType  string `json:"project_car_type"`
	VehicleSourceCn string `json:"vehicle_source_cn"`
}

// FffRunningRequest 筛选器运行明细查询请求
type FffRunningRequest struct {
	FilterName   string `form:"filter_name"`
	EventNames   string `form:"event_names"`
	EventName    string `form:"event_name"` // 兼容单数写法;与 event_names 任取其一
	ProjectName  string `form:"project_name"`
	CarTypes     string `form:"car_types"`
	CarType      string `form:"car_type"`      // 兼容单数写法;与 car_types 任取其一
	AnonymousIds string `form:"anonymous_ids"` // 车辆ID,多选逗号分隔
	AnonymousId  string `form:"anonymous_id"`  // 兼容单数写法(前端旧参数);与 anonymous_ids 任取其一
	SwitchOn     *int   `form:"switch_on"`     // 开关:1=开启 0=关闭(有索引)
	SwVersion    string `form:"sw_version"`    // 软件版本(有索引)
	StartDt      string `form:"start_dt"`
	EndDt        string `form:"end_dt"`
	Page         int    `form:"page"`
	PageSize     int    `form:"page_size"`
}

// FffRunningResponse 筛选器运行明细响应
type FffRunningResponse struct {
	BaseResponse
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
	List     []*FffRunningItem `json:"list"`
}

// FffTriggerItem 筛选器触发明细单条记录，字段与 dwd_cfdi_basic_fff_trigger 列对齐
type FffTriggerItem struct {
	Dt              string `json:"dt"`
	Uuid            string `json:"uuid"`
	EventName       string `json:"event_name"`
	AnonymousId     string `json:"anonymous_id"`
	TimestampUtc    string `json:"timestamp_utc"`
	CreateAt        string `json:"create_at"`
	TriggerTime     int64  `json:"trigger_time"`
	UtcDiffUs       int64  `json:"utc_diff_us"`
	Before          int    `json:"before"`
	After           int    `json:"after"`
	FilterName      string `json:"filter_name"`
	TriggerType     string `json:"trigger_type"`
	CollectType     string `json:"collect_type"`
	Status          string `json:"status"`
	OnAutopilot     bool   `json:"on_autopilot"`
	FunctionMode    string `json:"function_mode"`
	SwVersion       string `json:"sw_version"`
	ProjectName     string `json:"project_name"`
	CarType         string `json:"car_type"`
	VehicleSource   string `json:"vehicle_source"`
	Bj02Lat         string `json:"bj02_lat"`
	Bj02Lon         string `json:"bj02_lon"`
	RoadType        string `json:"road_type"`
	FdiProjectName  string `json:"fdi_project_name"`
	ProjectCarType  string `json:"project_car_type"`
	VehicleSourceCn string `json:"vehicle_source_cn"`
	Tags            string `json:"tags"`
	Detail          string `json:"detail"`
}

// FffTriggerRequest 筛选器触发明细查询请求
type FffTriggerRequest struct {
	FilterName   string `form:"filter_name"`
	EventNames   string `form:"event_names"` // 多选，逗号分隔
	EventName    string `form:"event_name"`  // 兼容单数写法;与 event_names 任取其一
	ProjectName  string `form:"project_name"`
	CarTypes     string `form:"car_types"`
	CarType      string `form:"car_type"`      // 兼容单数写法;与 car_types 任取其一
	AnonymousIds string `form:"anonymous_ids"` // 车辆ID,多选逗号分隔
	AnonymousId  string `form:"anonymous_id"`  // 兼容单数写法(前端旧参数);与 anonymous_ids 任取其一
	Uuid         string `form:"uuid"`          // 链路UUID(有索引)
	Status       string `form:"status"`        // 状态(有索引)
	TriggerType  string `form:"trigger_type"`  // 触发类型(有索引)
	Tags         string `form:"tags"`          // 标签(分词索引)
	StartDt      string `form:"start_dt"`
	EndDt        string `form:"end_dt"`
	Page         int    `form:"page"`
	PageSize     int    `form:"page_size"`
}

// FffTriggerResponse 筛选器触发明细响应
type FffTriggerResponse struct {
	BaseResponse
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
	List     []*FffTriggerItem `json:"list"`
}

// FffCloseItem 筛选器关闭明细单条记录，字段与 dwd_cfdi_basic_fff_close 列对齐
type FffCloseItem struct {
	Dt              string `json:"dt"`
	FilterName      string `json:"filter_name"`
	Version         string `json:"version"`
	Reason          string `json:"reason"`
	AnonymousId     string `json:"anonymous_id"`
	CreateAt        string `json:"create_at"`
	SwVersion       string `json:"sw_version"`
	TimestampUtc    string `json:"timestamp_utc"`
	ProjectName     string `json:"project_name"`
	CarType         string `json:"car_type"`
	VehicleSource   string `json:"vehicle_source"`
	FdiProjectName  string `json:"fdi_project_name"`
	ProjectCarType  string `json:"project_car_type"`
	VehicleSourceCn string `json:"vehicle_source_cn"`
}

// FffCloseRequest 筛选器关闭明细查询请求
type FffCloseRequest struct {
	FilterName   string `form:"filter_name"`
	ProjectName  string `form:"project_name"`
	CarTypes     string `form:"car_types"`
	AnonymousIds string `form:"anonymous_ids"` // 车辆ID,多选逗号分隔
	AnonymousId  string `form:"anonymous_id"`  // 兼容单数写法(前端旧参数);与 anonymous_ids 任取其一
	Reason       string `form:"reason"`        // 关闭原因(分词索引)
	Version      string `form:"version"`       // 筛选器版本(有索引)
	StartDt      string `form:"start_dt"`
	EndDt        string `form:"end_dt"`
	Page         int    `form:"page"`
	PageSize     int    `form:"page_size"`
}

// FffCloseResponse 筛选器关闭明细响应
type FffCloseResponse struct {
	BaseResponse
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
	List     []*FffCloseItem `json:"list"`
}

// FdrTriggerItem FDR 落盘明细单条记录，字段与 dwd_basic_fdr_trigger 列对齐
type FdrTriggerItem struct {
	Dt                string `json:"dt"`
	Uuid              string `json:"uuid"`
	EventName         string `json:"event_name"`
	AnonymousId       string `json:"anonymous_id"`
	TimestampUtc      string `json:"timestamp_utc"`
	CreateAt          string `json:"create_at"`
	SwVersion         string `json:"sw_version"`
	Dse               string `json:"dse"`
	TdMb              string `json:"td_mb"`
	TmMb              string `json:"tm_mb"`
	TriggerTimestamp  int64  `json:"trigger_timestamp"`
	BeginTimestampUts int64  `json:"begin_timestamp_uts"`
	EndTimestampUts   int64  `json:"end_timestamp_uts"`
	DumpTimestamp     int64  `json:"dump_timestamp"`
	Status            string `json:"status"`
	Detail            string `json:"detail"`
	TimeCostMs        string `json:"time_cost_ms"`
	RecordType        string `json:"type"`
	ProjectName       string `json:"project_name"`
	CarType           string `json:"car_type"`
	VehicleSource     string `json:"vehicle_source"`
	FdiProjectName    string `json:"fdi_project_name"`
	ProjectCarType    string `json:"project_car_type"`
	VehicleSourceCn   string `json:"vehicle_source_cn"`
}

// FdrTriggerRequest FDR 落盘明细查询请求
type FdrTriggerRequest struct {
	FilterName   string `form:"filter_name"`
	EventNames   string `form:"event_names"` // 多选，逗号分隔
	EventName    string `form:"event_name"`  // 兼容单数写法;与 event_names 任取其一
	ProjectName  string `form:"project_name"`
	CarTypes     string `form:"car_types"`
	CarType      string `form:"car_type"`      // 兼容单数写法;与 car_types 任取其一
	AnonymousIds string `form:"anonymous_ids"` // 车辆ID,多选逗号分隔
	AnonymousId  string `form:"anonymous_id"`  // 兼容单数写法(前端旧参数);与 anonymous_ids 任取其一
	Uuid         string `form:"uuid"`          // 链路UUID(有索引)
	Status       string `form:"status"`        // 状态(有索引)
	Detail       string `form:"detail"`        // 详情(分词索引)
	StartDt      string `form:"start_dt"`
	EndDt        string `form:"end_dt"`
	Page         int    `form:"page"`
	PageSize     int    `form:"page_size"`
}

// FdrTriggerResponse FDR 落盘明细响应
type FdrTriggerResponse struct {
	BaseResponse
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
	List     []*FdrTriggerItem `json:"list"`
}

// FclTriggerItem FCL 上传明细单条记录，字段与 dwd_cfdi_basic_fcl_trigger 列对齐
type FclTriggerItem struct {
	Dt              string `json:"dt"`
	Uuid            string `json:"uuid"`
	EventName       string `json:"event_name"`
	AnonymousId     string `json:"anonymous_id"`
	TimestampUtc    string `json:"timestamp_utc"`
	CreateAt        string `json:"create_at"`
	Status          string `json:"status"`
	Detail          string `json:"detail"`
	CompletePercent int    `json:"complete_percent"`
	LocalFile       string `json:"local_file"`
	UploadFailTimes int    `json:"upload_fail_times"`
	PrefixStitch    string `json:"prefix_stitch"`
	TriggerSource   string `json:"trigger_source"`
	SwVersion       string `json:"sw_version"`
	ProjectName     string `json:"project_name"`
	CarType         string `json:"car_type"`
	VehicleSource   string `json:"vehicle_source"`
	FdiProjectName  string `json:"fdi_project_name"`
	ProjectCarType  string `json:"project_car_type"`
	VehicleSourceCn string `json:"vehicle_source_cn"`
}

// FclTriggerRequest FCL 上传明细查询请求
type FclTriggerRequest struct {
	FilterName   string `form:"filter_name"`
	EventNames   string `form:"event_names"` // 多选，逗号分隔
	EventName    string `form:"event_name"`  // 兼容单数写法;与 event_names 任取其一
	ProjectName  string `form:"project_name"`
	CarTypes     string `form:"car_types"`
	CarType      string `form:"car_type"`      // 兼容单数写法;与 car_types 任取其一
	AnonymousIds string `form:"anonymous_ids"` // 车辆ID,多选逗号分隔
	AnonymousId  string `form:"anonymous_id"`  // 兼容单数写法(前端旧参数);与 anonymous_ids 任取其一
	Uuid         string `form:"uuid"`          // 链路UUID(有索引)
	Status       string `form:"status"`        // 状态(有索引)
	StartDt      string `form:"start_dt"`
	EndDt        string `form:"end_dt"`
	Page         int    `form:"page"`
	PageSize     int    `form:"page_size"`
}

// FclTriggerResponse FCL 上传明细响应
type FclTriggerResponse struct {
	BaseResponse
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
	List     []*FclTriggerItem `json:"list"`
}

// UuidDetailItem 全链路明细单条记录，字段与 dwd_cfdi_status_monitor_analysis 列对齐
type UuidDetailItem struct {
	Dt                string `json:"dt"`
	AnonymousId       string `json:"anonymous_id"`
	EventName         string `json:"event_name"`
	Uuid              string `json:"uuid"`
	CreateAt          string `json:"create_at"`
	FilterName        string `json:"filter_name"`
	FffSwVersion      string `json:"fff_sw_version"`
	FdrSwVersion      string `json:"fdr_sw_version"`
	FclSwVersion      string `json:"fcl_sw_version"`
	TriggerType       string `json:"trigger_type"`
	CollectType       string `json:"collect_type"`
	FffUpdatedAt      int64  `json:"fff_updated_at"`
	FdrUpdatedAt      int64  `json:"fdr_updated_at"`
	FclUpdatedAt      int64  `json:"fcl_updated_at"`
	FffStatus         string `json:"fff_status"`
	FdrStatus         string `json:"fdr_status"`
	FclStatus         string `json:"fcl_status"`
	FffDetail         string `json:"fff_detail"`
	FdrDetail         string `json:"fdr_detail"`
	FclDetail         string `json:"fcl_detail"`
	BeginTimestampUts int64  `json:"begin_timestamp_uts"`
	DumpTimestamp     int64  `json:"dump_timestamp"`
	EndTimestampUts   int64  `json:"end_timestamp_uts"`
	Md5               string `json:"md5"`
	BagName           string `json:"bag_name"`
	CompletePercent   int    `json:"complete_percent"`
	ProjectName       string `json:"project_name"`
	CarType           string `json:"car_type"`
	VehicleSource     string `json:"vehicle_source"`
	TimestampUtc      string `json:"timestamp_utc"`
	FdiProjectName    string `json:"fdi_project_name"`
	ProjectCarType    string `json:"project_car_type"`
	VehicleSourceCn   string `json:"vehicle_source_cn"`
	Dse               string `json:"dse"`
}

// UuidDetailRequest 全链路明细查询请求
// 筛选参数来自「筛选器诊断」filter bar（diag*），而非「明细分析」filter bar（daily*）
type UuidDetailRequest struct {
	FilterName   string `form:"filter_name"`
	EventNames   string `form:"event_names"` // 多选，逗号分隔
	EventName    string `form:"event_name"`  // 兼容单数写法;与 event_names 任取其一
	ProjectName  string `form:"project_name"`
	CarTypes     string `form:"car_types"`
	CarType      string `form:"car_type"`      // 兼容单数写法;与 car_types 任取其一
	AnonymousIds string `form:"anonymous_ids"` // 车辆ID,多选逗号分隔
	AnonymousId  string `form:"anonymous_id"`  // 兼容单数写法(前端旧参数);与 anonymous_ids 任取其一
	Uuid         string `form:"uuid"`          // 链路UUID(有索引)
	FffStatus    string `form:"fff_status"`    // FFF 阶段状态(有索引)
	FdrStatus    string `form:"fdr_status"`    // FDR 阶段状态(有索引)
	FclStatus    string `form:"fcl_status"`    // FCL 阶段状态(有索引)
	StartDt      string `form:"start_dt"`
	EndDt        string `form:"end_dt"`
	OnlyFail     int    `form:"only_fail"`    // 1=仅看失败
	StageFilter  string `form:"stage_filter"` // fff_discard/fdr_discard/fcl_discard/fcl_success
	Page         int    `form:"page"`
	PageSize     int    `form:"page_size"`
}

// UuidDetailResponse 全链路明细响应
type UuidDetailResponse struct {
	BaseResponse
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
	List     []*UuidDetailItem `json:"list"`
}

// FunnelStat 数采全链路统计数字
type FunnelStat struct {
	FffTotal   int64   `json:"fff_total"`
	FffAllow   int64   `json:"fff_allow"`
	FdrSuccess int64   `json:"fdr_success"`
	FdrFail    int64   `json:"fdr_fail"` // 直接从 FDR 表统计，避免跨表日期错位
	FclSuccess int64   `json:"fcl_success"`
	FclFail    int64   `json:"fcl_fail"` // 直接从 FCL 表统计
	CfdiRate   float64 `json:"cfdi_rate"`
}

// FunnelFailReason 单条失败原因
type FunnelFailReason struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// FunnelRequest 数采全链路分析请求
type FunnelRequest struct {
	FilterName  string `form:"filter_name"`
	EventNames  string `form:"event_names"`
	ProjectName string `form:"project_name"`
	CarTypes    string `form:"car_types"`
	StartDt     string `form:"start_dt"`
	EndDt       string `form:"end_dt"`
}

// DoFunnelRequest DO 数采全链路分析请求
type DoFunnelRequest struct {
	FilterName  string `form:"filter_name"`
	EventNames  string `form:"event_names"`
	ProjectName string `form:"project_name"`
	CarTypes    string `form:"car_types"`
	StartDt     string `form:"start_dt"`
	EndDt       string `form:"end_dt"`
}

// FunnelResponse 数采全链路分析响应
type FunnelResponse struct {
	BaseResponse
	Stat    *FunnelStat         `json:"stat"`
	FffFail []*FunnelFailReason `json:"fff_fail"`
	FdrFail []*FunnelFailReason `json:"fdr_fail"`
	FclFail []*FunnelFailReason `json:"fcl_fail"`
}

// StageTrendSeries 单条时序数据
type StageTrendSeries struct {
	Name string  `json:"name"`
	Data []int64 `json:"data"`
}

// StageTrendRequest 三阶段触发趋势请求
type StageTrendRequest struct {
	FilterName  string `form:"filter_name"`
	EventNames  string `form:"event_names"` // 多选，逗号分隔
	ProjectName string `form:"project_name"`
	CarTypes    string `form:"car_types"`
	StartDt     string `form:"start_dt"`
	EndDt       string `form:"end_dt"`
}

// StageTrendResponse 三阶段触发趋势响应
type StageTrendResponse struct {
	BaseResponse
	Dates []string            `json:"dates"`
	Fff   []*StageTrendSeries `json:"fff"`
	Fdr   []*StageTrendSeries `json:"fdr"`
	Fcl   []*StageTrendSeries `json:"fcl"`
}

// CloseReasonItem 算子关闭原因分布单项
type CloseReasonItem struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

// CloseReasonRequest 算子关闭原因分布请求
type CloseReasonRequest struct {
	FilterName  string `form:"filter_name"`
	EventNames  string `form:"event_names"`
	ProjectName string `form:"project_name"`
	CarTypes    string `form:"car_types"`
	StartDt     string `form:"start_dt"`
	EndDt       string `form:"end_dt"`
}

// CloseReasonResponse 算子关闭原因分布响应
type CloseReasonResponse struct {
	BaseResponse
	List []*CloseReasonItem `json:"list"`
}

// DoOverviewItem 事件横向对比单行
type DoOverviewItem struct {
	EventName    string  `json:"event_name"`
	VehicleCount int64   `json:"vehicle_count"`
	TriggerCount int64   `json:"trigger_count"`
	CfdiRate     float64 `json:"cfdi_rate"`
	FffCount     int64   `json:"fff_count"`
	FffRate      float64 `json:"fff_rate"`
	FdrCount     int64   `json:"fdr_count"`
	FdrRate      float64 `json:"fdr_rate"`
	FclCount     int64   `json:"fcl_count"`
	FclRate      float64 `json:"fcl_rate"`
}

// DoOverviewRequest 事件横向对比请求
type DoOverviewRequest struct {
	FilterName  string `form:"filter_name"`
	EventNames  string `form:"event_names"`
	ProjectName string `form:"project_name"`
	CarTypes    string `form:"car_types"`
	StartDt     string `form:"start_dt"`
	EndDt       string `form:"end_dt"`
}

// DoOverviewResponse 事件横向对比响应
type DoOverviewResponse struct {
	BaseResponse
	List []*DoOverviewItem `json:"list"`
}

// DoTrendRequest DO 数据总览趋势请求
type DoTrendRequest struct {
	FilterName  string `form:"filter_name"`
	EventNames  string `form:"event_names"` // 多选，逗号分隔
	ProjectName string `form:"project_name"`
	CarTypes    string `form:"car_types"` // 多选，逗号分隔
	StartDt     string `form:"start_dt"`
	EndDt       string `form:"end_dt"`
}

// DoTrendResponse DO 数据总览趋势响应
type DoTrendResponse struct {
	BaseResponse
	Dates         []string  `json:"dates"`
	SuccessCounts []int64   `json:"success_counts"`
	SuccessRates  []float64 `json:"success_rates"`
}

// DimensionsRequest 下拉维度请求(project_name 非空时 car_types 联动过滤)
type DimensionsRequest struct {
	ProjectName string `form:"project_name"`
}

// DimensionsResponse FO/DO Dashboard 公共下拉维度响应
type DimensionsResponse struct {
	BaseResponse
	FilterNames  []string `json:"filter_names"`
	EventNames   []string `json:"event_names"`
	ProjectNames []string `json:"project_names"`
	CarTypes     []string `json:"car_types"`
}

// FoDimensionsResponse 兼容别名，防止旧引用编译报错
type FoDimensionsResponse = DimensionsResponse

// DoCoolTopItem 冷却 Top 筛选器单条
type DoCoolTopItem struct {
	FilterName string `json:"filter_name"`
	Count      int64  `json:"count"`
}

// DoCoolTopRequest 冷却 Top 筛选器请求
type DoCoolTopRequest struct {
	FilterName  string `form:"filter_name"`
	EventNames  string `form:"event_names"`
	ProjectName string `form:"project_name"`
	CarTypes    string `form:"car_types"`
	StartDt     string `form:"start_dt"`
	EndDt       string `form:"end_dt"`
}

// DoCoolTopResponse 冷却 Top 筛选器响应
type DoCoolTopResponse struct {
	BaseResponse
	List []*DoCoolTopItem `json:"list"`
}

// DoProjectEventItem 项目触发回流事件总数单条
type DoProjectEventItem struct {
	ProjectName string `json:"project_name"`
	EventCount  int64  `json:"event_count"`
}

// DoProjectEventResponse 项目触发回流事件总数响应
type DoProjectEventResponse struct {
	BaseResponse
	List []*DoProjectEventItem `json:"list"`
}

// DoQuotaTopResponse FCL Quota 超限 Top20 响应
type DoQuotaTopResponse struct {
	BaseResponse
	List []*DoEventTopItem `json:"list"`
}

// DoEventTopItem FDR 内存/磁盘 Top 单条（按事件聚合）
type DoEventTopItem struct {
	EventName string `json:"event_name"`
	Count     int64  `json:"count"`
}

// DoMemTopResponse FDR 内存不足 Top20 响应
type DoMemTopResponse struct {
	BaseResponse
	List []*DoEventTopItem `json:"list"`
}

// DoDiskTopResponse FDR 磁盘不足 Top20 响应
type DoDiskTopResponse struct {
	BaseResponse
	List []*DoEventTopItem `json:"list"`
}

// DoCloseTopResponse 关闭次数 Top 筛选器响应
type DoCloseTopResponse struct {
	BaseResponse
	List []*DoCoolTopItem `json:"list"`
}

// DoProjectCarResponse 项目×车型分布响应
type DoProjectCarResponse struct {
	BaseResponse
	Projects      []string           `json:"projects"`       // Y轴项目名，按触发量降序
	CarTypes      []string           `json:"car_types"`      // 动态车型列表
	Matrix        map[string][]int64 `json:"matrix"`         // car_type -> 各项目触发次数
	ProjectTotals []int64            `json:"project_totals"` // 各项目触发总量
}

// DoTriggerRankResponse 触发频次排行响应
type DoTriggerRankResponse struct {
	BaseResponse
	List []*DoCoolTopItem `json:"list"`
}

// DoSwVersionItem 软件版本分布单条
type DoSwVersionItem struct {
	SwVersion string `json:"sw_version"`
	Count     int64  `json:"count"`
}

// DoSwVersionResponse 软件版本分布响应
type DoSwVersionResponse struct {
	BaseResponse
	List []*DoSwVersionItem `json:"list"`
}

// DoVehicleItem 车辆维度分析单条（Top活跃/异常车辆共用）
type DoVehicleItem struct {
	AnonymousId  string  `json:"anonymous_id"`
	CarType      string  `json:"car_type"`
	ProjectName  string  `json:"project_name"`
	TriggerCount int64   `json:"trigger_count"`
	SuccessCount int64   `json:"success_count"`
	CfdiRate     float64 `json:"cfdi_rate"`
	MainReason   string  `json:"main_reason"`
}

// DoTopVehicleRequest Top活跃车辆请求
type DoTopVehicleRequest struct {
	EventNames  string `form:"event_names"`
	ProjectName string `form:"project_name"`
	CarTypes    string `form:"car_types"`
	StartDt     string `form:"start_dt"`
	EndDt       string `form:"end_dt"`
}

// DoTopVehicleResponse Top活跃车辆响应
type DoTopVehicleResponse struct {
	BaseResponse
	List []*DoVehicleItem `json:"list"`
}

// DoAnomalyRequest 异常车辆请求
type DoAnomalyRequest struct {
	EventNames  string `form:"event_names"`
	ProjectName string `form:"project_name"`
	CarTypes    string `form:"car_types"`
	StartDt     string `form:"start_dt"`
	EndDt       string `form:"end_dt"`
	MaxRate     int    `form:"max_rate"` // 成功率上限（整数百分比）
}

// DoAnomalyResponse 异常车辆响应
type DoAnomalyResponse struct {
	BaseResponse
	List []*DoVehicleItem `json:"list"`
}

// DoActiveTrendResponse 活跃车辆趋势响应
type DoActiveTrendResponse struct {
	BaseResponse
	Dates  []string `json:"dates"`
	Counts []int64  `json:"counts"`
}

// DoNetSpeedSeries 网速统计单条时序（按车型）
type DoNetSpeedSeries struct {
	CarType string    `json:"car_type"`
	Data    []float64 `json:"data"`
}

// DoNetSpeedResponse 各车型平均上传带宽响应
type DoNetSpeedResponse struct {
	BaseResponse
	Dates  []string            `json:"dates"`
	Series []*DoNetSpeedSeries `json:"series"`
}

// DoFclBwResponse FCL 整体平均上传带宽响应
type DoFclBwResponse struct {
	BaseResponse
	Dates  []string  `json:"dates"`
	Values []float64 `json:"values"`
}

// DoFailReasonItem 失败原因分析单条
type DoFailReasonItem struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

// DoFailReasonRequest 失败原因分析请求
type DoFailReasonRequest struct {
	FilterName  string `form:"filter_name"`
	EventNames  string `form:"event_names"` // 多选，逗号分隔
	ProjectName string `form:"project_name"`
	CarTypes    string `form:"car_types"` // 多选，逗号分隔
	StartDt     string `form:"start_dt"`
	EndDt       string `form:"end_dt"`
}

// AiSummaryRequest AI 看板总结请求(SSE 流式响应),参数与其他看板接口一致
type AiSummaryRequest struct {
	FilterName  string `form:"filter_name"`
	EventNames  string `form:"event_names"` // 多选，逗号分隔
	ProjectName string `form:"project_name"`
	CarTypes    string `form:"car_types"` // 多选，逗号分隔
	StartDt     string `form:"start_dt"`
	EndDt       string `form:"end_dt"`
}

// AiChatRequest AI 问答请求(SSE 流式响应);历史由前端持有并回传,后端无状态
type AiChatRequest struct {
	Question string                     `json:"question"` // 本轮提问
	Messages []AiChatHistoryMessageView `json:"messages"` // 之前的会话历史(含工具调用/确认结果)
}

// AiChatHistoryMessageView 会话历史消息(API 视图)
type AiChatHistoryMessageView struct {
	Role        string                 `json:"role"` // user / assistant
	Text        string                 `json:"text,omitempty"`
	ToolCalls   []AiChatToolCallView   `json:"tool_calls,omitempty"`
	ToolResults []AiChatToolResultView `json:"tool_results,omitempty"`
}

type AiChatToolCallView struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type AiChatToolResultView struct {
	ID string `json:"id"`
	// Content 兼容字符串或任意 JSON 对象(前端可能直接透传结构化结果)
	Content json.RawMessage `json:"content"`
	IsError bool            `json:"is_error"`
}

// StringContent 返回文本形态的 content(对象则序列化为 JSON 字符串)
func (v AiChatToolResultView) StringContent() string {
	if len(v.Content) == 0 {
		return ""
	}
	if v.Content[0] == '"' {
		var s string
		if err := json.Unmarshal(v.Content, &s); err == nil {
			return s
		}
	}
	return string(v.Content)
}

// DoFailReasonResponse 失败原因分析响应
type DoFailReasonResponse struct {
	BaseResponse
	List []*DoFailReasonItem `json:"list"`
}

// EtlStatusResponse ETL 任务日志响应
type EtlStatusResponse struct {
	BaseResponse
	List interface{} `json:"list"`
}

// FffRunningTrendRequest 算子活跃车辆趋势请求
type FffRunningTrendRequest struct {
	FilterName  string `form:"filter_name"`  // 必填：事件名，会先解析为真实 running filter_name
	StartDt     string `form:"start_dt"`     // 必填：开始日期 YYYY-MM-DD
	EndDt       string `form:"end_dt"`       // 必填：结束日期 YYYY-MM-DD
	ProjectName string `form:"project_name"` // 选填：项目名称
	CarTypes    string `form:"car_types"`    // 选填：车型，多选逗号分隔
}

// FffRunningTrendResponse 算子活跃车辆趋势响应（按天聚合）
type FffRunningTrendResponse struct {
	BaseResponse
	Dates  []string `json:"dates"`  // 日期列表 YYYY-MM-DD
	Counts []int64  `json:"counts"` // 对应日期的活跃车辆数（switch_on=1 的分组去重车辆数）
}

// DoFdrQualityResponse FDR 质量 P95 响应
type DoFdrQualityResponse struct {
	BaseResponse
	TdMbP95       float64 `json:"td_mb_p95"`        // TD 磁盘大小 P95（MB）
	TmMbP95       float64 `json:"tm_mb_p95"`        // TM 内存大小 P95（MB）
	TimeCostMsP95 float64 `json:"time_cost_ms_p95"` // 落盘耗时 P95（ms）
	FdrTotal      int64   `json:"fdr_total"`        // FDR 落盘总数
	FdrSuccess    int64   `json:"fdr_success"`      // FDR 成功数
}

// DoFclQualityResponse FCL Bag 大小质量响应
type DoFclQualityResponse struct {
	BaseResponse
	BagSizeP95  float64 `json:"bag_size_p95"` // Bag 大小 P95（字节）
	BagSizeAvg  float64 `json:"bag_size_avg"` // Bag 大小均值（字节）
	BagSizeMax  float64 `json:"bag_size_max"` // Bag 大小最大值（字节）
	UploadTotal int64   `json:"upload_total"` // 上传总数
}

// DoFdrFragmentResponse FDR 碎片率响应
type DoFdrFragmentResponse struct {
	BaseResponse
	FragmentP95 float64 `json:"fragment_p95"` // 碎片率 P95
	FragmentAvg float64 `json:"fragment_avg"` // 碎片率均值
	FragmentMax float64 `json:"fragment_max"` // 碎片率最大值
}

// DoBandwidthTopItem FDR 带宽 Top 项(按带宽字段分组)
type DoBandwidthTopItem struct {
	BandwidthField string  `json:"bandwidth_field"` // 带宽字段名
	BwSum          float64 `json:"bw_sum"`          // 带宽总和
	BwAvg          float64 `json:"bw_avg"`          // 带宽均值
	BwP95          float64 `json:"bw_p95"`          // 带宽 P95
	BwMax          float64 `json:"bw_max"`          // 带宽最大值
	SampleCount    int64   `json:"sample_count"`    // 有效样本数
}

// DoFdrBandwidthTopResponse FDR 带宽 Top 统计响应
type DoFdrBandwidthTopResponse struct {
	BaseResponse
	List []*DoBandwidthTopItem `json:"list"`
}

// FoRunningOverviewResponse 筛选器运行健康概览响应
type FoRunningOverviewResponse struct {
	BaseResponse
	RunningTotal   int64   `json:"running_total"`    // 运行记录数
	VehicleTotal   int64   `json:"vehicle_total"`    // switch_on=1 运行车辆数（ADS event_name 口径）
	SwitchOnTotal  int64   `json:"switch_on_total"`  // 开启次数
	SwitchOffTotal int64   `json:"switch_off_total"` // 关闭次数
	SwitchOnRatio  float64 `json:"switch_on_ratio"`  // 开启占比（%）
	RunningSuccess int64   `json:"running_success"`  // 运行成功数
	RunningFailed  int64   `json:"running_failed"`   // 运行失败数
	FilterCount    int64   `json:"filter_count"`     // 运行筛选器数量
}

// FoFffOverviewResponse FFF 触发概览响应
type FoFffOverviewResponse struct {
	BaseResponse
	TriggerTotal       int64   `json:"trigger_total"`        // 触发总数
	TriggerSuccess     int64   `json:"trigger_success"`      // 触发成功数
	TriggerFailed      int64   `json:"trigger_failed"`       // 触发失败数
	TriggerSuccessRate float64 `json:"trigger_success_rate"` // 触发成功率（%）
	TriggerFilterCount int64   `json:"trigger_filter_count"` // 触发筛选器去重数
	CloseFilterCount   int64   `json:"close_filter_count"`   // 关闭筛选器去重数
}
