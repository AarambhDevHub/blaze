package blaze

const (
	Version   = "0.1.2"
	BuildDate = "2025-09-24"
	GoVersion = "1.24.0"
)

// GetVersion returns framework version info
func GetVersion() map[string]string {
	return map[string]string{
		"version":    Version,
		"build_date": BuildDate,
		"go_version": GoVersion,
		"framework":  "Blaze",
	}
}
