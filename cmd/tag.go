package cmd

import (
   "os"
	"log"
	"io/ioutil"
	"path/filepath"

	"github.com/spf13/cobra"
   "gopkg.in/yaml.v3"

	"github.com/beringresearch/macpine/host"
	"github.com/beringresearch/macpine/qemu"
)

var remove bool

func init() {
	includeTagFlag(tagCmd)
}

func includeTagFlag(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&remove, "remove", "r", false, "Remove tag(s) rather than add them.")
}

// infoCmd displays macpine machine info
var tagCmd = &cobra.Command{
	Use:   "tag NAME tag1 [tag2...]",
	Short: "Add or remove tags from an instance.",
	Run:   macpineTag,

	ValidArgsFunction:     host.AutoCompleteVMNames,
}

func macpineTag(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		log.Fatal("missing VM name")
	}
   vmName := args[0]

   tags := args[1:]
   userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	config, err := ioutil.ReadFile(filepath.Join(userHomeDir, ".macpine", vmName, "config.yaml"))
	if err != nil {
		log.Fatal(err)
	}

	var machineConfig = qemu.MachineConfig{}
	err = yaml.Unmarshal(config, &machineConfig)
	if err != nil {
		log.Fatal(err)
	}

   for _, tag := range(tags) {
      i, found := find(machineConfig.Tags, tag)
      if remove && found {
         machineConfig.Tags = append(machineConfig.Tags[:i], machineConfig.Tags[i+1:]...)
      } else if !remove && !found {
         machineConfig.Tags = append(machineConfig.Tags[:i], append([]string{tag}, machineConfig.Tags[i:]...)...)
      }
   }

	updatedConfig, err := yaml.Marshal(&machineConfig)
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(filepath.Join(userHomeDir, ".macpine", vmName, "config.yaml"), updatedConfig, 0644)
	if err != nil {
		log.Fatal(err)
	}
   log.Println(machineConfig.Tags)
}

func find(s []string, t string) (int, bool) {
   for i, e := range(s) {
      if (e == t) {
         return i, true
      } else if (e > t) {
         return i, false
      }
   }
   return len(s), false
}
