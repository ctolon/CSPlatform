package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
	"gopkg.in/natefinch/lumberjack.v2"
)

// ZerologBuilder implements a fluent/builder pattern for constructing a zerolog.Logger.
// Safe to reuse across goroutines only after Build() (builder itself is not locked).
//
// Example:
//  logger, _ := NewZeroLogLogger().
//      WithLevel("debug").
//      WithConsole(true).
//      WithTimeFormat(time.RFC3339).
//      WithGlobal(). // set log.Logger = built logger
//      WithFileRotation("app.log", 50, 5, 30, true).
//      WithFields(map[string]any{"app":"payments"}).
//      WithCaller(true).
//      Build()
//  logger.Info().Msg("ready")
//
// Notes:
//  - If both console and JSON are enabled, outputs are multi-written.
//  - Sampling can be enabled via WithBasicSampling or WithBurstSampling.
//  - Stack traces for errors use github.com/rs/zerolog/pkgerrors.

type ZerologBuilder struct {
	level          *zerolog.Level
	timeFormat     string
	console        bool
	consoleNoColor bool
	jsonStdout     bool
	fileRotator    *lumberjack.Logger
	outputs        []io.Writer
	basicSampleN   uint32
	burstSampleB   uint32
	burstSampleP   time.Duration
	fields         map[string]any
	caller         bool
	stack          bool
	hooks          []zerolog.Hook
	setGlobal      bool
}

// New creates a new logger builder with sensible defaults.
func NewZeroLogLoggerBuilder(jsonFmt bool) *ZerologBuilder {
	return &ZerologBuilder{
		// zerolog default level is Info
		jsonStdout: jsonFmt,
		// default ISO8601 timestamps
		timeFormat: time.RFC3339,
		fields:     map[string]any{},
	}
}

// WithLevel sets log level. Accepts either a zerolog.Level or string ("debug", "info", ...).
func (b *ZerologBuilder) WithLevel(l any) *ZerologBuilder {
	switch v := l.(type) {
	case zerolog.Level:
		b.level = &v
	case string:
		lvl, err := zerolog.ParseLevel(v)
		if err == nil {
			b.level = &lvl
		}
	}
	return b
}

// WithTimeFormat overrides the time format (default RFC3339).
func (b *ZerologBuilder) WithTimeFormat(fmt string) *ZerologBuilder { b.timeFormat = fmt; return b }

// WithConsole enables console writer to stdout (pretty printing). If noColor is true, color is disabled.
func (b *ZerologBuilder) WithConsole(noColor bool) *ZerologBuilder {
	b.console = true
	b.consoleNoColor = noColor
	return b
}

// WithJSONStdout ensures JSON is written to stdout (enabled by default). Set to false to disable.
func (b *ZerologBuilder) WithJSONStdout(enable bool) *ZerologBuilder { b.jsonStdout = enable; return b }

// WithOutput adds a custom io.Writer target.
func (b *ZerologBuilder) WithOutput(w io.Writer) *ZerologBuilder {
	if w != nil {
		b.outputs = append(b.outputs, w)
	}
	return b
}

// WithFileRotation adds a lumberjack file rotator target.
// sizeMB: max size in MB before rotation; backups: max old files; ageDays: max age; compress: gzip old files.
func (b *ZerologBuilder) WithFileRotation(path string, sizeMB, backups, ageDays int, compress bool) *ZerologBuilder {
	if path == "" {
		return b
	}
	b.fileRotator = &lumberjack.Logger{Filename: path, MaxSize: sizeMB, MaxBackups: backups, MaxAge: ageDays, Compress: compress}
	return b
}

// WithBasicSampling enables basic sampling: log 1 of N events.
func (b *ZerologBuilder) WithBasicSampling(n uint32) *ZerologBuilder {
	if n > 1 {
		b.basicSampleN = n
	}
	return b
}

// WithBurstSampling enables burst sampling: at most B events per period P.
func (b *ZerologBuilder) WithBurstSampling(burst uint32, period time.Duration) *ZerologBuilder {
	if burst > 0 && period > 0 {
		b.burstSampleB, b.burstSampleP = burst, period
	}
	return b
}

// WithField adds a single structured field.
func (b *ZerologBuilder) WithField(k string, v any) *ZerologBuilder {
	if b.fields == nil {
		b.fields = map[string]any{}
	}
	b.fields[k] = v
	return b
}

// WithFields adds multiple structured fields.
func (b *ZerologBuilder) WithFields(m map[string]any) *ZerologBuilder {
	if m != nil {
		if b.fields == nil {
			b.fields = map[string]any{}
		}
		for k, v := range m {
			b.fields[k] = v
		}
	}
	return b
}

// WithCaller toggles caller annotation (file:line).
func (b *ZerologBuilder) WithCaller(enable bool) *ZerologBuilder { b.caller = enable; return b }

// WithStack toggles error stack traces using pkgerrors.
func (b *ZerologBuilder) WithStack(enable bool) *ZerologBuilder { b.stack = enable; return b }

// WithHook registers a zerolog hook.
func (b *ZerologBuilder) WithHook(h zerolog.Hook) *ZerologBuilder {
	if h != nil {
		b.hooks = append(b.hooks, h)
	}
	return b
}

// WithGlobal sets the built logger as global (log.Logger = logger; zerolog.DefaultContextLogger too).
func (b *ZerologBuilder) WithGlobal() *ZerologBuilder { b.setGlobal = true; return b }

// Build assembles and returns the configured zerolog.Logger.
func (b *ZerologBuilder) Build() (zerolog.Logger, error) {
	// Time format
	zerolog.TimeFieldFormat = b.timeFormat
	if b.stack {
		zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	}

	var sinks []io.Writer
	// JSON stdout
	if b.jsonStdout {
		sinks = append(sinks, os.Stdout)
	}
	// Console writer
	if b.console {
		cw := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: b.timeFormat, NoColor: b.consoleNoColor}
		sinks = append(sinks, cw)
	}
	// File rotator
	if b.fileRotator != nil {
		sinks = append(sinks, b.fileRotator)
	}
	// Custom outputs
	if len(b.outputs) > 0 {
		sinks = append(sinks, b.outputs...)
	}
	if len(sinks) == 0 {
		// fallback to stdout JSON
		sinks = []io.Writer{os.Stdout}
	}

	mw := io.MultiWriter(sinks...)
	zl := zerolog.New(mw).With().Timestamp().Logger()

	// Level
	if b.level != nil {
		zl = zl.Level(*b.level)
	}

	// Context fields
	if len(b.fields) > 0 {
		zl = zl.With().Fields(b.fields).Logger()
	}

	// Caller
	if b.caller {
		zl = zl.With().Caller().Logger()
	}

	// Sampling
	if b.basicSampleN > 1 {
		zl = zl.Sample(&zerolog.BasicSampler{N: b.basicSampleN})
	} else if b.burstSampleB > 0 && b.burstSampleP > 0 {
		zl = zl.Sample(&zerolog.BurstSampler{Burst: b.burstSampleB, Period: b.burstSampleP})
	}

	// Hooks
	for _, h := range b.hooks {
		zl = zl.Hook(h)
	}

	if b.setGlobal {
		log.Logger = zl
	}
	return zl, nil
}

// --------- Optional example hooks & helpers ---------

// EnrichHook adds static fields to every event (alternative to WithFields when you prefer hook style).
type EnrichHook struct {
	K string
	V any
}

func (h EnrichHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	if h.K != "" {
		e.Interface(h.K, h.V)
	}
}

// LevelNameHook records the level name as a string field "level_name" (useful when consuming logs without zerolog decoder).
// Note zerolog already encodes level, this is purely demonstrative.
//var ptype := struct{}{}

type LevelNameHook struct{}

func (LevelNameHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	e.Str("level_name", level.String())
}

// --------- Example main (keep in separate file) ---------
// package main
//
// import (
//   "time"
//   zlb "your/module/logger"
// )
//
// func main() {
//   lg, _ := zlb.New().
//     WithLevel("debug").
//     WithConsole(false).        // pretty console (NoColor=false)
//     WithJSONStdout(true).      // keep JSON too
//     WithTimeFormat(time.RFC3339Nano).
//     WithFileRotation("app.log", 50, 3, 30, true).
//     WithFields(map[string]any{"service":"billing", "env":"dev"}).
//     WithCaller(true).
//     WithStack(true).
//     WithHook(zlb.LevelNameHook{}).
//     WithGlobal().
//     Build()
//
//   lg.Info().Str("user","batuhan").Msg("logger is ready")
// }
