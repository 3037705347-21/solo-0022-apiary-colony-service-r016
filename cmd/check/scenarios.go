package main

import (
	"fmt"
	"strings"
)

const (
	demoApiaryID = "apiary-north"
	demoHiveID   = "hive-01"
)

func seedApiary(c *client, id, name, district, keeper string) error {
	_, err := c.expect("POST", "/apiaries", map[string]any{
		"id":       id,
		"name":     name,
		"district": district,
		"keeper":   keeper,
	}, 201, `"id":"`+id+`"`)
	return err
}

func seedHive(c *client, apiaryID, hiveID, model string) error {
	_, err := c.expect("POST", "/apiaries/"+apiaryID+"/hives", map[string]any{
		"id":    hiveID,
		"model": model,
	}, 201, `"state":"colonized"`)
	return err
}

func seedDefault(c *client) error {
	if err := seedApiary(c, demoApiaryID, "北岭蜂场", "北岭片区", "周岚"); err != nil {
		return err
	}
	return seedHive(c, demoApiaryID, demoHiveID, "langstroth")
}

func checkRegisterApiary(c *client) error {
	if _, err := c.expect("POST", "/apiaries", map[string]any{
		"id": "apiary-north", "name": "北岭蜂场", "district": "北岭片区", "keeper": "周岚",
	}, 201, `"id":"apiary-north"`, `"name":"北岭蜂场"`); err != nil {
		return err
	}
	if _, err := c.expect("GET", "/apiaries/apiary-north", nil, 200,
		`"district":"北岭片区"`, `"keeper":"周岚"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/apiaries", map[string]any{
		"id": "apiary-north", "name": "重复蜂场", "district": "北岭片区", "keeper": "周岚",
	}, 409, `"code":"conflict"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/apiaries", map[string]any{
		"id": "apiary-south", "name": "", "district": "南坡片区", "keeper": "李澈",
	}, 400, `"code":"invalid_request"`); err != nil {
		return err
	}
	return nil
}

func checkHiveInstall(c *client) error {
	if err := seedApiary(c, demoApiaryID, "北岭蜂场", "北岭片区", "周岚"); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/apiaries/apiary-north/hives", map[string]any{
		"id": "hive-01", "model": "langstroth",
	}, 201, `"state":"colonized"`, `"apiaryId":"apiary-north"`); err != nil {
		return err
	}
	if _, err := c.expect("GET", "/hives/hive-01", nil, 200, `"model":"langstroth"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/apiaries/apiary-north/hives", map[string]any{
		"id": "hive-01", "model": "warre",
	}, 409, `"code":"conflict"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/apiaries/apiary-missing/hives", map[string]any{
		"id": "hive-09", "model": "warre",
	}, 404, `"code":"not_found"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/apiaries/apiary-north/hives", map[string]any{
		"id": "hive-02", "model": "dadant",
	}, 400, `"code":"invalid_request"`); err != nil {
		return err
	}
	return nil
}

func checkRecordInspection(c *client) error {
	if err := seedDefault(c); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt": "2026-04-10T09:00:00Z", "colonyStrength": 9, "strengthUnit": "grade",
		"broodPattern": "patchy", "temper": "calm", "miteCount": 3, "miteUnit": "per100", "note": "",
	}, 400, `"code":"invalid_request"`); err != nil {
		return err
	}
	// 未携带单位不能被静默接受。
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt": "2026-04-10T09:00:00Z", "colonyStrength": 3,
		"broodPattern": "patchy", "temper": "calm", "miteCount": 3, "miteUnit": "per100", "note": "",
	}, 400, `"code":"invalid_request"`, `"strengthUnit"`); err != nil {
		return err
	}
	// 未知单位与跨维度不兼容单位都必须显式拒绝。
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt": "2026-04-10T09:00:00Z", "colonyStrength": 3, "strengthUnit": "basket",
		"broodPattern": "patchy", "temper": "calm", "miteCount": 3, "miteUnit": "per100", "note": "",
	}, 400, `"code":"invalid_request"`, `"strengthUnit"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt": "2026-04-10T09:00:00Z", "colonyStrength": 3, "strengthUnit": "per100",
		"broodPattern": "patchy", "temper": "calm", "miteCount": 3, "miteUnit": "grade", "note": "",
	}, 400, `"code":"invalid_request"`, `"strengthUnit"`); err != nil {
		return err
	}
	// 非规范单位原始值越界同样拒绝：frame 限 1..10，per300 限 0..180。
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt": "2026-04-10T09:00:00Z", "colonyStrength": 11, "strengthUnit": "frame",
		"broodPattern": "patchy", "temper": "calm", "miteCount": 3, "miteUnit": "per100", "note": "",
	}, 400, `"code":"invalid_request"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt": "2026-04-10T09:00:00Z", "colonyStrength": 3, "strengthUnit": "grade",
		"broodPattern": "patchy", "temper": "calm", "miteCount": 200, "miteUnit": "per300", "note": "",
	}, 400, `"code":"invalid_request"`); err != nil {
		return err
	}
	// 非规范口径在接收时换算并固化：frame 3 -> grade 2，per300 30 -> per100 10，
	// 关注项基于换算后的口径派生（mite_load 与 low_strength）。
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt": "2026-04-10T08:00:00Z", "colonyStrength": 3, "strengthUnit": "frame",
		"broodPattern": "patchy", "temper": "calm", "miteCount": 30, "miteUnit": "per300", "note": "框数与三百只抽样",
	}, 201, `"inspection-000001"`,
		`"rawValue":3`, `"rawUnit":"frame"`, `"colonyStrength":2`, `"strengthUnit":"grade"`,
		`"rawUnit":"per300"`, `"miteCount":10`, `"miteUnit":"per100"`, `"rulesVersion":"v1"`,
		`"mite_load"`, `"low_strength"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt": "2026-04-10T09:00:00Z", "colonyStrength": 3, "strengthUnit": "grade", "broodPattern": "patchy",
		"temper": "aggressive", "miteCount": 12, "miteUnit": "per100", "note": "西侧巢门观察",
	}, 201, `"inspection-000002"`, `"mite_load"`, `"temper_risk"`); err != nil {
		return err
	}
	if _, err := c.expect("GET", "/hives/hive-01/inspections", nil, 200,
		`"note":"框数与三百只抽样"`, `"rawUnit":"frame"`, `"note":"西侧巢门观察"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/state", map[string]any{"target": "retired"}, 200,
		`"to":"retired"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt": "2026-04-11T09:00:00Z", "colonyStrength": 3, "strengthUnit": "grade", "broodPattern": "solid",
		"temper": "calm", "miteCount": 1, "miteUnit": "per100", "note": "",
	}, 409, `"code":"invalid_state"`); err != nil {
		return err
	}
	return nil
}

func checkReplaceQueen(c *client) error {
	if err := seedDefault(c); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/queens", map[string]any{
		"origin": "bred", "introducedAt": "2026-04-02T08:00:00Z", "reason": "aging",
	}, 201, `"queen-000001"`, `"markColor":"white"`, `"state":"laying"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/queens", map[string]any{
		"origin": "cells", "introducedAt": "2026-04-20T08:00:00Z", "reason": "aggression",
	}, 201, `"queen-000002"`, `"markColor":"white"`); err != nil {
		return err
	}
	if _, err := c.expect("GET", "/hives/hive-01/queens", nil, 200,
		`"queen-000001"`, `"superseded"`, `"queen-000002"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/queens", map[string]any{
		"origin": "imported", "introducedAt": "2026-05-01T08:00:00Z", "reason": "aging",
	}, 400, `"code":"invalid_request"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/state", map[string]any{"target": "retired"}, 200,
		`"to":"retired"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/queens", map[string]any{
		"origin": "bred", "introducedAt": "2026-05-02T08:00:00Z", "reason": "loss",
	}, 409, `"code":"invalid_state"`); err != nil {
		return err
	}
	return nil
}

func checkAdvanceHiveState(c *client) error {
	if err := seedDefault(c); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/state", map[string]any{"target": "production"}, 200,
		`"from":"colonized"`, `"to":"production"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/state", map[string]any{"target": "colonized"}, 409,
		`"code":"invalid_state"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/state", map[string]any{"target": "production"}, 409,
		`"code":"invalid_state"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/state", map[string]any{"target": "retired"}, 200,
		`"to":"retired"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/state", map[string]any{"target": "production"}, 409,
		`"code":"invalid_state"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-missing/state", map[string]any{"target": "production"}, 404,
		`"code":"not_found"`); err != nil {
		return err
	}
	return nil
}

func checkReadApiaryStatus(c *client) error {
	if err := seedApiary(c, demoApiaryID, "北岭蜂场", "北岭片区", "周岚"); err != nil {
		return err
	}
	for _, hiveID := range []string{"hive-01", "hive-02", "hive-03"} {
		if err := seedHive(c, demoApiaryID, hiveID, "langstroth"); err != nil {
			return err
		}
	}
	if _, err := c.expect("POST", "/hives/hive-02/state", map[string]any{"target": "production"}, 200,
		`"to":"production"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt": "2026-04-10T09:00:00Z", "colonyStrength": 8, "strengthUnit": "frame",
		"broodPattern": "solid", "temper": "calm", "miteCount": 36, "miteUnit": "per300", "note": "",
	}, 201, `"mite_load"`, `"colonyStrength":4`, `"strengthUnit":"grade"`, `"miteCount":12`, `"miteUnit":"per100"`); err != nil {
		return err
	}
	report, err := c.expect("GET", "/apiaries/apiary-north/status?asOf=2026-04-12T00:00:00Z", nil, 200,
		`"hiveCount":3`, `"colonized":2`, `"production":1`, `"layingQueens":0`)
	if err != nil {
		return err
	}
	for _, fragment := range []string{
		`"needsInspection":["hive-02","hive-03"]`,
		`"concernHives":["hive-01"]`,
		// 汇总呈现必须与巡检记录使用同一规范口径，且取自已保存的换算结果。
		`"latestMeasurements":{"colonyStrength":4,"strengthUnit":"grade","miteCount":12,"miteUnit":"per100"}`,
	} {
		if !strings.Contains(report.text(), fragment) {
			return fmt.Errorf("状态汇总缺少 %q：%s", fragment, report.text())
		}
	}
	if err := report.mustOrder(`"hiveId":"hive-01"`, `"hiveId":"hive-02"`); err != nil {
		return err
	}
	if _, err := c.expect("GET", "/apiaries/apiary-missing/status", nil, 404, `"code":"not_found"`); err != nil {
		return err
	}
	if _, err := c.expect("GET", "/apiaries/apiary-north/status?asOf=not-a-time", nil, 400,
		`"code":"invalid_request"`); err != nil {
		return err
	}
	return nil
}
