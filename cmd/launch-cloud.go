package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/beringresearch/macpine/host"
	"github.com/beringresearch/macpine/qemu"
	"github.com/beringresearch/macpine/utils"
	"github.com/spf13/cobra"
)

// launchCloudCmd launchCloudes an Alpine instance
var launchCloudCmd = &cobra.Command{
	Use:     "launch-cloud",
	Short:   "Create and start a cloud-init enabled instance.",
	Run:     launchCloud,
	Aliases: []string{"create", "new", "l"},

	ValidArgsFunction: flagsLaunchCloud,
}

var machineArchCloud, imageVersionCloud, machineCPUCloud, machineMemoryCloud, machineDiskCloud, machinePortCloud, sshPortCloud, machineNameCloud, machineMountCloud string
var vmnetCloud bool

var cloudInitCloud string

func init() {
	includeLaunchCloudFlags(launchCloudCmd)
}

func includeLaunchCloudFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&imageVersionCloud, "image", "i", "alpine_3.20.3", "Image to be launchClouded.")
	cmd.Flags().StringVarP(&machineArchCloud, "arch", "a", "", "Machine architecture. Defaults to host architecture.")
	cmd.Flags().StringVarP(&machineCPUCloud, "cpu", "c", "2", "Number of CPUs to allocate.")
	cmd.Flags().StringVarP(&machineMemoryCloud, "memory", "m", "2048", "Amount of memory (in kB) to allocate.")
	cmd.Flags().StringVarP(&machineDiskCloud, "disk", "d", "5G", "Disk space (in bytes) to allocate. K, M, G suffixes are supported.")
	cmd.Flags().StringVar(&machineMountCloud, "mount", "", "Path to a host directory to be shared with the instance.")
	cmd.Flags().StringVarP(&sshPortCloud, "ssh", "s", "22", "Host port to forward for SSH (required).")
	cmd.Flags().StringVarP(&machinePortCloud, "port", "p", "", "Forward additional host ports. Multiple ports can be separated by `,`.")
	cmd.Flags().StringVarP(&machineNameCloud, "name", "n", "", "Instance name for use in `alpine` commands.")
	cmd.Flags().BoolVarP(&vmnetCloud, "shared", "v", false, "Toggle whether to use mac's native vmnet-shared mode.")
	cmd.Flags().StringVar(&cloudInitCloud, "cloud-init", "", "Path to a cloud-init yaml file to be used for the instance.")

	cmd.MarkFlagRequired("cloud-init")
}

func CorrectArgumentsCloud(imageVersion string, machineArch string, machineCPU string,
	machineMemory string, machineDisk string, sshPort string, machinePort string) error {

	// if !utils.StringSliceContains([]string{"alpine_3.20.3"}, imageVersion) {
	// 	return errors.New("unsupported image. only -i alpine_3.20.3 are currently available")
	// }

	if machineArch != "" {
		if machineArch != "aarch64" && machineArch != "x86_64" {
			return errors.New("unsupported guest architecture. use x86_64 or aarch64")
		}
	}

	int, err := strconv.Atoi(machineCPU)
	if err != nil || int < 0 {
		return errors.New("number of cpus (-c) must be a positive integer")
	}

	int, err = strconv.Atoi(machineMemory)
	if err != nil || int < 256 {
		return errors.New("memory (-m) must be a positive integer greater than 256")
	}

	var l, n []rune
	for _, r := range machineDisk {
		switch {
		case r >= 'A' && r <= 'Z':
			l = append(l, r)
		case r >= '0' && r <= '9':
			n = append(n, r)
		}
	}

	int, err = strconv.Atoi(string(n))
	if err != nil || int < 0 {
		return errors.New("disk size (-d) must be a positive integer optionally followed by K, M, or G")
	}

	if !utils.StringSliceContains([]string{"", "K", "M", "G"}, string(l)) {
		return errors.New("disk size suffix must be K, M, or G")
	}

	int, err = strconv.Atoi(sshPort)
	if err != nil || int < 0 {
		return errors.New("ssh port (-s) must be a positive integer")
	}

	_, err = utils.ParsePort(machinePort)
	if err != nil {
		return err
	}

	if machineMount != "" {
		if dir, err := os.Stat(machineMount); os.IsNotExist(err) {
			return errors.New("mount target " + machineMount + " does not exist")
		} else if !dir.IsDir() {
			return errors.New("mount target " + machineMount + " is not a directory")
		}
	}

	return nil
}

func launchCloud(cmd *cobra.Command, args []string) {

	err := CorrectArgumentsCloud(imageVersionCloud, machineArchCloud, machineCPUCloud, machineMemoryCloud, machineDiskCloud, sshPortCloud, machinePortCloud)
	if err != nil {
		log.Fatalln(err.Error())
	}

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalln(err)
	}

	if machineArch == "" {
		arch := runtime.GOARCH

		switch arch {
		case "arm64":
			machineArch = "aarch64"
		case "amd64":
			machineArch = "x86_64"
		default:
			log.Fatal("unsupported host architecture: " + arch)
		}
	}

	vmList := host.ListVMNames()

	if machineName == "" {
		machineName = utils.GenerateRandomAlias()
		for utils.StringSliceContains(vmList, machineName) { // if exists, re-randomize
			machineName = utils.GenerateRandomAlias()
		}
	} else if utils.StringSliceContains(vmList, machineName) {
		log.Fatal("instance with name \"" + machineName + "\" already exists")
	}

	macAddress, err := utils.GenerateMACAddress()
	if err != nil {
		log.Fatal(err)
	}

	machineIP := "localhost"

	machineType := "bios"
	if machineArchCloud == "aarch64" {
		machineType = "uefi"
	}

	rootPassword := "root"

	machineConfig := qemu.MachineConfig{
		Alias:        machineNameCloud,
		Image:        fmt.Sprintf("nocloud_alpine-%s-%s-%s-cloudinit-r0.qcow2", imageVersionCloud, machineArchCloud, machineType),
		Arch:         machineArchCloud,
		CPU:          machineCPUCloud,
		Memory:       machineMemoryCloud,
		Disk:         machineDiskCloud,
		Mount:        machineMountCloud,
		MachineIP:    machineIP,
		Port:         machinePortCloud,
		SSHPort:      sshPortCloud,
		MACAddress:   macAddress,
		VMNet:        vmnetCloud,
		SSHUser:      "alpine",
		SSHPassword:  "raw::root",
		RootUsername: "alpine",
		RootPassword: &rootPassword,
		CloudInit:    cloudInitCloud,
		Tags:         []string{},
	}
	machineConfig.Location = filepath.Join(userHomeDir, ".macpine", machineConfig.Alias)

	err = host.Launch(machineConfig)
	if err != nil {
		// move the log file to the .error-logs directory
		os.MkdirAll(filepath.Join(userHomeDir, ".macpine", "cache", ".error-logs"), 0755)
		name := strings.ReplaceAll(machineConfig.Alias, " ", "_") + "_" + time.Now().Format("2006-01-02_15-04-05") + ".log"
		os.Rename(filepath.Join(machineConfig.Location, "alpine.log"), filepath.Join(userHomeDir, ".macpine", "cache", ".error-logs", name))
		fmt.Println("logs are in: " + filepath.Join(userHomeDir, ".macpine", "cache", ".error-logs", name))
		fmt.Println("run this to clean up:")
		fmt.Println("rm -rf " + filepath.Join(userHomeDir, ".macpine", machineConfig.Alias))
		pid, _ := machineConfig.GetInstancePID()
		fmt.Println("kill " + strconv.Itoa(pid))

		// os.RemoveAll(machineConfig.Location)
		// pid, _ := machineConfig.GetInstancePID()
		// p, _ := os.FindProcess(pid)
		// p.Signal(syscall.SIGKILL)
		log.Fatal(err)
	}

	fmt.Println("")
	log.Println("launchClouded: " + machineNameCloud)
}

func flagsLaunchCloud(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}
