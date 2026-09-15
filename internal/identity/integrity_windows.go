package identity

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

func checkExecutableIntegrity(path string) error {
	currentUser, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	for current := path; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("cannot inspect original gh integrity")
		}
		sd, err := windows.GetNamedSecurityInfo(current, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
		if err != nil {
			return fmt.Errorf("cannot inspect executable permissions")
		}
		acl, _, err := sd.DACL()
		if err != nil || acl == nil {
			return fmt.Errorf("executable requires a restrictive DACL")
		}
		mask := uint32(windows.GENERIC_ALL | windows.GENERIC_WRITE | windows.WRITE_DAC | windows.WRITE_OWNER | windows.DELETE)
		if info.IsDir() {
			mask |= 0x0040 // FILE_DELETE_CHILD
		} else {
			mask |= windows.FILE_WRITE_DATA | windows.FILE_APPEND_DATA
		}
		for i := uint32(0); i < uint32(acl.AceCount); i++ {
			var ace *windows.ACCESS_ALLOWED_ACE
			if err := windows.GetAce(acl, i, &ace); err != nil {
				return err
			}
			if ace.Header.AceFlags&windows.INHERIT_ONLY_ACE != 0 || ace.Header.AceType == windows.ACCESS_DENIED_ACE_TYPE {
				continue
			}
			if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
				return fmt.Errorf("unsupported executable permission type")
			}
			sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart)).String()
			if uint32(ace.Mask)&mask != 0 && sid != currentUser.User.Sid.String() && sid != "S-1-5-18" && sid != "S-1-5-32-544" && !strings.HasPrefix(sid, "S-1-5-80-") {
				return fmt.Errorf("original gh or a parent directory is writable outside the operator and system services")
			}
		}
		if filepath.Dir(current) == current {
			break
		}
	}
	return nil
}
