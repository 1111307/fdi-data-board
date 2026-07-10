package dashboard_api

// ReconcileOverviewRequest 对账总览请求
type ReconcileOverviewRequest struct {
	Date string `form:"date"`
}

// ReconcileOverviewBag L1 bag 级解码健康度
type ReconcileOverviewBag struct {
	Total             int64   `json:"total"`
	DecodeSuccess     int64   `json:"decode_success"`
	DecodeFailed      int64   `json:"decode_failed"`
	DecodePartial     int64   `json:"decode_partial"`
	DecodeSuccessRate float64 `json:"decode_success_rate"`
}

// ReconcileOverviewEventParse L2 event 级解析/发送健康度
type ReconcileOverviewEventParse struct {
	Expected       int64   `json:"expected"`
	SendFailed     int64   `json:"send_failed"`
	SendFailedRate float64 `json:"send_failed_rate"`
}

// ReconcileOverviewEventLand L3 event 级落库健康度
type ReconcileOverviewEventLand struct {
	Matched       int64   `json:"matched"`
	ConvertFailed int64   `json:"convert_failed"`
	LandingFailed int64   `json:"landing_failed"`
	Missing       int64   `json:"missing"`
	MatchRate     float64 `json:"match_rate"`
}

// ReconcileOverviewResponse 对账总览响应（三层嵌套：bag/event_parse/event_land）
type ReconcileOverviewResponse struct {
	BaseResponse
	Date         string                      `json:"date"`
	Bag          ReconcileOverviewBag        `json:"bag"`
	EventParse   ReconcileOverviewEventParse `json:"event_parse"`
	EventLand    ReconcileOverviewEventLand  `json:"event_land"`
	ExtraConsume int64                       `json:"extra_consume"`
}

// ReconcileTrendRequest 对账时间趋势请求
type ReconcileTrendRequest struct {
	StartDt string `form:"start_dt"`
	EndDt   string `form:"end_dt"`
}

// ReconcileTrendPoint 时间趋势单日数据点
type ReconcileTrendPoint struct {
	Dt            string  `json:"dt"`
	Expected      int64   `json:"expected"`
	Matched       int64   `json:"matched"`
	ConvertFailed int64   `json:"convert_failed"`
	LandingFailed int64   `json:"landing_failed"`
	Missing       int64   `json:"missing"`
	MatchRate     float64 `json:"match_rate"`
}

// ReconcileTrendResponse 对账时间趋势响应
type ReconcileTrendResponse struct {
	BaseResponse
	Start  string                 `json:"start"`
	End    string                 `json:"end"`
	Points []*ReconcileTrendPoint `json:"points"`
}

// ReconcileModuleRequest 对账模块维度请求
type ReconcileModuleRequest struct {
	Date    string `form:"date"`
	Project string `form:"project"` // 可选，交叉钻取到项目内模块分布
}

// ReconcileModuleItem 单个模块对账统计
type ReconcileModuleItem struct {
	ModuleName    string  `json:"module_name"`
	Expected      int64   `json:"expected"`
	Matched       int64   `json:"matched"`
	ConvertFailed int64   `json:"convert_failed"`
	LandingFailed int64   `json:"landing_failed"`
	Missing       int64   `json:"missing"`
	SendFailed    int64   `json:"send_failed"`
	MatchRate     float64 `json:"match_rate"`
}

// ReconcileModuleResponse 对账模块维度响应
type ReconcileModuleResponse struct {
	BaseResponse
	Date    string                 `json:"date"`
	Project string                 `json:"project,omitempty"`
	Modules []*ReconcileModuleItem `json:"modules"`
}

// ReconcileProjectRequest 对账项目维度请求
type ReconcileProjectRequest struct {
	Date    string `form:"date"`
	OrderBy string `form:"order_by"` // missing(默认) / expected
	Limit   int    `form:"limit"`
}

// ReconcileProjectItem 单个项目对账统计
type ReconcileProjectItem struct {
	Project       string  `json:"project"`
	Expected      int64   `json:"expected"`
	Matched       int64   `json:"matched"`
	ConvertFailed int64   `json:"convert_failed"`
	LandingFailed int64   `json:"landing_failed"`
	Missing       int64   `json:"missing"`
	MatchRate     float64 `json:"match_rate"`
}

// ReconcileProjectResponse 对账项目维度响应
type ReconcileProjectResponse struct {
	BaseResponse
	Date     string                  `json:"date"`
	Projects []*ReconcileProjectItem `json:"projects"`
}

// ReconcileDecodeStatusRequest 上游解码状态分布请求
type ReconcileDecodeStatusRequest struct {
	Date string `form:"date"`
}

// ReconcileDecodeStatusItem 解码状态分布单条
type ReconcileDecodeStatusItem struct {
	Status int8  `json:"status"`
	Stage  int8  `json:"stage"`
	Count  int64 `json:"count"`
}

// ReconcileDecodeStatusResponse 上游解码状态分布响应
type ReconcileDecodeStatusResponse struct {
	BaseResponse
	Date  string                       `json:"date"`
	Items []*ReconcileDecodeStatusItem `json:"items"`
}

// ReconcileMd5ListRequest 差异 md5 列表请求
type ReconcileMd5ListRequest struct {
	Date  string `form:"date"`
	Type  string `form:"type"` // missing / convert_failed / landing_failed / decode_failed
	Limit int    `form:"limit"`
}

// ReconcileMd5Item 差异 md5 单条，字段按 type 不同部分为空
type ReconcileMd5Item struct {
	Md5             string `json:"md5"`
	ModuleName      string `json:"module_name,omitempty"`
	Count           int64  `json:"count,omitempty"`
	Status          int8   `json:"status,omitempty"`
	Stage           int8   `json:"stage,omitempty"`
	ErrorMsg        string `json:"error_msg,omitempty"`
	ParsedLineCount int    `json:"parsed_line_count,omitempty"`
}

// ReconcileMd5ListResponse 差异 md5 列表响应
type ReconcileMd5ListResponse struct {
	BaseResponse
	Date  string              `json:"date"`
	Type  string              `json:"type"`
	Items []*ReconcileMd5Item `json:"items"`
}

// ReconcileMd5DetailRequest 单 md5 event 级明细请求
type ReconcileMd5DetailRequest struct {
	Date string `form:"date"`
	Md5  string `form:"md5"`
}

// ReconcileMd5DetailItem 单 md5 下单条 event 明细
type ReconcileMd5DetailItem struct {
	Uuid          string `json:"uuid"`
	ModuleName    string `json:"module_name"`
	SendStatus    int8   `json:"send_status"`
	ErrDetail     string `json:"err_detail"`
	Consumed      bool   `json:"consumed"`
	ConsumeStatus int8   `json:"consume_status"`
	ConsumeErr    string `json:"consume_err"`
}

// ReconcileMd5DetailResponse 单 md5 event 级明细响应
type ReconcileMd5DetailResponse struct {
	BaseResponse
	Md5   string                    `json:"md5"`
	Date  string                    `json:"date"`
	Items []*ReconcileMd5DetailItem `json:"items"`
}

// ReconcileEventListRequest event/uuid 级明细列表请求，不依赖先选 md5
type ReconcileEventListRequest struct {
	Date       string `form:"date"`
	ModuleName string `form:"module_name"`
	Project    string `form:"project"`
	Type       string `form:"type"` // missing / extra / send_failed / convert_failed / landing_failed / mismatched
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
}

// ReconcileEventItem event/uuid 级明细单条，字段按 type 不同部分为空
type ReconcileEventItem struct {
	Md5            string `json:"md5"`
	Uuid           string `json:"uuid"`
	ModuleName     string `json:"module_name,omitempty"`
	Project        string `json:"project,omitempty"`
	Status         int8   `json:"status,omitempty"`
	ErrDetail      string `json:"err_detail,omitempty"`
	DetailModule   string `json:"detail_module,omitempty"`
	ConsumeModule  string `json:"consume_module,omitempty"`
	DetailProject  string `json:"detail_project,omitempty"`
	ConsumeProject string `json:"consume_project,omitempty"`
}

// ReconcileEventListResponse event/uuid 级明细列表响应
type ReconcileEventListResponse struct {
	BaseResponse
	Date     string                `json:"date"`
	Type     string                `json:"type"`
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
	Items    []*ReconcileEventItem `json:"items"`
}

// ReconcileRecordConsistencyRequest record↔detail 内部一致性请求
type ReconcileRecordConsistencyRequest struct {
	Date  string `form:"date"`
	Limit int    `form:"limit"`
}

// ReconcileRecordConsistencyItem record↔detail 一致性单条（解码阶段是否丢行）
type ReconcileRecordConsistencyItem struct {
	Md5             string `json:"md5"`
	ParsedLineCount int    `json:"parsed_line_count"`
	DetailCount     int64  `json:"detail_count"`
	Diff            int    `json:"diff"`
}

// ReconcileRecordConsistencyResponse record↔detail 内部一致性响应
type ReconcileRecordConsistencyResponse struct {
	BaseResponse
	Date  string                            `json:"date"`
	Items []*ReconcileRecordConsistencyItem `json:"items"`
}

// ReconcileUuidSourceRequest uuid 来源细分请求
type ReconcileUuidSourceRequest struct {
	Date string `form:"date"`
}

// ReconcileUuidSourceItem uuid 来源细分单条，match_rate 在 expected=0 时为 null
type ReconcileUuidSourceItem struct {
	UuidSource string   `json:"uuid_source"`
	Expected   int64    `json:"expected"`
	Matched    int64    `json:"matched"`
	Missing    int64    `json:"missing"`
	MatchRate  *float64 `json:"match_rate"`
}

// ReconcileUuidSourceResponse uuid 来源细分响应
type ReconcileUuidSourceResponse struct {
	BaseResponse
	Date    string                     `json:"date"`
	Sources []*ReconcileUuidSourceItem `json:"sources"`
}

// ReconcileFailureSummaryRequest 失败汇总请求
type ReconcileFailureSummaryRequest struct {
	Date string `form:"date"`
}

// ReconcileDecodeFailedItem L1 解码失败按 stage 聚合单条
type ReconcileDecodeFailedItem struct {
	Stage          int8   `json:"stage"`
	StageDesc      string `json:"stage_desc"`
	Count          int64  `json:"count"`
	SampleErrorMsg string `json:"sample_error_msg"`
}

// ReconcileSendFailedItem L2 发送下游失败按 module_name+err_detail 聚合单条
type ReconcileSendFailedItem struct {
	ModuleName string `json:"module_name"`
	ErrDetail  string `json:"err_detail"`
	Count      int64  `json:"count"`
}

// ReconcileConvertFailedItem L3 转换失败按 module_name+err_detail 聚合单条
type ReconcileConvertFailedItem struct {
	ModuleName string `json:"module_name"`
	ErrDetail  string `json:"err_detail"`
	Count      int64  `json:"count"`
}

// ReconcileLandingFailedItem L3 落库失败按 module_name+err_detail 聚合单条
type ReconcileLandingFailedItem struct {
	ModuleName string `json:"module_name"`
	ErrDetail  string `json:"err_detail"`
	Count      int64  `json:"count"`
}

// ReconcileFailureSummaryResponse 失败汇总响应
type ReconcileFailureSummaryResponse struct {
	BaseResponse
	Date          string                        `json:"date"`
	DecodeFailed  []*ReconcileDecodeFailedItem  `json:"decode_failed"`
	SendFailed    []*ReconcileSendFailedItem    `json:"send_failed"`
	ConvertFailed []*ReconcileConvertFailedItem `json:"convert_failed"`
	LandingFailed []*ReconcileLandingFailedItem `json:"landing_failed"`
}
