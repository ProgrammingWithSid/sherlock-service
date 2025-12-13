package commands

import (
	"fmt"
	"regexp"
	"strings"
)

// Command represents a parsed @sherlock command
type Command struct {
	Name string
	Args []string
}

// Parser parses @sherlock commands from comment text
type Parser struct {
	botName string
}

func NewParser(botName string) *Parser {
	return &Parser{
		botName: botName,
	}
}

// ParseComment extracts @sherlock commands from comment text
func (p *Parser) ParseComment(commentBody string) ([]Command, error) {
	// Pattern to match @sherlock commands
	// Matches: @sherlock command [args...]
	pattern := fmt.Sprintf(`@%s\s+(\w+)(?:\s+(.+))?`, regexp.QuoteMeta(p.botName))
	re := regexp.MustCompile(pattern)

	matches := re.FindAllStringSubmatch(commentBody, -1)
	if len(matches) == 0 {
		return nil, nil
	}

	commands := make([]Command, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		cmd := Command{
			Name: strings.ToLower(strings.TrimSpace(match[1])),
		}

		// Parse arguments if present
		if len(match) > 2 && match[2] != "" {
			args := strings.Fields(match[2])
			cmd.Args = args
		}

		commands = append(commands, cmd)
	}

	return commands, nil
}

// IsCommandComment checks if a comment contains @sherlock commands
func (p *Parser) IsCommandComment(commentBody string) bool {
	pattern := fmt.Sprintf(`@%s`, regexp.QuoteMeta(p.botName))
	matched, _ := regexp.MatchString(pattern, commentBody)
	return matched
}

// ValidateCommand validates a command
func (p *Parser) ValidateCommand(cmd Command) error {
	validCommands := map[string]bool{
		"review":    true,
		"explain":   true,
		"fix":       true,
		"security":  true,
		"performance": true,
		"help":      true,
	}

	if !validCommands[cmd.Name] {
		return fmt.Errorf("unknown command: %s. Use '@sherlock help' for available commands", cmd.Name)
	}

	return nil
}

// GetHelpMessage returns the help message for @sherlock commands
func (p *Parser) GetHelpMessage() string {
	return fmt.Sprintf("## @%s Commands\n\n"+
		"Available commands:\n\n"+
		"- **@%s review** - Re-run the code review for this PR\n"+
		"- **@%s explain** - Explain the code changes in detail\n"+
		"- **@%s fix** - Generate suggested fixes for issues\n"+
		"- **@%s security** - Run a security-focused scan\n"+
		"- **@%s performance** - Analyze performance implications\n"+
		"- **@%s help** - Show this help message\n\n"+
		"Examples:\n"+
		"- `@%s review`\n"+
		"- `@%s explain src/utils.ts:45`\n"+
		"- `@%s fix`\n",
		p.botName, p.botName, p.botName, p.botName, p.botName, p.botName, p.botName, p.botName, p.botName, p.botName)
}
