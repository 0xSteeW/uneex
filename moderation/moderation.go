package moderation

var Permissions = map[string]int{

	// General
	"administrator":   0x8,
	"auditLog":        0x80,
	"manageServer":    0x20,
	"manageRoles":     0x10000000,
	"manageChannels":  0x10,
	"kick":            0x2,
	"ban":             0x4,
	"invite":          0x1,
	"nickname":        0x4000000,
	"manageNicknames": 0x8000000,
	"manageEmojis":    0x40000000,
	"manageWebhooks":  0x20000000,

	// Text
	"readMessages":   0x400,
	"sendTTS":        0x1000,
	"links":          0x4000,
	"readHistory":    0x10000,
	"externalEmojis": 0x40000,
	"sendMessages":   0x800,
	"manageMessages": 0x2000,
	"attachFiles":    0x8000,
	"everyone":       0x20000,
	"reactions":      0x40,

	// Voice
	"viewChannel":   0x400,
	"connect":       0x100000,
	"mute":          0x400000,
	"move":          0x100000,
	"speak":         0x200000,
	"deafen":        0x800000,
	"voiceActivity": 0x2000000,
	"priority":      0x100,
}

// HasPermission receives a permission name (Can be checked in map PERMISSIONS) and checks if permission is inside the permission int.
func HasPermission(name string, permissionsInt int) bool {
	if permissionsInt&Permissions[name] == Permissions[name] {
		return true
	} else if permissionsInt&Permissions["administator"] == Permissions["administrator"] {
		return true
	}
	return false
}
