// Message log, question input, and typewriter reveal — INT-2 vertical slice
// edition. Partial coverage of dialogue-ui stories 002 (text input) and 003
// (conversation log + typewriter); the full stories add topic chips, trust
// indicator, scrollback, and accessibility wiring.
package dialogue_ui

import (
	"strings"
)

// Conversation layout and typewriter tuning (data-driven — extract to
// config when tuning begins).
const (
	msgMaxCharsPerLine  = 58   // wrapped line width at 16px in a 720p panel
	maxLogLines         = 12   // lines kept in the log (oldest evicted)
	logLabelCount       = 8    // static label pool size (fits the 128px log area)
	typewriterChunkWords = 2   // words revealed per tick
	typewriterTickSec    = 0.22 // ≈55 chars/sec — ADR-0012's 45–60 cps window
)

// QuestionHandler receives the player's submitted question (trimmed,
// non-empty). Called on Enter in the question input.
type QuestionHandler func(question string)

// SetQuestionHandler registers the callback invoked when the player submits
// a question. Called by the game layer; the panel never owns dialogue logic.
func (dp *DialoguePanel) SetQuestionHandler(h QuestionHandler) {
	dp.questionHandler = h
}

// onQuestionEnter handles Enter in the question input: a non-empty question
// in an input-enabled state is passed to the game layer and the panel
// transitions to Processing. Input stays gated while Processing (ADR-0012
// state machine).
func (dp *DialoguePanel) onQuestionEnter() {
	if !dp.state.IsInputEnabled() {
		return
	}
	q := strings.TrimSpace(dp.input.Text())
	if q == "" {
		return
	}
	dp.input.SetTextWithoutEvent("")
	dp.SetState(StateProcessing)
	if dp.questionHandler != nil {
		dp.questionHandler(q)
	}
}

// AddMessage appends a completed message line to the conversation log
// ("speaker: text"), wrapping long text to the panel width. Oldest lines
// are evicted past maxLogLines.
func (dp *DialoguePanel) AddMessage(speaker, text string) {
	line := text
	if speaker != "" {
		line = speaker + ": " + text
	}
	for _, l := range WrapLines(line, msgMaxCharsPerLine) {
		dp.logLines = append(dp.logLines, l)
	}
	if len(dp.logLines) > maxLogLines {
		dp.logLines = dp.logLines[len(dp.logLines)-maxLogLines:]
	}
	dp.rebuildLog()
}

// AddResponse begins a typewriter reveal of the NPC's response. The panel
// must already be in StateResponse or StateProcessing; the reveal runs over
// frames and completes with input re-enabled.
func (dp *DialoguePanel) AddResponse(text string) {
	dp.SetState(StateResponse)
	dp.typewriterFull = text
	dp.typewriterRevealed = 0
	dp.typewriterTimer = 0
	dp.rebuildLog()
}

// SkipTypewriter completes the in-flight reveal instantly.
func (dp *DialoguePanel) SkipTypewriter() {
	if dp.typewriterFull == "" {
		return
	}
	dp.typewriterFull = ""
	dp.rebuildLog()
	if dp.state == StateResponse && dp.input != nil {
		dp.input.Focus()
	}
}

// ClearLog empties the conversation log and cancels any in-flight reveal.
func (dp *DialoguePanel) ClearLog() {
	dp.logLines = nil
	dp.typewriterFull = ""
	dp.typewriterRevealed = 0
	dp.rebuildLog()
}

// updateTypewriter advances the reveal by one tick when enough time has
// passed. Called every frame from updateAnimation while a reveal is active.
func (dp *DialoguePanel) updateTypewriter(deltaTime float64) {
	if dp.typewriterFull == "" {
		return
	}
	dp.typewriterTimer += deltaTime
	if dp.typewriterTimer < typewriterTickSec {
		return
	}
	dp.typewriterTimer = 0
	dp.typewriterRevealed += typewriterChunkWords
	words := strings.Fields(dp.typewriterFull)
	if dp.typewriterRevealed >= len(words) {
		dp.rebuildLog()
		dp.typewriterFull = ""
		if dp.state == StateResponse && dp.input != nil {
			dp.input.Focus()
		}
		return
	}
	dp.rebuildLog()
}

// rebuildLog updates the static log-label pool from dp.logLines plus the
// in-flight typewriter prefix (if any). Labels are text-assigned only —
// no tree mutations at runtime (the UI update pass iterates cached child
// slices; runtime add/remove races it — INT-4 live-run panic).
func (dp *DialoguePanel) rebuildLog() {
	if len(dp.logLabels) == 0 {
		return
	}

	lines := append([]string(nil), dp.logLines...)
	if dp.typewriterFull != "" {
		words := strings.Fields(dp.typewriterFull)
		revealed := dp.typewriterRevealed
		if revealed < 0 || revealed > len(words) {
			revealed = len(words)
		}
		lines = append(lines, WrapLines(strings.Join(words[:revealed], " "), msgMaxCharsPerLine)...)
	}

	if len(lines) > maxLogLines {
		lines = lines[len(lines)-maxLogLines:]
	}
	// Show the last logLabelCount lines; the newest line sits at the bottom
	// (highest pool index). Older slots clear to empty.
	start := len(dp.logLabels) - min(len(lines), len(dp.logLabels))
	for i := range dp.logLabels {
		if dp.logLabels[i] == nil {
			continue
		}
		if i < start {
			dp.logLabels[i].SetText("")
			continue
		}
		dp.logLabels[i].SetText(lines[len(lines)-len(dp.logLabels)+i])
	}
}

// WrapLines wraps text to lines of at most maxLen characters, breaking at
// word boundaries. Long words are hard-split. Exported for unit testing.
func WrapLines(text string, maxLen int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	current := ""
	for _, w := range words {
		for len(w) > maxLen {
			space := maxLen - len(current)
			if space <= 0 {
				lines = append(lines, current)
				current = ""
				space = maxLen
			}
			current += w[:space]
			lines = append(lines, current)
			current = ""
			w = w[space:]
		}
		if current == "" {
			current = w
		} else if len(current)+1+len(w) <= maxLen {
			current += " " + w
		} else {
			lines = append(lines, current)
			current = w
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
