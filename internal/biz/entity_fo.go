package biz

// FunnelStat 数采全链路统计数字（领域对象）
type FunnelStat struct {
	FffTotal   int64
	FffAllow   int64
	FdrSuccess int64
	FdrFail    int64
	FclSuccess int64
	FclFail    int64
	CfdiRate   float64
}

// FunnelFailReason 单条失败原因（领域对象）
type FunnelFailReason struct {
	Name  string
	Count int64
}

// StageTrendSeries 单条时序数据（领域对象）
type StageTrendSeries struct {
	Name string
	Data []int64
}

// FffRunningItem 筛选器运行明细（领域对象）
type FffRunningItem struct {
	Dt              string
	FilterName      string
	AnonymousId     string
	TimestampUtc    string
	CreateAt        string
	CollectType     string
	SwVersion       string
	ProjectName     string
	CarType         string
	VehicleSource   string
	SwitchOn        bool
	Version         string
	OnAutopilot     bool
	FunctionMode    string
	Status          string
	FdiProjectName  string
	ProjectCarType  string
	VehicleSourceCn string
}

// FffTriggerItem 筛选器触发明细（领域对象）
type FffTriggerItem struct {
	Dt              string
	Uuid            string
	EventName       string
	AnonymousId     string
	TimestampUtc    string
	CreateAt        string
	TriggerTime     int64
	UtcDiffUs       int64
	Before          int
	After           int
	FilterName      string
	TriggerType     string
	CollectType     string
	Status          string
	OnAutopilot     bool
	FunctionMode    string
	SwVersion       string
	ProjectName     string
	CarType         string
	VehicleSource   string
	Bj02Lat         string
	Bj02Lon         string
	RoadType        string
	FdiProjectName  string
	ProjectCarType  string
	VehicleSourceCn string
	Tags            string
	Detail          string
}

// FffCloseItem 筛选器关闭明细（领域对象）
type FffCloseItem struct {
	Dt              string
	FilterName      string
	Version         string
	Reason          string
	AnonymousId     string
	CreateAt        string
	SwVersion       string
	TimestampUtc    string
	ProjectName     string
	CarType         string
	VehicleSource   string
	FdiProjectName  string
	ProjectCarType  string
	VehicleSourceCn string
}

// FdrTriggerItem FDR 落盘明细（领域对象）
type FdrTriggerItem struct {
	Dt                string
	Uuid              string
	EventName         string
	AnonymousId       string
	TimestampUtc      string
	CreateAt          string
	SwVersion         string
	Dse               string
	TdMb              string
	TmMb              string
	TriggerTimestamp  int64
	BeginTimestampUts int64
	EndTimestampUts   int64
	DumpTimestamp     int64
	Status            string
	Detail            string
	TimeCostMs        string
	RecordType        string
	ProjectName       string
	CarType           string
	VehicleSource     string
	FdiProjectName    string
	ProjectCarType    string
	VehicleSourceCn   string
}

// FclTriggerItem FCL 上传明细（领域对象）
type FclTriggerItem struct {
	Dt              string
	Uuid            string
	EventName       string
	AnonymousId     string
	TimestampUtc    string
	CreateAt        string
	Status          string
	Detail          string
	CompletePercent int
	LocalFile       string
	UploadFailTimes int
	PrefixStitch    string
	TriggerSource   string
	SwVersion       string
	ProjectName     string
	CarType         string
	VehicleSource   string
	FdiProjectName  string
	ProjectCarType  string
	VehicleSourceCn string
}

// UuidDetailItem 全链路明细（领域对象）
type UuidDetailItem struct {
	Dt                string
	AnonymousId       string
	EventName         string
	Uuid              string
	CreateAt          string
	FilterName        string
	FffSwVersion      string
	FdrSwVersion      string
	FclSwVersion      string
	TriggerType       string
	CollectType       string
	FffUpdatedAt      int64
	FdrUpdatedAt      int64
	FclUpdatedAt      int64
	FffStatus         string
	FdrStatus         string
	FclStatus         string
	FffDetail         string
	FdrDetail         string
	FclDetail         string
	BeginTimestampUts int64
	DumpTimestamp     int64
	EndTimestampUts   int64
	Md5               string
	BagName           string
	CompletePercent   int
	ProjectName       string
	CarType           string
	VehicleSource     string
	TimestampUtc      string
	FdiProjectName    string
	ProjectCarType    string
	VehicleSourceCn   string
	Dse               string
}

// CloseReasonItem 算子关闭原因分布（领域对象）
type CloseReasonItem struct {
	Name  string
	Value int64
}
