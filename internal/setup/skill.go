package setup

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed skills/nudge/SKILL.md
var skills embed.FS

// WriteSkill writes the nudge skill into folder/nudge, which is what Copilot CLI's skill
// directory holds, replacing an older copy.
func WriteSkill(folder string) error {
	data, err := skills.ReadFile("skills/nudge/SKILL.md")
	if err != nil {
		return fmt.Errorf("read the embedded skill: %w", err)
	}

	dir := filepath.Join(folder, "nudge")
	if err := os.MkdirAll(dir, folderMode); err != nil {
		return fmt.Errorf("make the skill folder: %w", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), data, fileMode); err != nil {
		return fmt.Errorf("write the skill: %w", err)
	}

	return nil
}
