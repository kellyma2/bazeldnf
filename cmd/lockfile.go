package main

import (
	"github.com/rmohr/bazeldnf/pkg/api/bazeldnf"
	"github.com/rmohr/bazeldnf/pkg/bazel"
	"github.com/rmohr/bazeldnf/pkg/reducer"
	"github.com/rmohr/bazeldnf/pkg/repo"
	"github.com/rmohr/bazeldnf/pkg/sat"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type lockfileOpts struct {
	nobest         bool
	arch           string
	baseSystem     string
	repofiles      []string
	configname     string
	lockfile       string
	name           string
	dontAllowRegex []string
	onlyAllowRegex []string
}

var lockfileopts = lockfileOpts{}

func NewLockFileCmd() *cobra.Command {

	lockfileCmd := &cobra.Command{
		Use:   "lockfile",
		Short: "Writes a lockfile",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, required []string) error {
			repos, err := repo.LoadRepoFiles(lockfileopts.repofiles)
			if err != nil {
				return err
			}
			repoReducer := reducer.NewRepoReducer(repos, nil, "", lockfileopts.baseSystem, lockfileopts.arch, ".bazeldnf")
			logrus.Info("Loading packages.")
			if err := repoReducer.Load(); err != nil {
				return err
			}
			logrus.Info("Initial reduction of involved packages.")
			matched, involved, err := repoReducer.Resolve(required, true)
			if err != nil {
				return err
			}

			config := &bazeldnf.Config{
				Name: lockfileopts.configname,
				RPMs: []bazeldnf.RPM{},
			}
			if len(matched) > 0 {
				solver := sat.NewResolver(lockfileopts.nobest)
				logrus.Info("Loading involved packages into the lock file builder.")
				err = solver.LoadInvolvedPackages(involved, lockfileopts.dontAllowRegex, lockfileopts.onlyAllowRegex)
				if err != nil {
					return err
				}
				logrus.Info("Adding required packages to the lock file builder.")
				err = solver.ConstructRequirements(matched)
				if err != nil {
					return err
				}
				logrus.Info("Solving.")
				install, _, _, err := solver.Resolve()
				if err != nil {
					return err
				}

				err = bazel.AddConfigRPMs(config, install, lockfileopts.arch)
				if err != nil {
					return err
				}
			}

			logrus.Info("Writing bazel files.")
			return bazel.WriteLockFile(config, lockfileopts.lockfile)
		},
	}

	lockfileCmd.Flags().StringVar(&lockfileopts.baseSystem, "basesystem", "fedora-release-container", "base system to use (e.g. fedora-release-server, centos-stream-release, ...)")
	lockfileCmd.Flags().StringVarP(&lockfileopts.arch, "arch", "a", "x86_64", "target architecture")
	lockfileCmd.Flags().BoolVarP(&lockfileopts.nobest, "nobest", "n", false, "allow picking versions which are not the newest")
	lockfileCmd.Flags().StringArrayVarP(&lockfileopts.repofiles, "repofile", "r", []string{"repo.yaml"}, "repository information file. Can be specified multiple times. Will be used by default if no explicit inputs are provided.")
	lockfileCmd.Flags().StringVar(&lockfileopts.configname, "configname", "rpms", "config name to use in lockfile")
	lockfileCmd.Flags().StringVar(&lockfileopts.lockfile, "lockfile", "", "lockfile for RPMs")
	lockfileCmd.Flags().StringArrayVar(&lockfileopts.dontAllowRegex, "force-ignore-with-dependencies", []string{}, "Packages matching these regex patterns will not be installed. Allows force-removing unwanted dependencies. Be careful, this can lead to hidden missing dependencies.")
	lockfileCmd.Flags().StringArrayVar(&lockfileopts.onlyAllowRegex, "only-allow", []string{}, "Packages matching these regex patterns may be installed. Allows scoping dependencies. Be careful, this can lead to hidden missing dependencies.")
	return lockfileCmd
}
