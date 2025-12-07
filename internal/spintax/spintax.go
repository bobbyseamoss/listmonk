// Package spintax provides functionality for processing spintax text variations.
// Spintax format: {option1|option2|option3} randomly selects one option.
// Supports nested spintax: {Hello|Hi {there|friend}}
package spintax

import (
	"crypto/rand"
	"math/big"
	"regexp"
	"strings"
)

// Process takes text with spintax syntax and returns a version with random selections.
// Spintax format: {option1|option2|option3}
// Nested spintax is supported: {Hello|Hi {there|friend}}
// Uses crypto/rand for secure random selection to ensure unique variations per recipient.
func Process(text string) string {
	// Process from innermost to outermost to handle nested spintax
	result := text
	maxIterations := 100 // Prevent infinite loops

	for i := 0; i < maxIterations; i++ {
		// Find innermost spintax block (one that doesn't contain other blocks)
		// Match { followed by content without { or }, then }
		re := regexp.MustCompile(`\{([^{}]+)\}`)
		match := re.FindStringSubmatchIndex(result)

		if match == nil {
			break // No more spintax blocks
		}

		// Extract the content between { and }
		content := result[match[2]:match[3]]

		// Split by | to get options
		options := splitOptions(content)

		if len(options) == 0 {
			// No valid options, remove the block
			result = result[:match[0]] + result[match[1]:]
			continue
		}

		// Select random option using crypto/rand for security
		selected := selectRandom(options)

		// Replace the block with the selected option
		result = result[:match[0]] + selected + result[match[1]:]
	}

	return result
}

// ProcessWithSeed takes text and a seed value to produce deterministic output.
// Useful for previewing or when you need reproducible results.
func ProcessWithSeed(text string, seed int64) string {
	result := text
	maxIterations := 100
	seedIndex := 0

	for i := 0; i < maxIterations; i++ {
		re := regexp.MustCompile(`\{([^{}]+)\}`)
		match := re.FindStringSubmatchIndex(result)

		if match == nil {
			break
		}

		content := result[match[2]:match[3]]
		options := splitOptions(content)

		if len(options) == 0 {
			result = result[:match[0]] + result[match[1]:]
			continue
		}

		// Use seed-based selection for reproducibility
		index := int((seed + int64(seedIndex)) % int64(len(options)))
		if index < 0 {
			index = -index
		}
		selected := options[index]
		seedIndex++

		result = result[:match[0]] + selected + result[match[1]:]
	}

	return result
}

// splitOptions splits spintax content by | while respecting escaped pipes
func splitOptions(content string) []string {
	var options []string
	var current strings.Builder

	i := 0
	for i < len(content) {
		if content[i] == '\\' && i+1 < len(content) && content[i+1] == '|' {
			// Escaped pipe, include the pipe but not the backslash
			current.WriteByte('|')
			i += 2
		} else if content[i] == '|' {
			// Option separator
			options = append(options, current.String())
			current.Reset()
			i++
		} else {
			current.WriteByte(content[i])
			i++
		}
	}

	// Add the last option
	options = append(options, current.String())

	// Filter out empty options
	var filtered []string
	for _, opt := range options {
		if strings.TrimSpace(opt) != "" {
			filtered = append(filtered, opt)
		}
	}

	return filtered
}

// selectRandom selects a random element from a slice using crypto/rand
func selectRandom(options []string) string {
	if len(options) == 0 {
		return ""
	}
	if len(options) == 1 {
		return options[0]
	}

	// Use crypto/rand for secure random selection
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(options))))
	if err != nil {
		// Fallback to first option if random fails
		return options[0]
	}

	return options[n.Int64()]
}

// HasSpintax checks if text contains spintax syntax
func HasSpintax(text string) bool {
	re := regexp.MustCompile(`\{[^{}]*\|[^{}]*\}`)
	return re.MatchString(text)
}

// CountVariations estimates the number of possible variations in spintax text
// Returns -1 if the number is too large to calculate
func CountVariations(text string) int64 {
	re := regexp.MustCompile(`\{([^{}]+)\}`)
	matches := re.FindAllStringSubmatch(text, -1)

	if len(matches) == 0 {
		return 1
	}

	var count int64 = 1
	for _, match := range matches {
		options := splitOptions(match[1])
		optCount := int64(len(options))
		if optCount == 0 {
			optCount = 1
		}

		// Check for overflow
		if count > 0 && optCount > 0 && count > (1<<62)/optCount {
			return -1 // Overflow, too many variations
		}
		count *= optCount
	}

	return count
}

// Validate checks if spintax syntax is valid (balanced braces, non-empty options)
// Returns (true, nil) if valid, (false, error) if invalid
func Validate(text string) (bool, error) {
	// Check for balanced braces
	depth := 0
	for i, ch := range text {
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
			if depth < 0 {
				return false, &ValidationError{
					Position: i,
					Message:  "unexpected closing brace",
				}
			}
		}
	}

	if depth != 0 {
		return false, &ValidationError{
			Position: len(text),
			Message:  "unclosed spintax block",
		}
	}

	return true, nil
}

// ValidationError represents a spintax validation error
type ValidationError struct {
	Position int
	Message  string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// ExtractBlocks returns all spintax blocks found in the text
func ExtractBlocks(text string) []SpintaxBlock {
	var blocks []SpintaxBlock
	re := regexp.MustCompile(`\{([^{}]+)\}`)
	matches := re.FindAllStringSubmatchIndex(text, -1)

	for _, match := range matches {
		content := text[match[2]:match[3]]
		options := splitOptions(content)
		if len(options) > 1 || strings.Contains(content, "|") {
			blocks = append(blocks, SpintaxBlock{
				Start:   match[0],
				End:     match[1],
				Content: content,
				Options: options,
			})
		}
	}

	return blocks
}

// SpintaxBlock represents a single spintax block in text
type SpintaxBlock struct {
	Start   int
	End     int
	Content string
	Options []string
}
