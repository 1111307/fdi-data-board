package server

import (
	"context"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	"fdi_data_board/internal/biz"
)

// etlRefreshDays 每日定时任务滚动刷新的天数窗口
const etlRefreshDays = 7

type ETLServer struct {
	uc     *biz.ETLUseCase
	log    *log.Helper
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewETLServer(uc *biz.ETLUseCase, logger log.Logger) *ETLServer {
	return &ETLServer{uc: uc, log: log.NewHelper(logger)}
}

func (s *ETLServer) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.log.Info("[etl] server starting")

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		// 第一次启动时回填近180天数据，后续每天定时刷新近7天数据
		s.log.Info("[etl] starting backfill for 180 days")
		s.uc.Backfill(ctx, 180)
		s.log.Info("[etl] backfill done")
		for {
			next := nextRunTime("Asia/Shanghai", 2, 0)
			s.log.Infof("[etl] next scheduled run at %s", next.Format(time.RFC3339))

			select {
			case <-ctx.Done():
				s.log.Info("[etl] scheduler stopped")
				return
			case <-time.After(time.Until(next)):
				s.log.Infof("[etl] daily window refresh, last %d days", etlRefreshDays)
				s.uc.RunWindow(ctx, etlRefreshDays)
			}
		}
	}()

	return nil
}

func (s *ETLServer) Stop(_ context.Context) error {
	s.log.Info("[etl] server stopping")
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
	return nil
}

// nextRunTime 返回指定时区 hour:minute 的下一次触发时刻
func nextRunTime(timezone string, hour, minute int) time.Time {
	loc, _ := time.LoadLocation(timezone)
	now := time.Now().In(loc)
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}
