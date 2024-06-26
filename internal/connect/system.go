package connect

import (
	"strings"

	"github.com/SUSE/connect-ng/internal/credentials"
	"github.com/SUSE/connect-ng/internal/util"
	"github.com/SUSE/connect-ng/internal/zypper"
)

// Cleanup removes system credentials and installed services
func Cleanup() error {
	systemCredPath := credentials.SystemCredentialsPath(CFG.FsRoot)
	err := util.RemoveFile(systemCredPath)
	if err != nil {
		return err
	}

	// remove all suse services from zypper
	services, err := zypper.InstalledServices()
	if err != nil {
		return err
	}

	for _, service := range services {
		// NOTE: this check might not work correctly with SMT depending
		//       on the configuration (e.g. listen on https but API
		//       returns URL with http).
		if !strings.Contains(service.URL, CFG.BaseURL) {
			util.Debug.Printf("%s not in %s\n", CFG.BaseURL, service.URL)
			continue
		}
		if err := zypper.RemoveService(service.Name); err != nil {
			return err
		}

	}
	return nil
}

// UpdateCertificates runs system certificate update command
func UpdateCertificates() error {
	cmd := []string{"/usr/sbin/update-ca-certificates"}
	_, err := util.Execute(cmd, []int{0})
	if err != nil {
		return err
	}
	// reload CA certs in Go
	return ReloadCertPool()
}
