package biz

// DoOverviewItem 事件横向对比单行（领域对象）
type DoOverviewItem struct {
	EventName    string
	VehicleCount int64
	TriggerCount int64
	CfdiRate     float64
	FffCount     int64
	FffRate      float64
	FdrCount     int64
	FdrRate      float64
	FclCount     int64
	FclRate      float64
}

// DoCoolTopItem 冷却/关闭/触发频次 Top 单条（领域对象）
type DoCoolTopItem struct {
	FilterName string
	Count      int64
}

// DoSwVersionItem 软件版本分布单条（领域对象）
type DoSwVersionItem struct {
	SwVersion string
	Count     int64
}

// DoEventTopItem FDR 内存/磁盘/Quota Top 单条（领域对象）
type DoEventTopItem struct {
	EventName string
	Count     int64
}

// DoProjectEventItem 项目触发回流事件总数单条（领域对象）
type DoProjectEventItem struct {
	ProjectName string
	EventCount  int64
}

// DoVehicleItem 车辆维度分析单条（领域对象）
type DoVehicleItem struct {
	AnonymousId  string
	CarType      string
	ProjectName  string
	TriggerCount int64
	SuccessCount int64
	CfdiRate     float64
	MainReason   string
}

// DoFailReasonItem 失败原因分析单条（领域对象）
type DoFailReasonItem struct {
	Name  string
	Value int64
}

// DoNetSpeedSeries 网速统计单条时序（领域对象）
type DoNetSpeedSeries struct {
	CarType string
	Data    []float64
}

// DoProjectCarData 项目×车型分布数据（领域对象）
type DoProjectCarData struct {
	Projects      []string
	CarTypes      []string
	Matrix        map[string][]int64
	ProjectTotals []int64
}

// DoNetSpeedData 各车型平均上传带宽数据（领域对象）
type DoNetSpeedData struct {
	Dates  []string
	Series []*DoNetSpeedSeries
}

// DoFclBwData FCL 整体平均上传带宽数据（领域对象）
type DoFclBwData struct {
	Dates  []string
	Values []float64
}

// DoActiveTrendData 活跃车辆趋势数据（领域对象）
type DoActiveTrendData struct {
	Dates  []string
	Counts []int64
}
