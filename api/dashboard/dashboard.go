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
