package app

import (
	"runtime"

	"github.com/egoist/mygo/ui"

	"mygo-music/internal/i18n"
)

// A CJK primary face keeps Chinese labels at a consistent weight instead
// of letting DirectWrite choose a fallback independently for missing glyphs.
func applyTypography(t *ui.Theme) {
	if runtime.GOOS != "windows" {
		return
	}
	t.Font = ""
	if i18n.Language() == i18n.Chinese {
		t.Font = `"Microsoft YaHei UI", "Microsoft YaHei", "Noto Sans CJK SC", sans-serif`
	}
}
