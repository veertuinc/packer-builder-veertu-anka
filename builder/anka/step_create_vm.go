package anka

import (
	"fmt"
	"math/rand"

	"github.com/hashicorp/packer/packer"
	"github.com/mitchellh/multistep"
)

type StepCreateVM struct {
}

func (s *StepCreateVM) Run(state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	client := state.Get("client").(*Client)
	ui := state.Get("ui").(packer.Ui)
	sourceVM := config.SourceVMName

	if sourceVM == "" {
		ui.Say("Creating a new disk from installer, this will take a while")
		imageID, err := client.CreateDisk(CreateDiskParams{
			DiskSize:     config.DiskSize,
			InstallerApp: config.InstallerApp,
		})
		if err != nil {
			state.Put("error", err)
			ui.Error(err.Error())
		}

		ui.Say(fmt.Sprintf("Creating disk image from app: %s", config.InstallerApp))

		ui.Say("Creating a new virtual machine")
		sourceVM = fmt.Sprintf("anka-disk-base-%s", randSeq(10))
		_, err = client.Create(CreateParams{
			ImageID:  imageID,
			RamSize:  "2G",
			CPUCount: 2,
			Name:     sourceVM,
		})
		if err != nil {
			err := fmt.Errorf("Error creating VM: %v", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		ui.Say(fmt.Sprintf("VM %s was created", sourceVM))
	}

	descr, err := client.Describe(sourceVM)
	if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	vmName := fmt.Sprintf("anka-packer-%s", randSeq(10))

	ui.Say(fmt.Sprintf("Cloning source VM %s into a new virtual machine %s", sourceVM, vmName))
	err = client.Clone(CloneParams{
		VMName:     vmName,
		SourceUUID: descr.UUID,
	})
	if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Starting up %s", vmName))
	err = client.Start(vmName)
	if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	state.Put("vm_name", vmName)
	return multistep.ActionContinue
}

func (s *StepCreateVM) Cleanup(state multistep.StateBag) {
	client := state.Get("client").(*Client)
	ui := state.Get("ui").(packer.Ui)
	vmName := state.Get("vm_name").(string)

	err := client.Stop(vmName)
	if err != nil {
		ui.Error(err.Error())
	}
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
