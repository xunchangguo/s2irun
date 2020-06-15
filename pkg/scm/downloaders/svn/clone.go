package svn

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/kubesphere/s2irun/pkg/utils/cmd"
	"github.com/kubesphere/s2irun/pkg/utils/cygpath"
	"os"
	"path/filepath"

	"github.com/kubesphere/s2irun/pkg/api"
	"github.com/kubesphere/s2irun/pkg/api/constants"
	"github.com/kubesphere/s2irun/pkg/scm/git"
	"github.com/kubesphere/s2irun/pkg/utils/fs"
)

// Clone knows how to clone a Git repository.
type Clone struct {
	fs.FileSystem
	cmd.CommandRunner
}

// Download downloads the application source code from the Git repository
// and checkout the Ref specified in the config.
func (c *Clone) Download(config *api.Config) (*git.SourceInfo, error) {
	targetSourceDir := filepath.Join(config.WorkingDir, constants.Source)
	config.WorkingSourceDir = targetSourceDir

	RevisionId := config.RevisionId

	if len(config.ContextDir) > 0 {
		targetSourceDir = filepath.Join(config.WorkingDir, constants.ContextTmp)
		glog.V(9).Infof("Downloading %q (%q) ...", config.Source, config.ContextDir)
	} else {
		glog.V(9).Infof("Downloading %q ...", config.Source)
	}

	if !config.IgnoreSubmodules {
		glog.V(2).Infof("SVN Cloning sources into %q", targetSourceDir)
	} else {
		glog.V(2).Infof("SVN Cloning sources (ignoring submodules) into %q", targetSourceDir)
	}

	err := c.Clone(config.Source, targetSourceDir, RevisionId, config.SvnAuthentication.Username, config.SvnAuthentication.Password)
	if err != nil {
		glog.V(0).Infof("error: git clone failed: %v", err)
		return nil, err
	}

	glog.V(0).Infof("Checked out to %q", RevisionId)

	//TODO svn info --xml <targetSourceDir>
	info := &git.SourceInfo{
		Location: config.Source.StringNoFragment(),
		Ref:      RevisionId,
	}
	if len(config.ContextDir) > 0 {
		originalTargetDir := filepath.Join(config.WorkingDir, constants.Source)
		c.RemoveDirectory(originalTargetDir)
		path := filepath.Join(targetSourceDir, config.ContextDir)
		err := c.CopyContents(path, originalTargetDir)
		if err != nil {
			return nil, err
		}
		c.RemoveDirectory(targetSourceDir)
	}

	if len(config.ContextDir) > 0 {
		info.ContextDir = config.ContextDir
	}

	return info, nil
}

// Clone clones a svn repository to a specific target directory.
func (c *Clone) Clone(src *git.URL, target string, RevisionId string, username string, password string) error {
	var err error

	source := *src

	if cygpath.UsingCygwinGit {
		if source.IsLocal() {
			source.URL.Path, err = cygpath.ToSlashCygwin(source.LocalPath())
			if err != nil {
				return err
			}
		}

		target, err = cygpath.ToSlashCygwin(target)
		if err != nil {
			return err
		}
	}

	sourceString := source.StringNoFragment()
	if RevisionId != "" {
		sourceString = fmt.Sprintf("%s@%s", sourceString, RevisionId)
	}
	var cloneArgs []string
	if username == "" || password == "" {
		cloneArgs = append([]string{"checkout", "--non-interactive"}, []string{sourceString, target}...)
	} else {
		cloneArgs = append([]string{"checkout", "--username", username, "--password", password, "--non-interactive"}, []string{sourceString, target}...)
	}
	opts := cmd.CommandOpts{
		Stderr: os.Stderr,
		Stdout: os.Stdout,
	}
	err = c.RunWithOptions(opts, "svn", cloneArgs...)
	if err != nil {
		glog.Errorf("Clone failed: source %s, target %s, with output %q", source, target, err)
		return err
	}
	return nil
}
