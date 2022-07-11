package main

import (
	"fmt"
	"strconv"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ebs"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

type Config struct {
	AZ       string
	Instance string
	Ami      string
	Vol      []Volume
}

type Volume struct {
	Type        string
	Size        int
	Iops        int
	MultiAttach bool
	Throughput  int
	// DeviceName, https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/device_naming.html
	DeviceName string
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		var bConfig Config
		cfg := config.New(ctx, "benchmark")
		cfg.RequireObject("config", &bConfig)

		// Create a new security group for port 80
		sg, err := ec2.NewSecurityGroup(ctx, "benchmark-sg", &ec2.SecurityGroupArgs{
			Ingress: ec2.SecurityGroupIngressArray{
				ec2.SecurityGroupIngressArgs{
					Protocol:   pulumi.String("tcp"),
					FromPort:   pulumi.Int(80),
					ToPort:     pulumi.Int(80),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
			},
		}, nil)
		if err != nil {
			return err
		}

		var volList []*ebs.Volume
		var args ebs.VolumeArgs
		var deviceNameList []string
		for k, v := range bConfig.Vol {
			switch v.Type {
			case "gp2":
				args = ebs.VolumeArgs{
					AvailabilityZone: pulumi.String(bConfig.AZ),
					Size:             pulumi.Int(v.Size),
					Type:             pulumi.String(v.Type),
				}
			case "gp3":
				args = ebs.VolumeArgs{
					AvailabilityZone: pulumi.String(bConfig.AZ),
					Size:             pulumi.Int(v.Size),
					// The amount of IOPS to provision for the disk. Only valid for `type` of `io1`, `io2` or `gp3`.
					Iops: pulumi.Int(v.Iops),
					// The throughput that the volume supports, in MiB/s. Only valid for `type` of `gp3`.
					Throughput: pulumi.Int(v.Throughput),
					// The type of EBS volume. Can be `standard`, `gp2`, `gp3`, `io1`, `io2`, `sc1` or `st1` (Default: `gp2`).
					Type: pulumi.String(v.Type),
				}
			case "io1":
				args = ebs.VolumeArgs{
					AvailabilityZone: pulumi.String(bConfig.AZ),
					Size:             pulumi.Int(v.Size),
					Iops:             pulumi.Int(v.Iops),
					Type:             pulumi.String(v.Type),
					// Specifies whether to enable Amazon EBS Multi-Attach. Multi-Attach is supported on `io1` and `io2` volumes.
					MultiAttachEnabled: pulumi.Bool(v.MultiAttach),
				}
			case "io2":
				args = ebs.VolumeArgs{
					AvailabilityZone: pulumi.String(bConfig.AZ),
					Size:             pulumi.Int(v.Size),
					Iops:             pulumi.Int(v.Iops),

					Type:               pulumi.String(v.Type),
					MultiAttachEnabled: pulumi.Bool(v.MultiAttach),
				}
			case "sc1":
				args = ebs.VolumeArgs{
					AvailabilityZone: pulumi.String(bConfig.AZ),
					Size:             pulumi.Int(v.Size),
					Type:             pulumi.String(v.Type),
				}
			case "st1":
				args = ebs.VolumeArgs{
					AvailabilityZone: pulumi.String(bConfig.AZ),
					Size:             pulumi.Int(v.Size),
					Type:             pulumi.String(v.Type),
				}

			}

			vol, err := ebs.NewVolume(ctx, "benchmark-volume-"+strconv.Itoa(k), &args, pulumi.DependsOn([]pulumi.Resource{sg}))
			if err != nil {
				return err
			}

			volList = append(volList, vol)
			deviceNameList = append(deviceNameList, v.DeviceName)
		}

		userData := fmt.Sprintf("#!/bin/bash \n ")

		instance, err := ec2.NewInstance(ctx, "benchmark-ec2", &ec2.InstanceArgs{
			Ami:              pulumi.String(bConfig.Ami),
			AvailabilityZone: pulumi.String(bConfig.AZ),
			InstanceType:     pulumi.String(bConfig.Instance),
			UserData:         pulumi.String(userData),
			Tags: pulumi.StringMap{
				"Name": pulumi.String("benchmark"),
			},
		}, pulumi.DependsOn([]pulumi.Resource{sg}))
		if err != nil {
			return err
		}

		for k, v := range volList {
			_, err = ec2.NewVolumeAttachment(ctx, "benchmark-volumeAttach-"+strconv.Itoa(k), &ec2.VolumeAttachmentArgs{
				DeviceName: pulumi.String(deviceNameList[k]),
				VolumeId:   v.ID(),
				InstanceId: instance.ID(),
			}, pulumi.DependsOn([]pulumi.Resource{instance, v}))
			if err != nil {
				return err
			}
		}

		return nil
	})
}
