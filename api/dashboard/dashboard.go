package dashboard_api

// BaseResponse 统一响应基础结构，实现 api.HttpResponse 接口
type BaseResponse struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
}

func (r *BaseResponse) GetCode() int32    { return r.Code }
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
	FilterName  string `form:"filter_name"`
	ProjectName string `form:"project_name"`
	StartDt     string `form:"start_dt"`
	EndDt       string `form:"end_dt"`
	Page        int    `form:"page"`
	PageSize    int    `form:"page_size"`
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
	FilterName  string `form:"filter_name"`
	EventNames  string `form:"event_names"` // 多选，逗号分隔
	ProjectName string `form:"project_name"`
	StartDt     string `form:"start_dt"`
	EndDt       string `form:"end_dt"`
	Page        int    `form:"page"`
	PageSize    int    `form:"page_size"`
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
	FilterName  string `form:"filter_name"`
	ProjectName string `form:"project_name"`
	StartDt     string `form:"start_dt"`
	EndDt       string `form:"end_dt"`
	Page        int    `form:"page"`
	PageSize    int    `form:"page_size"`
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
	FilterName  string `form:"filter_name"`
	EventNames  string `form:"event_names"` // 多选，逗号分隔
	ProjectName string `form:"project_name"`
	StartDt     string `form:"start_dt"`
	EndDt       string `form:"end_dt"`
	Page        int    `form:"page"`
	PageSize    int    `form:"page_size"`
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
	FilterName  string `form:"filter_name"`
	EventNames  string `form:"event_names"` // 多选，逗号分隔
	ProjectName string `form:"project_name"`
	StartDt     string `form:"start_dt"`
	EndDt       string `form:"end_dt"`
	Page        int    `form:"page"`
	PageSize    int    `form:"page_size"`
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
	EventNames   string `form:"event_names"`   // 多选，逗号分隔
	ProjectName  string `form:"project_name"`
	StartDt      string `form:"start_dt"`
	EndDt        string `form:"end_dt"`
	OnlyFail     int    `form:"only_fail"`     // 1=仅看失败
	StageFilter  string `form:"stage_filter"`  // fff_discard/fdr_discard/fcl_discard/fcl_success
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
	StartDt     string `form:"start_dt"`
	EndDt       string `form:"end_dt"`
}

// StageTrendResponse 三阶段触发趋势响应
type StageTrendResponse struct {
	BaseResponse
	Dates []string           `json:"dates"`
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
	ProjectName string `form:"project_name"`
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
	CarTypes    string `form:"car_types"`   // 多选，逗号分隔
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

// DoFailReasonResponse 失败原因分析响应
type DoFailReasonResponse struct {
	BaseResponse
	List []*DoFailReasonItem `json:"list"`
}
