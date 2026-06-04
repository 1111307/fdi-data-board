package data

import (
	"context"
	"errors"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"

	"fdi_data_board/internal/biz"
	"fdi_data_board/internal/data/orm"
)

const targetCfdiDaily = "ads_do_cfdi_daily"

const deleteCfdiDailySQL = `DELETE FROM fdi.ads_do_cfdi_daily WHERE dt = ?`
const insertCfdiDailySQL = `
INSERT INTO fdi.ads_do_cfdi_daily
SELECT
    dt,
    COALESCE(event_name, '')     AS event_name,
    COALESCE(filter_name, '')    AS filter_name,
    COALESCE(car_type, '')       AS car_type,
    COALESCE(project_name, '')   AS project_name,
    COALESCE(fff_sw_version, '') AS fff_sw_version,
    COALESCE(fff_status, '')     AS fff_status,
    COALESCE(fdr_status, '')     AS fdr_status,
    COALESCE(fcl_status, '')     AS fcl_status,
    CASE
        WHEN fff_detail = 'check_is_no_need_cooldown'                             THEN 'cooldown'
        WHEN fff_detail IN ('check_drm_quota', 'check_drm_quota_weight')          THEN 'drm_quota'
        WHEN fff_detail = 'check_need_acquire_data'                               THEN 'acquire_data'
        WHEN fff_detail = 'check_not_reach_trigger_maximum'                       THEN 'trigger_maximum'
        WHEN fff_detail LIKE 'Bag invalid:%'                                      THEN 'bag_invalid'
        WHEN fff_detail LIKE 'event_name do not recognized%'                      THEN 'event_not_recognized'
        WHEN fff_detail LIKE 'tls%'                                               THEN 'tls_error'
        WHEN fff_detail = 'query cloud DISCARD, detail:Filter quota exceeded'     THEN 'quota_exceeded'
        WHEN fff_detail = 'query cloud DISCARD, detail:EventName is in blacklist' THEN 'event_in_blacklist'
        WHEN fff_detail IS NULL OR fff_detail = ''                                THEN ''
        ELSE 'other'
    END AS fff_detail_tag,
    CASE
        WHEN fdr_detail IN ('because of full gc', 'mem pool water line')           THEN 'memory'
        WHEN fdr_detail IN ('Disk overrun', 'Exceeds the maximum number of files') THEN 'disk'
        WHEN fdr_detail LIKE 'Bag invalid:%'                                       THEN 'bag_invalid'
        WHEN fdr_detail LIKE 'Dump bag dir missing%'                               THEN 'bag_dir_missing'
        WHEN fdr_detail LIKE 'event_name do not recognized%'                       THEN 'event_not_recognized'
        WHEN fdr_detail = 'unauthorized'                                           THEN 'unauthorized'
        WHEN fdr_detail IS NULL OR fdr_detail = ''                                 THEN ''
        ELSE 'other'
    END AS fdr_detail_tag,
    CASE
        WHEN fcl_detail = 'query cloud DISCARD, detail:Filter quota exceeded'     THEN 'quota_exceeded'
        WHEN fcl_detail = 'reach upload limit'                                    THEN 'reach_upload_limit'
        WHEN fcl_detail = 'query cloud DISCARD, detail:EventName is in blacklist' THEN 'event_in_blacklist'
        WHEN fcl_detail LIKE 'geofence forbidden%' OR fcl_detail = 'unexpected geofence cause' THEN 'geofence_error'
        WHEN fcl_detail LIKE 'tls%'                                               THEN 'tls_error'
        WHEN fcl_detail IN ('bag not exist', 'meta file lost', 'meta file empty') THEN 'bag_missing'
        WHEN fcl_detail IN ('unexpected bag_upload_query cause',
                            's3 upload force quit')                               THEN 'upload_error'
        WHEN fcl_detail IN ('create socket failed', 'http request failed',
                            'transfer dns failed')                                THEN 'network_error'
        WHEN fcl_detail IS NULL OR fcl_detail = ''                                THEN ''
        ELSE 'other'
    END AS fcl_detail_tag,
    COUNT(*) AS cnt
FROM fdi.dwd_cfdi_status_monitor_analysis
WHERE dt = ? AND event_name != 'Forever_log'
GROUP BY
    dt, event_name, filter_name, car_type, project_name,
    fff_sw_version, fff_status, fdr_status, fcl_status,
    fff_detail_tag, fdr_detail_tag, fcl_detail_tag`

var _ biz.EtlRepo = (*etlRepo)(nil)

type etlRepo struct {
	*baseRepo
	log *log.Helper
}

func NewEtlRepo(data *Data, logger log.Logger) biz.EtlRepo {
	return &etlRepo{
		baseRepo: &baseRepo{data: data},
		log:      log.NewHelper(logger),
	}
}

func (r *etlRepo) RunETL(ctx context.Context, dt string, runType string) (cnt int64, err error) {
	doris := r.dorisDB(ctx)
	start := time.Now()

	r.upsertLog(dt, "running", runType, 0, 0, "")

	defer func() {
		cost := time.Since(start).Milliseconds()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				r.upsertLog(dt, "cancelled", runType, 0, cost, err.Error())
			} else {
				r.upsertLog(dt, "failed", runType, 0, cost, err.Error())
			}
		} else {
			r.upsertLog(dt, "success", runType, cnt, cost, "")
		}
	}()

	if err = doris.Exec(deleteCfdiDailySQL, dt).Error; err != nil {
		return
	}
	if err = doris.Exec(insertCfdiDailySQL, dt).Error; err != nil {
		return
	}

	if err = doris.Raw(`SELECT COUNT(1) FROM fdi.ads_do_cfdi_daily WHERE dt = ?`, dt).Scan(&cnt).Error; err != nil {
		return
	}
	return
}

func (r *etlRepo) IsSuccess(ctx context.Context, dt string) (bool, error) {
	var record orm.EtlJobLogDo
	err := r.mysqlDB(ctx).
		Where("dt = ? AND table_name = ? AND status = 'success'", dt, targetCfdiDaily).
		First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *etlRepo) ListLogs(ctx context.Context, limit int) ([]*biz.EtlLog, error) {
	var list []*orm.EtlJobLogDo
	err := r.mysqlDB(ctx).
		Where("table_name = ?", targetCfdiDaily).
		Order("dt DESC").
		Limit(limit).
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	result := make([]*biz.EtlLog, len(list))
	for i, v := range list {
		result[i] = &biz.EtlLog{
			Dt:        v.Dt,
			Target:    v.Target,
			Status:    v.Status,
			RunType:   v.RunType,
			Cnt:       v.Cnt,
			CostMs:    v.CostMs,
			ErrorMsg:  v.ErrorMsg,
			CreatedAt: v.CreatedAt,
			UpdatedAt: v.UpdatedAt,
		}
	}
	return result, nil
}

func (r *etlRepo) upsertLog(dt, status, runType string, cnt, costMs int64, errMsg string) {
	if status == "running" {
		if err := r.data.mysqlDB.Exec(`
			INSERT INTO etl_job_log (dt, table_name, status, run_type, cnt, cost_ms, error_msg, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, NOW())
			ON DUPLICATE KEY UPDATE
				status    = VALUES(status),
				run_type  = VALUES(run_type),
				updated_at = NOW()`,
			dt, targetCfdiDaily, status, runType, cnt, costMs, errMsg).Error; err != nil {
			r.log.Errorf("[etl] upsertLog(running) failed: dt=%s err=%v", dt, err)
		}
	} else {
		if err := r.data.mysqlDB.Exec(`
			UPDATE etl_job_log SET
				status    = ?,
				run_type  = ?,
				cnt       = ?,
				cost_ms   = ?,
				error_msg = ?,
				updated_at = NOW()
			WHERE dt = ? AND table_name = ?`,
			status, runType, cnt, costMs, errMsg, dt, targetCfdiDaily).Error; err != nil {
			r.log.Errorf("[etl] upsertLog(%s) failed: dt=%s err=%v", status, dt, err)
		}
	}
}
