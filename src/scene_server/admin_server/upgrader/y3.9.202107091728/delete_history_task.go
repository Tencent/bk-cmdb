package y3_9_202107091728

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/metadata"
	"configcenter/src/scene_server/admin_server/upgrader"
	"configcenter/src/storage/dal"
)

// deleteTaskInvalidHistory delete task which settemplate or set has been removed
func deleteTaskInvalidHistory(ctx context.Context, db dal.RDB, conf *upgrader.Config) error {
	cond := mapstr.New()
	cond.Set(common.BKSetTemplateIDField, map[string]interface{}{common.BKDBNE: 0})
	var setIDs []metadata.SetInst
	err := db.Table(common.BKTableNameBaseSet).Find(cond).Fields(common.BKSetIDField).All(ctx, &setIDs)
	if err != nil {
		blog.Errorf("get set id failed, tablename: %s, err: %s", common.BKTableNameBaseSet, err.Error())
		return err
	}

	flags := []string{"set_template_sync"}
	for _, setID := range setIDs {
		flags = append(flags, fmt.Sprintf("set_template_sync:%d", setID.SetID))
	}

	deleteCond := mapstr.New()
	deleteCond.Set("flag", map[string]interface{}{common.BKDBNIN: flags})
	err = db.Table(common.BKTableNameAPITask).Delete(ctx, deleteCond)
	if err != nil {
		blog.Errorf("delete invalid task history failed, tablename: %s, err: %s",
			common.BKTableNameAPITask, err.Error())
		return err
	}

	for _, flag := range flags {
		if flag == "set_template_sync" {
			continue
		}

		split := strings.Split(flag, ":")
		if len(split) != 2 {
			continue
		}
		instID, err := strconv.ParseInt(split[1], 10, 64)
		if err != nil {
			blog.Errorf("change string to int64 failed, value: %s, type: %T, err: %s",
				split[1], split[1], err.Error())
			return err
		}
		filter := map[string]interface{}{"flag": flag}
		flagDoc := mapstr.MapStr{
			"flag":       split[0],
			"bk_inst_id": instID,
		}
		if err = db.Table(common.BKTableNameAPITask).Update(ctx, filter, flagDoc); err != nil {
			blog.Errorf("upsert task flag failed, tablename: %s, err: %s", common.BKTableNameAPITask, err.Error())
			return err
		}
	}

	return nil
}
