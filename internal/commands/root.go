package commands

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/docker/app/internal"

	"github.com/docker/app/internal/store"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	completion  string
	showVersion bool
)

// NewRootCmd returns the base root command.
func NewRootCmd(use string, dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Short:       "Docker Application",
		Long:        `A tool to build and manage Docker Applications.`,
		Use:         use,
		Annotations: map[string]string{"experimentalCLI": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				fmt.Fprintln(os.Stdout, internal.FullVersion()) //nolint:errcheck
				return nil
			}

			switch completion {
			case "bash":
				return cmd.GenBashCompletion(dockerCli.Out())
			case "zsh":
				return cmd.GenZshCompletion(dockerCli.Out())
			case "":
				// Actually unset
				return nil
			default:
				return fmt.Errorf("%q is not a supported shell", completion)
			}
		},
	}
	addCommands(cmd, dockerCli)

	cmd.Flags().StringVar(&completion, "completion", "", "Generates completion scripts for the specified shell (bash or zsh)")
	if err := cmd.Flags().MarkHidden("completion"); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register command line options: %v", err.Error()) //nolint:errcheck
		return nil
	}

	cmd.Flags().BoolVar(&showVersion, "version", false, "Print version information")
	return cmd
}

func addCommands(cmd *cobra.Command, dockerCli command.Cli) {
	cmd.AddCommand(
		installCmd(dockerCli),
		upgradeCmd(dockerCli),
		uninstallCmd(dockerCli),
		listCmd(dockerCli),
		statusCmd(dockerCli),
		initCmd(dockerCli),
		inspectCmd(dockerCli),
		renderCmd(dockerCli),
		validateCmd(),
		bundleCmd(dockerCli),
		pushCmd(dockerCli),
		pullCmd(dockerCli),
	)
}

func firstOrEmpty(list []string) string {
	if len(list) != 0 {
		return list[0]
	}
	return ""
}

func muteDockerCli(dockerCli command.Cli) func() {
	stdout := dockerCli.Out()
	stderr := dockerCli.Err()
	dockerCli.Apply(command.WithCombinedStreams(ioutil.Discard)) //nolint:errcheck // WithCombinedStreams cannot error
	return func() {
		dockerCli.Apply(command.WithOutputStream(stdout), command.WithErrorStream(stderr)) //nolint:errcheck // as above
	}
}

func prepareStores(context string) (store.BundleStore, store.InstallationStore, store.CredentialStore, error) {
	appstore, err := store.NewApplicationStore(config.Dir())
	if err != nil {
		return nil, nil, nil, err
	}
	installationStore, err := appstore.InstallationStore(context)
	if err != nil {
		return nil, nil, nil, err
	}
	bundleStore, err := appstore.BundleStore()
	if err != nil {
		return nil, nil, nil, err
	}
	credentialStore, err := appstore.CredentialStore(context)
	if err != nil {
		return nil, nil, nil, err
	}
	return bundleStore, installationStore, credentialStore, nil
}

func prepareBundleStore() (store.BundleStore, error) {
	appstore, err := store.NewApplicationStore(config.Dir())
	if err != nil {
		return nil, err
	}
	bundleStore, err := appstore.BundleStore()
	if err != nil {
		return nil, err
	}
	return bundleStore, nil
}

type parametersOptions struct {
	parametersFiles []string
	overrides       []string
}

func (o *parametersOptions) addFlags(flags *pflag.FlagSet) {
	flags.StringArrayVar(&o.parametersFiles, "parameters-file", []string{}, "Override parameters file")
	flags.StringArrayVarP(&o.overrides, "set", "s", []string{}, "Override parameter value")
}

type installerContextOptions struct {
	installerContext string
}

func (o *installerContextOptions) addFlags(flags *pflag.FlagSet) {
	if internal.Experimental == "on" {
		flags.StringVar(&o.installerContext, "installer-context", "",
			"Context on which the application's installer is installed (default: <current-context>)")
	}
}

func (o *installerContextOptions) SetDefaultInstallerContext(dockerCli command.Cli) {
	o.installerContext = getInstallerContext(o.installerContext, dockerCli.CurrentContext())
}

type credentialOptions struct {
	installerContextOptions
	credentialsets   []string
	credentials      []string
	sendRegistryAuth bool
}

func (o *credentialOptions) addFlags(flags *pflag.FlagSet) {
	o.installerContextOptions.addFlags(flags)
	flags.StringArrayVar(&o.credentialsets, "credential-set", []string{}, "Use a YAML file containing a credential set or a credential set present in the credential store")
	flags.StringArrayVar(&o.credentials, "credential", nil, "Add a single credential, additive ontop of any --credential-set used")
	flags.BoolVar(&o.sendRegistryAuth, "with-registry-auth", false, "Sends registry auth")
}

func (o *credentialOptions) CredentialSetOpts(dockerCli command.Cli, credentialStore store.CredentialStore) []credentialSetOpt {
	return []credentialSetOpt{
		addNamedCredentialSets(credentialStore, o.credentialsets),
		addCredentials(o.credentials),
		addDockerCredentials(dockerCli.CurrentContext(), dockerCli.ContextStore()),
		addRegistryCredentials(o.sendRegistryAuth, dockerCli),
	}
}

type registryOptions struct {
	insecureRegistries []string
}

func (o *registryOptions) addFlags(flags *pflag.FlagSet) {
	flags.StringSliceVar(&o.insecureRegistries, "insecure-registries", nil, "Use HTTP instead of HTTPS when pulling from/pushing to those registries")
}

type pullOptions struct {
	pull bool
}

func (o *pullOptions) addFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&o.pull, "pull", false, "Pull the bundle")
}

func setInstallerContext(dockerCli command.Cli, installerContext string) error {
	//fmt.Println("This is my setInstallerContext with:", installerContext)
	if installerContext != "" {
		if _, err := dockerCli.ContextStore().GetMetadata(installerContext); err != nil && installerContext != "default" {
			fmt.Println("This is a greeattttt error:", err)
			return err
		}
		//fmt.Println("Setting current context with:", installerContext)
		dockerCli.ConfigFile().CurrentContext = installerContext
	}
	return nil
}
