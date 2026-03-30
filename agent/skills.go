package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/skill"
)

func resolveSkillsDir() (string, bool) {
	skillsDir := strings.TrimSpace("/home/lsk/projects/eino-demo/ext/skills")
	if skillsDir == "" {
		return "", false
	}
	if absSkillsDir, absErr := filepath.Abs(skillsDir); absErr == nil {
		skillsDir = absSkillsDir
	}
	fi, err := os.Stat(skillsDir)
	if err != nil || !fi.IsDir() {
		return "", false
	}
	return skillsDir, true
}

func SkillMiddleware(ctx context.Context, backend *local.Local) (adk.ChatModelAgentMiddleware, error) {
	skillsDir, found := resolveSkillsDir()
	if found {
		skillBackend, err := skill.NewBackendFromFilesystem(ctx, &skill.BackendFromFilesystemConfig{
			Backend: backend,
			BaseDir: skillsDir,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to load skill backend: %w", err)
		}

		return skill.NewMiddleware(ctx, &skill.Config{
			Backend: skillBackend,
		})
	}
	return nil, fmt.Errorf("skill dir not found")
}
