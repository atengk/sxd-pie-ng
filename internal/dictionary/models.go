// Package dictionary 提供基于老版 Pieb.db 游戏数据字典的高性能只读查询仓储。
//
// @author Ateng
// @since 2026-10-08
package dictionary

// Item 代表神仙道游戏内的物品或道具元数据。
type Item struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	TypeID       int    `json:"type_id"`
	Description  string `json:"description"`
	Quality      int    `json:"quality"`
	RequireLevel int    `json:"require_level"`
	Price        int    `json:"price"`
	Ingot        int    `json:"ingot"`
}

// NPC 代表城镇或副本中的 NPC 元数据及坐标。
type NPC struct {
	ID     int    `json:"id"`
	TownID int    `json:"town_id"`
	NPCID  int    `json:"npc_id"`
	Name   string `json:"name"`
	CoordX int    `json:"coord_x"`
	CoordY int    `json:"coord_y"`
}

// RoleType 代表游戏内的主角与招募伙伴元数据。
type RoleType struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	JobID   int    `json:"job_id"`
	Type    int    `json:"type"`
	Fame    int    `json:"fame"`
	StuntID int    `json:"stunt_id"`
	Gender  int    `json:"gender"`
}
