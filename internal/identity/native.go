package identity

import (
	"fmt"
	"os/user"
)

func nativePrincipal(value string) (string, error) {
	account, err := user.LookupId(value)
	if err != nil {
		account, err = user.Lookup(value)
	}
	if err != nil {
		return "", fmt.Errorf("native agent account does not exist")
	}
	current, err := user.Current()
	if err != nil {
		return "", err
	}
	if account.Uid == current.Uid || account.Uid == "0" || account.Uid == "S-1-5-18" {
		return "", fmt.Errorf("isolation requires a separate non-administrative account")
	}
	groups, err := account.GroupIds()
	if err != nil {
		return "", fmt.Errorf("cannot inspect native account groups")
	}
	for _, id := range groups {
		switch id {
		case "S-1-5-32-544", "S-1-5-32-551", "S-1-5-32-578":
			return "", fmt.Errorf("native agent account has privileged group membership")
		}
		group, err := user.LookupGroupId(id)
		if err != nil {
			return "", fmt.Errorf("cannot resolve native account group privileges")
		}
		switch group.Name {
		case "root", "sudo", "wheel", "admin", "docker", "docker-users", "podman", "lxd", "disk", "Administrators", "BUILTIN\\Administrators":
			return "", fmt.Errorf("native agent account has privileged group membership")
		}
	}
	return account.Uid, nil
}
