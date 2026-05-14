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
	EventName   string `form:"event_name"`
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
	EventName   string `form:"event_name"`
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
	EventName   string `form:"event_name"`
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

// FoDimensionsResponse FO Dashboard 下拉维度响应
type FoDimensionsResponse struct {
	BaseResponse
	FilterNames  []string `json:"filter_names"`
	EventNames   []string `json:"event_names"`
	ProjectNames []string `json:"project_names"`
}
