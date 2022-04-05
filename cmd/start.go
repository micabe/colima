package cmd

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/abiosoft/colima/cmd/root"
	"github.com/abiosoft/colima/config"
	"github.com/abiosoft/colima/config/configmanager"
	"github.com/abiosoft/colima/environment"
	"github.com/abiosoft/colima/environment/container/docker"
	"github.com/abiosoft/colima/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start [profile]",
	Short: "start Colima",
	Long: `Start Colima with the specified container runtime (and kubernetes if --with-kubernetes is passed).
The --runtime, --disk and --arch flags are only used on initial start and ignored on subsequent starts.
`,
	Example: "  colima start\n" +
		"  colima start --runtime containerd\n" +
		"  colima start --with-kubernetes\n" +
		"  colima start --runtime containerd --with-kubernetes\n" +
		"  colima start --cpu 4 --memory 8 --disk 100\n" +
		"  colima start --arch aarch64\n" +
		"  colima start --dns 1.1.1.1 --dns 8.8.8.8",
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return newApp().Start(startCmdArgs.Config)
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		current, err := configmanager.Load()
		if err != nil {
			// not fatal, will proceed with defaults
			log.Warnln(fmt.Errorf("config load failed: %w", err))
			log.Warnln("reverting to default settings")
		}

		// use default config
		if current.Empty() {
			return nil
		}

		// runtime, ssh port, disk size, kubernetes version and arch are only effective on VM create
		// set it to the current settings
		startCmdArgs.Runtime = current.Runtime
		startCmdArgs.Disk = current.Disk
		startCmdArgs.Arch = current.Arch
		startCmdArgs.Kubernetes.Version = current.Kubernetes.Version

		// use current settings for unchanged configs
		// otherwise may be reverted to their default values.
		if !cmd.Flag("with-kubernetes").Changed {
			startCmdArgs.Kubernetes.Enabled = current.Kubernetes.Enabled
		}
		if !cmd.Flag("cpu").Changed {
			startCmdArgs.CPU = current.CPU
		}
		if !cmd.Flag("cpu-type").Changed {
			startCmdArgs.CPUType = current.CPUType
		}
		if !cmd.Flag("memory").Changed {
			startCmdArgs.Memory = current.Memory
		}
		if !cmd.Flag("mount").Changed {
			startCmdArgs.Mounts = current.Mounts
		}
		if !cmd.Flag("ssh-agent").Changed {
			startCmdArgs.ForwardAgent = current.ForwardAgent
		}
		if !cmd.Flag("dns").Changed {
			startCmdArgs.DNS = current.DNS
		}
		if util.MacOS() {
			if !cmd.Flag("network-address").Changed {
				startCmdArgs.Network.Address = current.Network.Address
			}
			if !cmd.Flag("network-user-mode").Changed {
				startCmdArgs.Network.UserMode = current.Network.UserMode
			}
		}

		log.Println("using", current.Runtime, "runtime")

		// remaining settings do not survive VM reboots.
		return nil
	},
	PostRunE: func(cmd *cobra.Command, args []string) error {
		return configmanager.Save(startCmdArgs.Config)
	},
}

const (
	defaultCPU               = 2
	defaultMemory            = 2
	defaultDisk              = 60
	defaultKubernetesVersion = "v1.23.4"
)

var startCmdArgs struct {
	config.Config
}

func init() {
	runtimes := strings.Join(environment.ContainerRuntimes(), ", ")
	defaultArch := string(environment.Arch(runtime.GOARCH).Value())

	root.Cmd().AddCommand(startCmd)
	startCmd.Flags().StringVarP(&startCmdArgs.Runtime, "runtime", "r", docker.Name, "container runtime ("+runtimes+")")
	startCmd.Flags().IntVarP(&startCmdArgs.CPU, "cpu", "c", defaultCPU, "number of CPUs")
	startCmd.Flags().StringVar(&startCmdArgs.CPUType, "cpu-type", "", "the Qemu CPU type")
	startCmd.Flags().IntVarP(&startCmdArgs.Memory, "memory", "m", defaultMemory, "memory in GiB")
	startCmd.Flags().IntVarP(&startCmdArgs.Disk, "disk", "d", defaultDisk, "disk size in GiB")
	startCmd.Flags().StringVarP(&startCmdArgs.Arch, "arch", "a", defaultArch, "architecture (aarch64, x86_64)")

	// network
	if util.MacOS() {
		startCmd.Flags().BoolVar(&startCmdArgs.Network.Address, "network-address", true, "assign reachable IP address to the VM")
		startCmd.Flags().BoolVar(&startCmdArgs.Network.UserMode, "network-user-mode", true, "use Qemu user-mode network for internet, ignored if --network-address=false")
	}

	// mounts
	startCmd.Flags().StringSliceVarP(&startCmdArgs.Mounts, "mount", "v", nil, "directories to mount, suffix ':w' for writable")

	// ssh agent
	startCmd.Flags().BoolVarP(&startCmdArgs.ForwardAgent, "ssh-agent", "s", false, "forward SSH agent to the VM")

	// k8s
	startCmd.Flags().BoolVarP(&startCmdArgs.Kubernetes.Enabled, "with-kubernetes", "k", false, "start VM with Kubernetes")
	startCmd.Flags().StringVar(&startCmdArgs.Kubernetes.Version, "kubernetes-version", defaultKubernetesVersion, "the Kubernetes version")
	// not so familiar with k3s versioning atm, hide for now.
	_ = startCmd.Flags().MarkHidden("kubernetes-version")

	// not sure of the usefulness of env vars for now considering that interactions will be with the containers, not the VM.
	// leaving it undocumented until there is a need.
	startCmd.Flags().StringToStringVarP(&startCmdArgs.Env, "env", "e", nil, "environment variables for the VM")
	_ = startCmd.Flags().MarkHidden("env")

	startCmd.Flags().IPSliceVarP(&startCmdArgs.DNS, "dns", "n", nil, "DNS servers for the VM")
}
