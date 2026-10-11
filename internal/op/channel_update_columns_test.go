package op

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"

	"github.com/bestruirui/octopus/internal/model"
)

// TestChannelUpdateColumnsCoverConfig 渠道更新的列白名单必须覆盖 ChannelConfig 的全部可写字段。
//
// 背景: ChannelUpdate 逐列点名写回 (不能整行覆盖, 否则统计列被抹), 于是白名单就是一张
// 手抄的字段清单 —— client_emulation 上线时漏抄, 表现是"设置写入成功、重启后悄悄回退":
// 值只进了内存缓存, 落库时被静默跳过, 排查成本极高。
// 这个测试用 GORM 的 schema 解析器取出每个字段的真实数据库列名, 与白名单对账:
// 给 ChannelConfig 加字段而没补白名单, 这里直接红。
func TestChannelUpdateColumnsCoverConfig(t *testing.T) {
	parsed, err := schema.Parse(&model.ChannelConfig{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}

	columns := make(map[string]struct{}, len(channelUpdateColumns))
	for _, column := range channelUpdateColumns {
		columns[column] = struct{}{}
	}

	for _, field := range parsed.Fields {
		if _, ok := columns[field.DBName]; !ok {
			t.Fatalf("ChannelConfig.%s (列 %q) 不在 channelUpdateColumns 里: "+
				"补上对应列, 否则该字段会写进缓存却落不了库, 重启后静默回退", field.Name, field.DBName)
		}
	}
	// 反向检查: 白名单里不能有拼错的列名 (对不上的列 GORM 更新时会报错, 但错误只在运行时出现)。
	for _, column := range channelUpdateColumns {
		if parsed.LookUpField(column) == nil {
			// LookUpField 按 Go 字段名查, 这里按列名查, 故先建立列名集合再比对。
			found := false
			for _, field := range parsed.Fields {
				if field.DBName == column {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("channelUpdateColumns 含未知列 %q: 检查拼写", column)
			}
		}
	}
}
