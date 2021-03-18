package anka

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	c "github.com/veertuinc/packer-builder-veertu-anka/client"
	mocks "github.com/veertuinc/packer-builder-veertu-anka/mocks"
	"gotest.tools/assert"
)

func TestHyperthreadingRun(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	client := mocks.NewMockClient(mockCtrl)
	util := mocks.NewMockUtil(mockCtrl)

	step := StepSetHyperThreading{}
	ui := packer.TestUi(t)
	ctx := context.Background()
	state := new(multistep.BasicStateBag)

	state.Put("ui", ui)
	state.Put("util", util)
	state.Put("vm_name", "foo")

	t.Run("disabled or nil htt values", func(t *testing.T) {
		config := &Config{
			EnableHtt:  false,
			DisableHtt: false,
		}

		state.Put("client", client)
		state.Put("config", config)

		stepAction := step.Run(ctx, state)
		assert.Equal(t, stepAction, multistep.ActionContinue)
	})

	t.Run("conflicting htt values", func(t *testing.T) {
		config := &Config{
			EnableHtt:  true,
			DisableHtt: true,
		}

		state.Put("client", client)
		state.Put("config", config)

		util.EXPECT().
			StepError(ui, state, fmt.Errorf("Conflicting setting enable_htt and disable_htt both true")).
			Return(multistep.ActionHalt).
			Times(1)

		stepAction := step.Run(ctx, state)
		assert.Equal(t, multistep.ActionHalt, stepAction)
	})

	t.Run("configure htt", func(t *testing.T) {
		config := &Config{
			EnableHtt:  true,
			DisableHtt: false,
		}

		state.Put("client", client)
		state.Put("config", config)

		gomock.InOrder(
			client.EXPECT().Describe("foo").Return(c.DescribeResponse{}, nil).Times(1),
			client.EXPECT().Show("foo").Return(c.ShowResponse{}, nil).Times(1),
			client.EXPECT().Stop(c.StopParams{VMName: "foo", Force: true}).Return(nil).Times(1),
			client.EXPECT().Modify("foo", "set", "cpu", "--htt").Return(nil).Times(1),
		)

		stepAction := step.Run(ctx, state)
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("enable htt when already configured", func(t *testing.T) {
		var describeResponse c.DescribeResponse
		err := json.Unmarshal(json.RawMessage(`{"CPU": {"Threads": 2}}`), &describeResponse)
		if err != nil {
			t.Fail()
		}

		config := &Config{
			EnableHtt:  true,
			DisableHtt: false,
		}

		state.Put("client", client)
		state.Put("config", config)

		client.EXPECT().Describe("foo").Return(describeResponse, nil).Times(1)

		stepAction := step.Run(ctx, state)
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("disable htt when not configured", func(t *testing.T) {
		config := &Config{
			EnableHtt:  false,
			DisableHtt: true,
		}

		state.Put("client", client)
		state.Put("config", config)

		client.EXPECT().Describe("foo").Return(c.DescribeResponse{}, nil).Times(1)

		stepAction := step.Run(ctx, state)
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("disable htt", func(t *testing.T) {
		var describeResponse c.DescribeResponse
		err := json.Unmarshal(json.RawMessage(`{"CPU": {"Threads": 2}}`), &describeResponse)
		if err != nil {
			t.Fail()
		}

		config := &Config{
			EnableHtt:  false,
			DisableHtt: true,
		}

		state.Put("client", client)
		state.Put("config", config)

		gomock.InOrder(
			client.EXPECT().Describe("foo").Return(describeResponse, nil).Times(1),
			client.EXPECT().Show("foo").Return(c.ShowResponse{}, nil).Times(1),
			client.EXPECT().Stop(c.StopParams{VMName: "foo", Force: true}).Return(nil).Times(1),
			client.EXPECT().Modify("foo", "set", "cpu", "--no-htt").Return(nil).Times(1),
		)

		stepAction := step.Run(ctx, state)
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("test rerun when vm is currently running", func(t *testing.T) {
		var showResponse c.ShowResponse
		err := json.Unmarshal(json.RawMessage(`{ "Status": "running" }`), &showResponse)
		if err != nil {
			t.Fail()
		}

		config := &Config{
			EnableHtt:  true,
			DisableHtt: false,
		}

		state.Put("client", client)
		state.Put("config", config)

		gomock.InOrder(
			client.EXPECT().Describe("foo").Return(c.DescribeResponse{}, nil).Times(1),
			client.EXPECT().Show("foo").Return(showResponse, nil).Times(1),
			client.EXPECT().Stop(c.StopParams{VMName: "foo", Force: true}).Return(nil).Times(1),
			client.EXPECT().Modify("foo", "set", "cpu", "--htt").Return(nil).Times(1),
			client.EXPECT().Start(c.StartParams{VMName: "foo"}).Return(nil).Times(1),
		)

		stepAction := step.Run(ctx, state)
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})
}