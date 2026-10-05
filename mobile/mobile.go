// ==============================================================================
// ZarvandVPN
// Author: ZarvandVPN Project
// Github: https://github.com/AnishtayiN
// ==============================================================================

package mobile

import (
	"zarvand/internal/version"
)

// Version returns the library version.
func Version() string {
	return version.GetVersion()
}
