package progress

import (
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"sync"
	"time"
)

// Template keys for use with WithTemplate.
const (
	KeySpinner = "{spinner}"
	KeyDesc    = "{desc}"
	KeyLeft    = "{left}"
	KeyBar     = "{bar}"
	KeyPercent = "{percent}"
	KeyCurrent = "{current}"
	KeyTotal   = "{total}"
	KeySpeed   = "{speed}"
	KeyETA     = "{eta}"
	KeyElapsed = "{elapsed}"
	KeyRight   = "{right}"
)

// UnknownValue is used when a template placeholder cannot be resolved.
const UnknownValue = "?"

// Internal formatting constants.
const (
	percentScale              = 100.0
	animStepDuration          = 200 * time.Millisecond
	indeterminateBlockDivisor = 3
	speedFormat               = "%.1f/s"
	emptySpeed                = "0.0/s"
	etaPrefix                 = "ETA "
	etaLessThanSecond         = "<1s"
)

// Sentinel errors returned by update methods.
var (
	ErrClosed         = errors.New("progress: tracker is closed")
	ErrInvalidPercent = errors.New("progress: percent must be between 0 and 100")
	ErrNoTotal        = errors.New("progress: total steps not set, cannot use percentage")
)

// Format defines the visual layout and appearance of a progress tracker.
type Format struct {
	Desc           string
	LeftText       string
	RightText      string
	BarStart       string
	BarEnd         string
	BarWidth       int
	Filled         string
	Pointer        string
	Unfilled       string
	Spinner        []string
	ShowPercentage bool
	ShowCount      bool
	ShowSpeed      bool
	SpeedUnits     string
	ShowETA        bool
	Template       string

	// DoneString is shown instead of ETA when current >= total.
	// If empty, no ETA replacement is done (backward compatible).
	DoneString string
	// TimeFormatter, if non-nil, formats elapsed and ETA durations.
	// The default formatter uses Truncate(time.Second) for both,
	// with the special case "<1s" for ETA below one second.
	TimeFormatter func(time.Duration) string
}

// DefaultFormat returns a Format with sensible defaults.
func DefaultFormat() Format {
	return Format{
		Desc:           "",
		LeftText:       "",
		RightText:      "",
		BarStart:       "[",
		BarEnd:         "]",
		BarWidth:       40,
		Filled:         "=",
		Pointer:        ">",
		Unfilled:       " ",
		Spinner:        nil,
		ShowPercentage: true,
		ShowCount:      true,
		ShowSpeed:      false,
		SpeedUnits:     "",
		ShowETA:        false,
		Template:       "",
		DoneString:     "Done",
		TimeFormatter:  nil,
	}
}

// Tracker monitors the progress of a process.
type Tracker struct {
	mu              sync.Mutex
	currentStep     int64
	totalSteps      int64
	startTime       time.Time
	lastUpdateTime  time.Time
	prevStep        int64
	elapsed         time.Duration
	eta             time.Duration
	speed           float64
	writer          io.Writer
	ticker          *time.Ticker
	renderFormat    Format
	manualRender    bool
	autoStart       bool
	done            chan struct{}
	wg              sync.WaitGroup
	endClear        bool
	lastString      string
	refreshInterval time.Duration
	spinnerIdx      int

	timeout      time.Duration
	timeoutTimer *time.Timer

	closed bool
}

// Option configures a Tracker.
type Option func(*Tracker)

// WithTotal sets the total number of steps. Values <= 0 indicate indeterminate progress.
func WithTotal(n int64) Option {
	return func(t *Tracker) {
		t.totalSteps = n
	}
}

// WithDescription sets the static description shown before the progress bar.
func WithDescription(desc string) Option {
	return func(t *Tracker) {
		t.renderFormat.Desc = desc
	}
}

// WithFormat replaces the entire visual format.
func WithFormat(f Format) Option {
	return func(t *Tracker) {
		t.renderFormat = f
	}
}

// WithBarWidth sets the width of the progress bar in characters.
func WithBarWidth(w int) Option {
	return func(t *Tracker) {
		if w > 0 {
			t.renderFormat.BarWidth = w
		}
	}
}

// WithBarFilled sets the character(s) used for completed progress.
func WithBarFilled(s string) Option {
	return func(t *Tracker) {
		t.renderFormat.Filled = s
	}
}

// WithBarUnfilled sets the character(s) used for remaining progress.
func WithBarUnfilled(s string) Option {
	return func(t *Tracker) {
		t.renderFormat.Unfilled = s
	}
}

// WithBarPointer sets the character at the head of the completed section.
func WithBarPointer(s string) Option {
	return func(t *Tracker) {
		t.renderFormat.Pointer = s
	}
}

// WithBarBounds sets the opening and closing bracket of the progress bar.
func WithBarBounds(start, end string) Option {
	return func(t *Tracker) {
		t.renderFormat.BarStart = start
		t.renderFormat.BarEnd = end
	}
}

// WithSpinner enables a spinner animation using the provided frames.
func WithSpinner(frames []string) Option {
	return func(t *Tracker) {
		t.renderFormat.Spinner = frames
	}
}

// WithOutput sets the io.Writer to which the progress line is written automatically.
func WithOutput(w io.Writer) Option {
	return func(t *Tracker) {
		t.writer = w
	}
}

// WithRefreshInterval sets the interval between automatic redraws.
func WithRefreshInterval(d time.Duration) Option {
	return func(t *Tracker) {
		if d > 0 {
			t.refreshInterval = d
		}
	}
}

// // WithEndSeparation controls whether a newline is printed after the progress line
// // when the tracker is closed.
// func WithEndSeparation(sep bool) Option {
// 	return func(t *Tracker) {
// 		t.endSeparation = sep
// 	}
// }

// WithEndClear controls whether the progress line is erased when the tracker is closed.
func WithEndClear(clear bool) Option {
	return func(t *Tracker) {
		t.endClear = clear
	}
}

// WithManualRender disables the background goroutine.
func WithManualRender() Option {
	return func(t *Tracker) {
		t.manualRender = true
	}
}

// WithAutoStart controls whether the internal clock starts immediately upon creation.
func WithAutoStart(auto bool) Option {
	return func(t *Tracker) {
		t.autoStart = auto
	}
}

// WithInitialStep sets the starting step value.
func WithInitialStep(n int64) Option {
	return func(t *Tracker) {
		t.currentStep = n
	}
}

// WithTemplate sets a custom template string that fully controls the output line.
func WithTemplate(tmpl string) Option {
	return func(t *Tracker) {
		t.renderFormat.Template = tmpl
	}
}

// WithTimeout sets a timeout after which the tracker is automatically closed.
func WithTimeout(d time.Duration) Option {
	return func(t *Tracker) {
		if d > 0 {
			t.timeout = d
		}
	}
}

// WithTimeFormatter sets a custom duration formatter for elapsed and ETA.
func WithTimeFormatter(f func(time.Duration) string) Option {
	return func(t *Tracker) {
		t.renderFormat.TimeFormatter = f
	}
}

// WithDoneString sets the text shown instead of ETA when the progress completes.
func WithDoneString(s string) Option {
	return func(t *Tracker) {
		t.renderFormat.DoneString = s
	}
}

// CommonSpinnerFrames returns a frequently used spinner animation.
func CommonSpinnerFrames() []string {
	return []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
}

// NewTracker creates a new Tracker, applies options, and starts the background loop if required.
func NewTracker(opts ...Option) *Tracker {
	t := &Tracker{
		totalSteps:      100,
		renderFormat:    DefaultFormat(),
		refreshInterval: 100 * time.Millisecond,
		autoStart:       false,
		manualRender:    false,
		endClear:        false,
	}

	for _, opt := range opts {
		opt(t)
	}

	// Disable automatic output if no writer is set.
	if t.writer == nil {
		t.manualRender = true
	}

	if t.autoStart {
		t.startTiming()
	}

	if !t.manualRender && t.writer != nil {
		t.done = make(chan struct{})
		t.ticker = time.NewTicker(t.refreshInterval)
		t.wg.Add(1)
		go t.run()
		if t.autoStart {
			t.renderAndWrite()
		}
	}

	if t.timeout > 0 {
		t.timeoutTimer = time.AfterFunc(t.timeout, func() {
			t.Close()
		})
	}

	return t
}

// startTiming initialises the start and last-update timestamps if not already set.
func (t *Tracker) startTiming() {
	if !t.startTime.IsZero() {
		return
	}
	now := time.Now()
	t.startTime = now
	t.lastUpdateTime = now
	t.prevStep = t.currentStep
}

// run is the background loop that periodically redraws the progress line.
func (t *Tracker) run() {
	defer t.wg.Done()
	for {
		select {
		case <-t.ticker.C:
			t.renderAndWrite()
		case <-t.done:
			return
		}
	}
}

// renderAndWrite writes the current progress line with a carriage return.
func (t *Tracker) renderAndWrite() {
	line := t.renderLine()
	if t.writer != nil {
		fmt.Fprintf(t.writer, "\r%s", line)
	}
}

// Render recalculates elapsed time, speed, ETA and returns the current progress line.
func (t *Tracker) Render() string {
	return t.renderLine()
}

// String returns the most recently rendered progress line.
func (t *Tracker) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.lastString
}

// renderLine updates internal statistics and builds the progress string.
// Must be called with the mutex held.
func (t *Tracker) renderLine() string {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	t.updateStats(now)

	line := t.buildString()
	t.lastString = line
	return line
}

// updateStats recomputes elapsed time, speed and ETA based on the current time.
// Must be called with the mutex held.
func (t *Tracker) updateStats(now time.Time) {
	if !t.startTime.IsZero() {
		t.elapsed = now.Sub(t.startTime)
	}

	if t.lastUpdateTime.IsZero() || t.startTime.IsZero() {
		t.lastUpdateTime = now
		t.prevStep = t.currentStep
		return
	}

	dt := now.Sub(t.lastUpdateTime)
	if dt > 0 {
		delta := t.currentStep - t.prevStep
		if delta > 0 {
			t.speed = float64(delta) / dt.Seconds()
		}
		// if delta <= 0, keep previous speed
	}

	if t.totalSteps > 0 {
		remaining := t.totalSteps - t.currentStep
		if remaining <= 0 {
			t.eta = 0
		} else if t.speed > 0 {
			t.eta = time.Duration(float64(remaining) / t.speed * float64(time.Second))
		}
	} else {
		t.eta = 0
	}

	t.lastUpdateTime = now
	t.prevStep = t.currentStep
}

// buildString assembles the full progress line from all visual components.
// Must be called with the mutex held.
func (t *Tracker) buildString() string {
	spin := t.spinFrame()
	bar := t.renderBar()
	percent := t.percentString()
	current := t.currentString()
	total := t.totalString()
	eta := t.etaString()
	speed := t.speedString()
	elapsed := t.elapsedString()
	right := t.buildRightText(percent, current, total, speed, eta)

	if t.renderFormat.Template != "" {
		return t.applyTemplate(spin, t.renderFormat.Desc, t.renderFormat.LeftText, bar, percent, current, total, speed, eta, elapsed, right)
	}
	return t.defaultLine(spin, t.renderFormat.Desc, t.renderFormat.LeftText, bar, right)
}

// spinFrame returns the current spinner frame and advances the index.
// Must be called with the mutex held.
func (t *Tracker) spinFrame() string {
	if len(t.renderFormat.Spinner) == 0 {
		return ""
	}
	frame := t.renderFormat.Spinner[t.spinnerIdx]
	t.spinnerIdx = (t.spinnerIdx + 1) % len(t.renderFormat.Spinner)
	return frame
}

// renderBar returns the textual progress bar for the current step.
// Must be called with the mutex held.
func (t *Tracker) renderBar() string {
	if t.totalSteps <= 0 {
		return t.indeterminateBar(t.renderFormat)
	}
	return t.determinateBar(t.renderFormat)
}

// percentString returns the formatted percentage string, or UnknownValue.
func (t *Tracker) percentString() string {
	if t.totalSteps <= 0 {
		return UnknownValue
	}
	p := float64(t.currentStep) / float64(t.totalSteps) * percentScale
	if p > percentScale {
		p = percentScale
	}
	return fmt.Sprintf("%.0f%%", p)
}

// currentString returns the current step as a string, or UnknownValue.
func (t *Tracker) currentString() string {
	if t.totalSteps <= 0 {
		return UnknownValue
	}
	return fmt.Sprintf("%d", t.currentStep)
}

// totalString returns the total steps as a string, or UnknownValue.
func (t *Tracker) totalString() string {
	if t.totalSteps <= 0 {
		return UnknownValue
	}
	return fmt.Sprintf("%d", t.totalSteps)
}

// etaString returns the formatted ETA string, handling completion and unknown states.
func (t *Tracker) etaString() string {
	if t.totalSteps <= 0 {
		return UnknownValue
	}
	if t.currentStep >= t.totalSteps {
		if t.renderFormat.DoneString != "" {
			return t.renderFormat.DoneString
		}
		return UnknownValue
	}
	if t.eta < 0 {
		return UnknownValue
	}
	if t.renderFormat.TimeFormatter != nil {
		return t.renderFormat.TimeFormatter(t.eta)
	}
	if t.eta < time.Second {
		return etaLessThanSecond
	}
	return t.eta.Truncate(time.Second).String()
}

// speedString returns the formatted speed string.
func (t *Tracker) speedString() string {
	if t.speed == 0 {
		return emptySpeed
	}
	return fmt.Sprintf(speedFormat, t.speed)
}

// elapsedString returns the formatted elapsed time string, or UnknownValue.
func (t *Tracker) elapsedString() string {
	if t.startTime.IsZero() {
		return UnknownValue
	}
	if t.renderFormat.TimeFormatter != nil {
		return t.renderFormat.TimeFormatter(t.elapsed)
	}
	return t.elapsed.Truncate(time.Second).String()
}

// buildRightText composes the right-side text from the enabled components.
// It uses only the provided string arguments, without reading Tracker fields.
func (t *Tracker) buildRightText(percent, current, total, speed, eta string) string {
	f := t.renderFormat
	if f.RightText != "" {
		return f.RightText
	}
	var parts []string
	if f.ShowPercentage && total != UnknownValue {
		parts = append(parts, percent)
	}
	if f.ShowCount && total != UnknownValue {
		parts = append(parts, current+"/"+total)
	}
	if f.ShowSpeed {
		parts = append(parts, speed)
	}
	if f.ShowETA && eta != UnknownValue {
		// If the completion string is set and matches, output without prefix.
		if f.DoneString != "" && eta == f.DoneString {
			parts = append(parts, eta)
		} else {
			parts = append(parts, etaPrefix+eta)
		}
	}
	return strings.Join(parts, " ")
}

// applyTemplate fills the custom template with the provided values.
func (t *Tracker) applyTemplate(spin, desc, left, bar, percent, current, total, speed, eta, elapsed, right string) string {
	replacer := strings.NewReplacer(
		KeySpinner, spin,
		KeyDesc, desc,
		KeyLeft, left,
		KeyBar, bar,
		KeyPercent, percent,
		KeyCurrent, current,
		KeyTotal, total,
		KeySpeed, speed,
		KeyETA, eta,
		KeyElapsed, elapsed,
		KeyRight, right,
	)
	result := replacer.Replace(t.renderFormat.Template)
	return sanitizeUnknownPlaceholders(result)
}

// defaultLine builds the standard progress line without a template.
func (t *Tracker) defaultLine(spin, desc, left, bar, right string) string {
	var line string
	if spin != "" {
		line += spin + " "
	}
	if left != "" {
		line += left + " "
	}
	if desc != "" {
		line += desc + " "
	}
	line += bar
	if right != "" {
		line += " " + right
	}
	return line
}

// knownKeys is the set of valid template placeholders.
var knownKeys = map[string]bool{
	KeySpinner: true,
	KeyDesc:    true,
	KeyLeft:    true,
	KeyBar:     true,
	KeyPercent: true,
	KeyCurrent: true,
	KeyTotal:   true,
	KeySpeed:   true,
	KeyETA:     true,
	KeyElapsed: true,
	KeyRight:   true,
}

// sanitizeUnknownPlaceholders replaces unrecognised {placeholders} with UnknownValue.
func sanitizeUnknownPlaceholders(s string) string {
	if !strings.Contains(s, "{") {
		return s
	}
	var result strings.Builder
	result.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == '{' {
			end := strings.IndexByte(s[i:], '}')
			if end == -1 {
				result.WriteString(s[i:])
				break
			}
			end += i
			placeholder := s[i : end+1]
			if knownKeys[placeholder] {
				result.WriteString(placeholder)
			} else {
				result.WriteString(UnknownValue)
			}
			i = end + 1
		} else {
			result.WriteByte(s[i])
			i++
		}
	}
	return result.String()
}

// determinateBar builds a progress bar for a known total.
func (t *Tracker) determinateBar(f Format) string {
	barWidth := f.BarWidth
	if barWidth <= 0 {
		barWidth = 40
	}
	if t.currentStep >= t.totalSteps {
		return f.BarStart + strings.Repeat(f.Filled, barWidth) + f.BarEnd
	}

	percent := float64(t.currentStep) / float64(t.totalSteps)
	filledLen := int(math.Round(percent * float64(barWidth)))
	if filledLen > barWidth {
		filledLen = barWidth
	}

	out := make([]rune, 0, barWidth+len(f.BarStart)+len(f.BarEnd))
	out = append(out, []rune(f.BarStart)...)
	for i := 0; i < barWidth; i++ {
		if i < filledLen-1 {
			out = append(out, []rune(f.Filled)...)
		} else if i == filledLen-1 && filledLen > 0 {
			out = append(out, []rune(f.Pointer)...)
		} else {
			out = append(out, []rune(f.Unfilled)...)
		}
	}
	out = append(out, []rune(f.BarEnd)...)
	return string(out)
}

// indeterminateBar builds an animated bar for unknown total.
func (t *Tracker) indeterminateBar(f Format) string {
	barWidth := f.BarWidth
	if barWidth <= 0 {
		barWidth = 40
	}
	blockWidth := max(barWidth/indeterminateBlockDivisor, 1)

	steps := int(time.Since(t.startTime) / animStepDuration)

	offset := min(steps%(barWidth+blockWidth), barWidth)

	out := make([]rune, 0, barWidth+len(f.BarStart)+len(f.BarEnd))
	out = append(out, []rune(f.BarStart)...)
	for i := 0; i < barWidth; i++ {
		if i >= offset && i < offset+blockWidth {
			out = append(out, []rune(f.Filled)...)
		} else {
			out = append(out, []rune(f.Unfilled)...)
		}
	}
	out = append(out, []rune(f.BarEnd)...)
	return string(out)
}

// Set updates the current step value. Returns error if the tracker is closed.
func (t *Tracker) Set(step int64) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return ErrClosed
	}
	t.currentStep = step
	return nil
}

// SetPct sets progress by percentage (0–100). Returns error if total steps unknown,
// percent out of range, or tracker closed.
func (t *Tracker) SetPct(pct float64) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return ErrClosed
	}
	if t.totalSteps <= 0 {
		return ErrNoTotal
	}
	if pct < 0 || pct > percentScale {
		return ErrInvalidPercent
	}
	t.currentStep = int64(math.Round(pct / percentScale * float64(t.totalSteps)))
	return nil
}

// Add adds delta to the current step. Returns error if tracker is closed.
func (t *Tracker) Add(delta int64) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return ErrClosed
	}
	t.currentStep += delta
	return nil
}

// Increment adds 1 to the current step. Returns error if tracker is closed.
func (t *Tracker) Increment() error {
	return t.Add(1)
}

// Done sets the current step equal to the total. Returns error if tracker is closed.
func (t *Tracker) Done() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return ErrClosed
	}
	t.currentStep = t.totalSteps
	return nil
}

// Current returns the current step count.
func (t *Tracker) Current() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.currentStep
}

// Total returns the total number of steps.
func (t *Tracker) Total() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.totalSteps
}

// Percent returns the completion percentage (0–100) if total > 0, otherwise NaN.
func (t *Tracker) Percent() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.totalSteps <= 0 {
		return math.NaN()
	}
	p := float64(t.currentStep) / float64(t.totalSteps) * percentScale
	if p > percentScale {
		p = percentScale
	}
	return p
}

// Elapsed returns the time since the tracker started.
func (t *Tracker) Elapsed() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.startTime.IsZero() {
		return 0
	}
	return time.Since(t.startTime)
}

// ETA returns the estimated time until completion.
func (t *Tracker) ETA() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.eta
}

// Speed returns the processing speed in units per second.
func (t *Tracker) Speed() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.speed
}

// UpdateFormat replaces the display format at runtime.
func (t *Tracker) UpdateFormat(f Format) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.renderFormat = f
}

// Start begins the internal clock if it was delayed via WithAutoStart(false).
func (t *Tracker) Start() {
	t.mu.Lock()
	already := !t.startTime.IsZero()
	t.mu.Unlock()
	if already {
		return
	}
	t.mu.Lock()
	t.startTiming()
	t.mu.Unlock()
}

// Close shuts down the background goroutine, stops the ticker and timeout timer,
// optionally clears the last line, and finalises the output. Safe to call multiple times.
func (t *Tracker) Close() {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return
	}
	t.closed = true
	t.mu.Unlock()

	if t.timeoutTimer != nil {
		t.timeoutTimer.Stop()
	}

	if t.done != nil {
		close(t.done)
	}
	t.wg.Wait()

	if t.ticker != nil {
		t.ticker.Stop()
	}

	// Final render to show complete state.
	if t.writer != nil && !t.manualRender {
		t.renderAndWrite()
	}

	if t.writer != nil {
		if t.endClear {
			clear := strings.Repeat(" ", len(t.lastString))
			fmt.Fprintf(t.writer, "\r%s\r", clear)
		} else {
			fmt.Fprintln(t.writer)
		}
	}
}

// SetTotal updates the total number of steps. Values <= 0 indicate indeterminate progress.
// Safe for concurrent use.
func (t *Tracker) SetTotal(total int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.totalSteps = total
}
