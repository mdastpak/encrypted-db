import re

with open('cmd/server/main.go', 'r', encoding='utf-8') as f:
    content = f.read()

# Add embed, flag, filepath imports
content = content.replace(
    '"context"\n\t"fmt"',
    '"context"\n\t"embed"\n\t"flag"\n\t"fmt"'
)
content = content.replace(
    '"os/signal"\n\t"syscall"',
    '"os/signal"\n\t"path/filepath"\n\t"syscall"'
)

# Add variables and flag parsing before main
old_main_start = 'func main() {\n\tconfig.LoadConfig()'
new_main_start = '''var (
\tconfigPath    string
\tmigrationsPath string
)

func init() {
\tflag.StringVar(&configPath, "config", "config/config.yaml", "Path to config file")
\tflag.StringVar(&migrationsPath, "migrations", "file://internal/db/migrations", "Path to migrations")
}

func main() {
\tflag.Parse()
\tconfig.SetConfigPath(configPath)
\tconfig.SetMigrationsPath(migrationsPath)
\tconfig.LoadConfig()'''
content = content.replace(old_main_start, new_main_start)

with open('cmd/server/main.go', 'w', encoding='utf-8') as f:
    f.write(content)
print('Done')
