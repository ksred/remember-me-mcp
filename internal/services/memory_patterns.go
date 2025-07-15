package services

import (
	"regexp"
	"strings"
)

// MemoryPattern represents a pattern for automatic memory detection
type MemoryPattern struct {
	Pattern    *regexp.Regexp
	Type       string
	Category   string
	Priority   MemoryPriority
	KeyExtract func(string) string // Extract key for deduplication
}

// MemoryPriority represents the importance level of a memory
type MemoryPriority int

const (
	LowPriority MemoryPriority = iota
	MediumPriority
	HighPriority
	CriticalPriority
)

// String returns the string representation of priority
func (p MemoryPriority) String() string {
	switch p {
	case LowPriority:
		return "low"
	case MediumPriority:
		return "medium"
	case HighPriority:
		return "high"
	case CriticalPriority:
		return "critical"
	default:
		return "medium"
	}
}

// Memory detection patterns
var memoryPatterns = []MemoryPattern{
	// Explicit memory requests (HIGH priority)
	{
		Pattern:  regexp.MustCompile(`(?i)remember that (.+)`),
		Type:     "fact",
		Category: "personal",
		Priority: HighPriority,
		KeyExtract: func(content string) string {
			return strings.ToLower(content)
		},
	},
	{
		Pattern:  regexp.MustCompile(`(?i)don't forget (.+)`),
		Type:     "fact",
		Category: "personal",
		Priority: HighPriority,
		KeyExtract: func(content string) string {
			return strings.ToLower(content)
		},
	},
	{
		Pattern:  regexp.MustCompile(`(?i)make a note that (.+)`),
		Type:     "fact",
		Category: "personal",
		Priority: HighPriority,
		KeyExtract: func(content string) string {
			return strings.ToLower(content)
		},
	},
	{
		Pattern:  regexp.MustCompile(`(?i)keep in mind (.+)`),
		Type:     "context",
		Category: "business",
		Priority: HighPriority,
		KeyExtract: func(content string) string {
			return strings.ToLower(content)
		},
	},

	// Personal preferences (HIGH priority, deduplication key)
	{
		Pattern:  regexp.MustCompile(`(?i)i prefer (.+)`),
		Type:     "preference",
		Category: "personal",
		Priority: HighPriority,
		KeyExtract: func(content string) string {
			return "preference:" + extractPreferenceKey(content)
		},
	},
	{
		Pattern:  regexp.MustCompile(`(?i)i like (.+)`),
		Type:     "preference",
		Category: "personal",
		Priority: MediumPriority,
		KeyExtract: func(content string) string {
			return "like:" + extractPreferenceKey(content)
		},
	},
	{
		Pattern:  regexp.MustCompile(`(?i)i dislike (.+)`),
		Type:     "preference",
		Category: "personal",
		Priority: MediumPriority,
		KeyExtract: func(content string) string {
			return "dislike:" + extractPreferenceKey(content)
		},
	},

	// Personal facts with deduplication (MEDIUM priority)
	{
		Pattern:  regexp.MustCompile(`(?i)my (.+) is (.+)`),
		Type:     "fact",
		Category: "personal",
		Priority: MediumPriority,
		KeyExtract: func(content string) string {
			matches := regexp.MustCompile(`(?i)my (.+?) is`).FindStringSubmatch(content)
			if len(matches) > 1 {
				return "my:" + strings.ToLower(matches[1])
			}
			return strings.ToLower(content)
		},
	},
	{
		Pattern:  regexp.MustCompile(`(?i)i work at (.+)`),
		Type:     "fact",
		Category: "personal",
		Priority: HighPriority,
		KeyExtract: func(content string) string {
			return "work:company"
		},
	},
	{
		Pattern:  regexp.MustCompile(`(?i)i live in (.+)`),
		Type:     "fact",
		Category: "personal",
		Priority: HighPriority,
		KeyExtract: func(content string) string {
			return "location:residence"
		},
	},

	// Project/work context (HIGH priority)
	{
		Pattern:  regexp.MustCompile(`(?i)i'm working on (.+)`),
		Type:     "context",
		Category: "project",
		Priority: HighPriority,
		KeyExtract: func(content string) string {
			return "project:" + extractProjectKey(content)
		},
	},
	{
		Pattern:  regexp.MustCompile(`(?i)i'm learning (.+)`),
		Type:     "context",
		Category: "personal",
		Priority: MediumPriority,
		KeyExtract: func(content string) string {
			return "learning:" + extractLearningKey(content)
		},
	},

	// Decisions and outcomes (HIGH priority)
	{
		Pattern:  regexp.MustCompile(`(?i)i decided to (.+)`),
		Type:     "fact",
		Category: "personal",
		Priority: HighPriority,
		KeyExtract: func(content string) string {
			return "decision:" + strings.ToLower(content)
		},
	},
	{
		Pattern:  regexp.MustCompile(`(?i)we agreed that (.+)`),
		Type:     "fact",
		Category: "business",
		Priority: HighPriority,
		KeyExtract: func(content string) string {
			return "agreement:" + strings.ToLower(content)
		},
	},

	// Measurements and stats (for your running time example)
	{
		Pattern:  regexp.MustCompile(`(?i)my (.+) (?:time|speed|score|result) is (.+)`),
		Type:     "fact",
		Category: "personal",
		Priority: MediumPriority,
		KeyExtract: func(content string) string {
			matches := regexp.MustCompile(`(?i)my (.+?) (?:time|speed|score|result) is`).FindStringSubmatch(content)
			if len(matches) > 1 {
				return "performance:" + strings.ToLower(matches[1])
			}
			return strings.ToLower(content)
		},
	},
}

// Sensitive information patterns that should NOT be stored
var sensitivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)password`),
	regexp.MustCompile(`(?i)secret`),
	regexp.MustCompile(`(?i)token`),
	regexp.MustCompile(`(?i)api.?key`),
	regexp.MustCompile(`(?i)ssn|social.?security`),
	regexp.MustCompile(`(?i)credit.?card`),
	regexp.MustCompile(`(?i)banking|account.?number`),
	regexp.MustCompile(`(?i)pin.?code`),
	regexp.MustCompile(`(?i)private.?key`),
	regexp.MustCompile(`(?i)oauth`),
}

// DetectedMemory represents automatically detected memory content
type DetectedMemory struct {
	Content    string
	Type       string
	Category   string
	Priority   MemoryPriority
	UpdateKey  string // Key for deduplication/updates
	Confidence float64
}

// DetectMemoryPatterns automatically detects memory-worthy content
func DetectMemoryPatterns(content string) []DetectedMemory {
	var detected []DetectedMemory

	// Check if content contains sensitive information
	if containsSensitiveInfo(content) {
		return detected // Return empty if sensitive
	}

	// Check against all memory patterns
	for _, pattern := range memoryPatterns {
		if pattern.Pattern.MatchString(content) {
			memory := DetectedMemory{
				Content:    content,
				Type:       pattern.Type,
				Category:   pattern.Category,
				Priority:   pattern.Priority,
				UpdateKey:  pattern.KeyExtract(content),
				Confidence: calculateConfidence(content, pattern),
			}
			detected = append(detected, memory)
		}
	}

	return detected
}

// containsSensitiveInfo checks if content contains sensitive information
func containsSensitiveInfo(content string) bool {
	for _, pattern := range sensitivePatterns {
		if pattern.MatchString(content) {
			return true
		}
	}
	return false
}

// calculateConfidence calculates how confident we are about the memory detection
func calculateConfidence(content string, pattern MemoryPattern) float64 {
	// Base confidence based on pattern type
	baseConfidence := 0.7
	
	// Higher confidence for explicit requests
	if strings.Contains(strings.ToLower(content), "remember") {
		baseConfidence = 0.95
	}
	
	// Higher confidence for strong personal indicators
	if strings.Contains(strings.ToLower(content), "i prefer") || 
	   strings.Contains(strings.ToLower(content), "i work at") {
		baseConfidence = 0.9
	}
	
	// Lower confidence for casual mentions
	if strings.Contains(strings.ToLower(content), "maybe") || 
	   strings.Contains(strings.ToLower(content), "might") {
		baseConfidence = 0.5
	}
	
	return baseConfidence
}

// Helper functions for extracting deduplication keys
func extractPreferenceKey(content string) string {
	// Extract the subject of preference (e.g., "TypeScript" from "I prefer TypeScript")
	re := regexp.MustCompile(`(?i)i (?:prefer|like|dislike) (.+?)(?:\s+(?:over|to|for|because)|$)`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.ToLower(strings.TrimSpace(matches[1]))
	}
	return strings.ToLower(content)
}

func extractProjectKey(content string) string {
	// Extract project name from "I'm working on X"
	re := regexp.MustCompile(`(?i)i'm working on (.+?)(?:\s+(?:project|app|feature|with)|$)`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.ToLower(strings.TrimSpace(matches[1]))
	}
	return strings.ToLower(content)
}

func extractLearningKey(content string) string {
	// Extract subject from "I'm learning X"
	re := regexp.MustCompile(`(?i)i'm learning (.+?)(?:\s+(?:language|framework|about)|$)`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.ToLower(strings.TrimSpace(matches[1]))
	}
	return strings.ToLower(content)
}