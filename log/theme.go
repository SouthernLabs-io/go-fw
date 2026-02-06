package log

import (
	"log/slog"

	"github.com/phsym/console-slog"
)

type brightMessageTheme struct {
	base console.Theme
}

func (t brightMessageTheme) Name() string               { return t.base.Name() }
func (t brightMessageTheme) Timestamp() console.ANSIMod { return t.base.Timestamp() }
func (t brightMessageTheme) Source() console.ANSIMod    { return t.base.Source() }
func (t brightMessageTheme) Message() console.ANSIMod {
	// Just bold, but not hard-coded white (that only works on dark terminals)
	return console.ToANSICode(console.Bold)
}
func (t brightMessageTheme) MessageDebug() console.ANSIMod   { return t.base.MessageDebug() }
func (t brightMessageTheme) AttrKey() console.ANSIMod        { return t.base.AttrKey() }
func (t brightMessageTheme) AttrValue() console.ANSIMod      { return t.base.AttrValue() }
func (t brightMessageTheme) AttrValueError() console.ANSIMod { return t.base.AttrValueError() }
func (t brightMessageTheme) LevelError() console.ANSIMod     { return t.base.LevelError() }
func (t brightMessageTheme) LevelWarn() console.ANSIMod      { return t.base.LevelWarn() }
func (t brightMessageTheme) LevelInfo() console.ANSIMod      { return t.base.LevelInfo() }
func (t brightMessageTheme) LevelDebug() console.ANSIMod     { return t.base.LevelDebug() }
func (t brightMessageTheme) Level(level slog.Level) console.ANSIMod {
	return t.base.Level(level)
}

func newConsoleTheme() console.Theme {
	return brightMessageTheme{base: console.NewBrightTheme()}
}
