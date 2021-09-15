package connect

import (
	_ "embed" //golint
	"fmt"
	"os"
	"path"
	"strings"
	"text/template"
)

var (
	//go:embed zypper-restore.tmpl
	restoreTemplate string
)

func createZypperTarball(tarballPath, root string, paths []string) error {
	// tar reports an error if a file does not exist.
	// So we have to check this before.
	var existingPaths []string
	for _, p := range paths {
		if !fileExists(p) {
			continue
		}
		// remove leading "/" from paths to allow using them from different root
		existingPaths = append(existingPaths, strings.TrimLeft(p, "/"))
	}

	// make tarball path relative to root
	tarballPath = strings.TrimLeft(tarballPath, "/")
	tarballPathWithRoot := path.Join(root, tarballPath)

	// ensure directory exists
	if err := os.MkdirAll(path.Dir(tarballPathWithRoot), os.ModeDir); err != nil {
		return err
	}

	// using -f tarballPathWithRoot here instead of -f tarballPath because
	// tar doesn't seem to use -C root for output files
	command := []string{"tar", "cz", "-C", root, "-f", tarballPathWithRoot}
	command = append(command, existingPaths...)
	_, err := execute(command, []int{0})

	// tarball can contain sensitive data, so prevent read to non-root
	// do it for sure even if tar failed as it can contain partial content
	if fileExists(tarballPathWithRoot) {
		os.Chmod(tarballPathWithRoot, 0600)
	}

	if err != nil {
		return fmt.Errorf("failed to create backup: %v", err)
	}
	return nil
}

func createZypperRestoreScript(scriptPath, tarballPath, root string, paths []string) error {
	var data struct {
		Paths   []string
		Tarball string
	}
	// remove leading "/" from paths to allow using them from different root
	for _, p := range paths {
		data.Paths = append(data.Paths, strings.TrimLeft(p, "/"))
	}
	data.Tarball = strings.TrimLeft(tarballPath, "/")

	// make script path relative to root
	scriptPath = strings.TrimLeft(scriptPath, "/")
	scriptPathWithRoot := path.Join(root, scriptPath)

	t, err := template.New("restore-script").Parse(restoreTemplate)
	if err != nil {
		return err
	}
	f, err := os.Create(scriptPathWithRoot)
	if err != nil {
		return err
	}
	defer f.Close()

	err = t.Execute(f, data)
	if err != nil {
		return err
	}
	// allow execution of script
	os.Chmod(scriptPathWithRoot, 0744)
	return nil
}

// ZypperBackup creates backup of zypper configuration files
func ZypperBackup() error {
	root := CFG.FsRoot
	if root == "" {
		root = "/"
	}
	paths := []string{
		"/etc/zypp/repos.d",
		"/etc/zypp/credentials.d",
		"/etc/zypp/services.d",
	}
	tarballPath := "/var/adm/backup/system-upgrade/repos.tar.gz"
	if err := createZypperTarball(tarballPath, root, paths); err != nil {
		return err
	}

	scriptPath := "/var/adm/backup/system-upgrade/repos.sh"
	if err := createZypperRestoreScript(scriptPath, tarballPath, root, paths); err != nil {
		return err
	}
	return nil
}

// ZypperRestore restores zypper configuration from backup created by ZypperBackup
func ZypperRestore() error {
	root := CFG.FsRoot
	if root == "" {
		root = "/"
	}
	_, err := execute([]string{"sh",
		path.Join(root, "var/adm/backup/system-upgrade/repos.sh"),
		root}, []int{0})
	return err
}
