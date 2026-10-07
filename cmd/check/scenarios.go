package main

import (
	"encoding/json"
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
		"observedAt": "2026-04-10T09:00:00Z", "colonyStrength": 9, "broodPattern": "patchy",
		"temper": "calm", "miteCount": 3, "note": "",
	}, 400, `"code":"invalid_request"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt": "2026-04-10T09:00:00Z", "colonyStrength": 3, "broodPattern": "patchy",
		"temper": "aggressive", "miteCount": 12, "note": "西侧巢门观察",
	}, 201, `"inspection-000001"`, `"mite_load"`, `"temper_risk"`); err != nil {
		return err
	}
	if _, err := c.expect("GET", "/hives/hive-01/inspections", nil, 200, `"note":"西侧巢门观察"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/state", map[string]any{"target": "retired"}, 200,
		`"to":"retired"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt": "2026-04-11T09:00:00Z", "colonyStrength": 3, "broodPattern": "solid",
		"temper": "calm", "miteCount": 1, "note": "",
	}, 409, `"code":"invalid_state"`); err != nil {
		return err
	}
	return nil
}

func checkMeasurementUnits(c *client) error {
	if err := seedDefault(c); err != nil {
		return err
	}
	// 首次巡检以十分制群势、三百只样本螨计数提交，蜂箱展示口径随之钉住。
	// 7/10 换算到五分制为 4，螨计数 30/300 换算为 10/100：仅触发 mite_load。
	first, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt":     "2026-04-10T09:00:00Z",
		"colonyStrength": 7, "colonyStrengthUnit": "score10",
		"broodPattern": "patchy", "temper": "calm",
		"miteCount": 30, "miteCountUnit": "per300", "note": "跨岗位协作记录",
	}, 201,
		`"inspection-000001"`,
		`"colonyStrength":8,"colonyStrengthUnit":"score10"`,
		`"rawColonyStrength":7,"rawColonyStrengthUnit":"score10"`,
		`"miteCount":30,"miteCountUnit":"per300"`,
		`"rawMiteCount":30,"rawMiteCountUnit":"per300"`,
		`"conversionVersion":"unit-rules-v1"`,
		`"mite_load"`,
	)
	if err != nil {
		return err
	}
	if strings.Contains(first.text(), "low_strength") {
		return fmt.Errorf("换算后群势为 4，不应派生 low_strength：%s", first.text())
	}
	// 第二次巡检缺省按标准口径提交：原始读数留痕为标准口径，
	// 但呈现仍换算到蜂箱已钉住的 score10/per300；关注项基于标准口径派生。
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt":     "2026-04-12T09:00:00Z",
		"colonyStrength": 2, "broodPattern": "solid", "temper": "calm",
		"miteCount": 9, "note": "",
	}, 201,
		`"colonyStrength":4,"colonyStrengthUnit":"score10"`,
		`"rawColonyStrength":2,"rawColonyStrengthUnit":"scale5"`,
		`"miteCount":27,"miteCountUnit":"per300"`,
		`"rawMiteCount":9,"rawMiteCountUnit":"per100"`,
		`"mite_load"`, `"low_strength"`,
	); err != nil {
		return err
	}
	// 未知单位不能被静默接受。
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt":     "2026-04-13T09:00:00Z",
		"colonyStrength": 3, "miteCount": 3, "miteCountUnit": "per1000",
		"broodPattern": "solid", "temper": "calm", "note": "",
	}, 400, `"code":"invalid_request"`, `"field":"miteCountUnit"`); err != nil {
		return err
	}
	// 维度不兼容的单位必须拒绝，不能当成同一口径比较。
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt":     "2026-04-13T09:00:00Z",
		"colonyStrength": 3, "colonyStrengthUnit": "per100",
		"miteCount": 3, "broodPattern": "solid", "temper": "calm", "note": "",
	}, 400, `"code":"invalid_request"`, `"field":"colonyStrengthUnit"`); err != nil {
		return err
	}
	// 原始读数超出提交单位自身范围也要拒绝。
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt":     "2026-04-13T09:00:00Z",
		"colonyStrength": 11, "colonyStrengthUnit": "score10",
		"miteCount": 0, "broodPattern": "solid", "temper": "calm", "note": "",
	}, 400, `"code":"invalid_request"`, `"field":"colonyStrength"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt":     "2026-04-13T09:00:00Z",
		"colonyStrength": 3, "miteCount": 181, "miteCountUnit": "per300",
		"broodPattern": "solid", "temper": "calm", "note": "",
	}, 400, `"code":"invalid_request"`, `"field":"miteCount"`); err != nil {
		return err
	}
	// 状态汇总中的量测呈现必须与巡检列表使用同一钉住口径，且取最近一次巡检。
	if _, err := c.expect("GET", "/apiaries/apiary-north/status?asOf=2026-04-13T00:00:00Z", nil, 200,
		`"latestMeasurement":{"colonyStrength":4,"colonyStrengthUnit":"score10","miteCount":27,"miteCountUnit":"per300"}`,
		`"concernHives":["hive-01"]`,
	); err != nil {
		return err
	}
	// 历史巡检在保存时即固化：换算结果、原始读数与关注项不随后续提交变化。
	list, err := c.expect("GET", "/hives/hive-01/inspections", nil, 200)
	if err != nil {
		return err
	}
	var records []struct {
		ID       string   `json:"id"`
		Concerns []string `json:"concerns"`
	}
	if err := json.Unmarshal(list.body, &records); err != nil {
		return err
	}
	if len(records) != 2 {
		return fmt.Errorf("期望 2 条已固化巡检，实际 %d：%s", len(records), list.text())
	}
	wantConcerns := [][]string{{"mite_load"}, {"mite_load", "low_strength"}}
	for i, want := range wantConcerns {
		if len(records[i].Concerns) != len(want) {
			return fmt.Errorf("第 %d 条巡检关注项被改写：%s", i+1, list.text())
		}
		for j := range want {
			if records[i].Concerns[j] != want[j] {
				return fmt.Errorf("第 %d 条巡检关注项被改写：%s", i+1, list.text())
			}
		}
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
		"observedAt": "2026-04-10T09:00:00Z", "colonyStrength": 4, "broodPattern": "solid",
		"temper": "calm", "miteCount": 12, "note": "",
	}, 201, `"mite_load"`); err != nil {
		return err
	}
	report, err := c.expect("GET", "/apiaries/apiary-north/status?asOf=2026-04-12T00:00:00Z", nil, 200,
		`"hiveCount":3`, `"colonized":2`, `"production":1`, `"layingQueens":0`,
		`"latestMeasurement":{"colonyStrength":4,"colonyStrengthUnit":"scale5","miteCount":12,"miteCountUnit":"per100"}`,
	)
	if err != nil {
		return err
	}
	for _, fragment := range []string{
		`"needsInspection":["hive-02","hive-03"]`,
		`"concernHives":["hive-01"]`,
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
