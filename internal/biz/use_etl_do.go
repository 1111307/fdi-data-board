package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// EtlLog ETL 任务日志 DTO
type EtlLog struct {
	Dt        string    `json:"dt"`
	Target    string    `json:"target"`
	Status    string    `json:"status"`
	RunType   string    `json:"run_type"`
	Cnt       int64     `json:"cnt"`
	CostMs    int64     `json:"cost_ms"`
	ErrorMsg  string    `json:"error_msg"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// EtlRepo ETL 数据仓储接口
type EtlRepo interface {
	RunETL(ctx context.Context, dt string, runType string) (cnt int64, err error)
	IsSuccess(ctx context.Context, dt string) bool
	ListLogs(ctx context.Context, limit int) ([]*EtlLog, error)
}

// ETLUseCase ETL 业务逻辑
type ETLUseCase struct {
	repo EtlRepo
	log  *log.Helper
}

func NewETLUseCase(repo EtlRepo, logger log.Logger) *ETLUseCase {
	return &ETLUseCase{repo: repo, log: log.NewHelper(logger)}
}

// RunForDate 对指定日期执行 ETL（强制覆盖）
func (uc *ETLUseCase) RunForDate(ctx context.Context, dt string) (int64, error) {
	return uc.repo.RunETL(ctx, dt, "manual")
}

// Backfill 回流近 days 天数据，已成功的跳过
func (uc *ETLUseCase) Backfill(ctx context.Context, days int) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	today := time.Now().In(loc)
	for i := 1; i <= days; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}
		dt := today.AddDate(0, 0, -i).Format("2006-01-02")
		if uc.repo.IsSuccess(ctx, dt) {
			uc.log.Infof("[etl] backfill skip %s (already success)", dt)
			continue
		}
		cnt, err := uc.repo.RunETL(ctx, dt, "backfill")
		if err != nil {
			uc.log.Errorf("[etl] backfill %s error: %v", dt, err)
		} else {
			uc.log.Infof("[etl] backfill %s done, cnt=%d", dt, cnt)
		}
		time.Sleep(time.Second)
	}
}

// RunWindow 刷新近 days 天数据（不跳过已成功），用于每日滚动刷新
func (uc *ETLUseCase) RunWindow(ctx context.Context, days int) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	today := time.Now().In(loc)
	for i := 1; i <= days; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}
		dt := today.AddDate(0, 0, -i).Format("2006-01-02")
		cnt, err := uc.repo.RunETL(ctx, dt, "daily_window")
		if err != nil {
			uc.log.Errorf("[etl] window refresh %s error: %v", dt, err)
		} else {
			uc.log.Infof("[etl] window refresh %s done, cnt=%d", dt, cnt)
		}
		time.Sleep(time.Second)
	}
}

// ListLogs 查询最近 limit 条任务日志
func (uc *ETLUseCase) ListLogs(ctx context.Context, limit int) ([]*EtlLog, error) {
	return uc.repo.ListLogs(ctx, limit)
}
