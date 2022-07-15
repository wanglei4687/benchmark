package main

import (
	"fmt"
	"strconv"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ebs"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/pricing"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

type Config struct {
	keyName        string
	Region         string
	BURL           string
	AZ             string
	Instance       string
	Ami            string
	Vol            []Volume
	Capacitystatus string
	Operation      string
}

type Volume struct {
	Type string
	// General Purpose or Provisioned IOPS
	Version    string
	File       string
	VPath      string
	Size       int
	Iops       int
	Throughput int
	// DeviceName, https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/device_naming.html
	DeviceName string
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		var bConfig Config
		cfg := config.New(ctx, "benchmark")
		cfg.RequireObject("config", &bConfig)

		//  This data source is only available in a us-east-1 or ap-south-1 provider
		instancePrice, err := pricing.GetProduct(ctx, &pricing.GetProductArgs{
			ServiceCode: "AmazonEC2",
			Filters: []pricing.GetProductFilter{
				pricing.GetProductFilter{
					Field: "instanceType",
					Value: bConfig.Instance,
				},
				pricing.GetProductFilter{
					Field: "regionCode",
					Value: bConfig.Region,
				},
				pricing.GetProductFilter{
					Field: "operatingSystem",
					Value: "linux",
				},
				pricing.GetProductFilter{
					Field: "capacitystatus",
					Value: bConfig.Capacitystatus,
				},
				pricing.GetProductFilter{
					Field: "tenancy",
					Value: "Shared",
				},
				// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/billing-info-fields.html
				pricing.GetProductFilter{
					Field: "operation",
					Value: bConfig.Operation,
				},
			},
		}, nil)
		if err != nil {
			return err
		}

		insT := fmt.Sprintf("%s:%s", bConfig.Region, bConfig.Instance)
		ctx.Export(insT, pulumi.String(instancePrice.Result))

		var volPriceList []*pricing.GetProductResult
		for _, v := range bConfig.Vol {
			if v.Type != "local" {
				vol, err := pricing.GetProduct(ctx, &pricing.GetProductArgs{
					ServiceCode: "AmazonEC2",
					Filters: []pricing.GetProductFilter{
						pricing.GetProductFilter{
							Field: "regionCode",
							Value: bConfig.Region,
						},
						pricing.GetProductFilter{
							Field: "volumeType",
							Value: v.Version,
						},
						pricing.GetProductFilter{
							Field: "volumeApiName",
							Value: v.Type,
						},
					},
				}, nil)
				if err != nil {
					return err
				}

				volPriceList = append(volPriceList, vol)

				voT := fmt.Sprintf("%s:%s:%s ", bConfig.Region, v.Version, v.Type)

				ctx.Export(voT, pulumi.String(vol.Result))
			}
		}

		// Create a new security group for port 80
		sg, err := ec2.NewSecurityGroup(ctx, "benchmark-sg", &ec2.SecurityGroupArgs{
			Ingress: ec2.SecurityGroupIngressArray{
				ec2.SecurityGroupIngressArgs{
					Protocol:   pulumi.String("tcp"),
					FromPort:   pulumi.Int(80),
					ToPort:     pulumi.Int(80),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
				ec2.SecurityGroupIngressArgs{
					Protocol:   pulumi.String("tcp"),
					FromPort:   pulumi.Int(22),
					ToPort:     pulumi.Int(22),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
			},
			Egress: ec2.SecurityGroupEgressArray{
				ec2.SecurityGroupEgressArgs{
					Protocol:   pulumi.String("-1"),
					FromPort:   pulumi.Int(0),
					ToPort:     pulumi.Int(0),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
			},
			Name: pulumi.String("benchmark-sg"),
		}, nil)
		if err != nil {
			return err
		}

		ctx.Export("security group", sg.Name)

		var volList []*ebs.Volume
		var args ebs.VolumeArgs
		var deviceNameList []string
		var pathList []string
		var formatCommand string
		for k, v := range bConfig.Vol {
			switch v.Type {
			case "gp2":
				args = ebs.VolumeArgs{
					AvailabilityZone: pulumi.String(bConfig.AZ),
					Size:             pulumi.Int(v.Size),
					Type:             pulumi.String(v.Type),
				}

				deviceP := len(v.DeviceName)
				vPath := v.VPath + v.DeviceName[deviceP-1:]
				mountPath := "/benchmark" + v.DeviceName[deviceP-1:]

				formatCommand += fmt.Sprintf("sudo mkfs -t %s %s \n", v.File, vPath)
				formatCommand += fmt.Sprintf("sudo mkdir %s \n", mountPath)
				formatCommand += fmt.Sprintf("sudo mount %s %s \n", vPath, mountPath)
				pathList = append(pathList, mountPath)
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
				deviceP := len(v.DeviceName)
				vPath := v.VPath + v.DeviceName[deviceP-1:]
				mountPath := "/benchmark" + v.DeviceName[deviceP-1:]

				formatCommand += fmt.Sprintf("sudo mkfs -t %s %s \n", v.File, vPath)
				formatCommand += fmt.Sprintf("sudo mkdir %s \n", mountPath)
				formatCommand += fmt.Sprintf("sudo mount %s %s \n", vPath, mountPath)
				pathList = append(pathList, mountPath)
			case "io1":
				args = ebs.VolumeArgs{
					AvailabilityZone: pulumi.String(bConfig.AZ),
					Size:             pulumi.Int(v.Size),
					Iops:             pulumi.Int(v.Iops),
					Type:             pulumi.String(v.Type),
					// Specifies whether to enable Amazon EBS Multi-Attach. Multi-Attach is supported on `io1` and `io2` volumes.
					MultiAttachEnabled: pulumi.Bool(false),
				}
				deviceP := len(v.DeviceName)
				vPath := v.VPath + v.DeviceName[deviceP-1:]
				mountPath := "/benchmark" + v.DeviceName[deviceP-1:]

				formatCommand += fmt.Sprintf("sudo mkfs -t %s %s \n", v.File, vPath)
				formatCommand += fmt.Sprintf("sudo mkdir %s \n", mountPath)
				formatCommand += fmt.Sprintf("sudo mount %s %s \n", vPath, mountPath)
				pathList = append(pathList, mountPath)
			case "io2":
				args = ebs.VolumeArgs{
					AvailabilityZone: pulumi.String(bConfig.AZ),
					Size:             pulumi.Int(v.Size),
					Iops:             pulumi.Int(v.Iops),

					Type:               pulumi.String(v.Type),
					MultiAttachEnabled: pulumi.Bool(false),
				}
				deviceP := len(v.DeviceName)
				vPath := v.VPath + v.DeviceName[deviceP-1:]
				mountPath := "/benchmark" + v.DeviceName[deviceP-1:]

				formatCommand += fmt.Sprintf("sudo mkfs -t %s %s \n", v.File, vPath)
				formatCommand += fmt.Sprintf("sudo mkdir %s \n", mountPath)
				formatCommand += fmt.Sprintf("sudo mount %s %s \n", vPath, mountPath)
				pathList = append(pathList, mountPath)
			case "sc1":
				args = ebs.VolumeArgs{
					AvailabilityZone: pulumi.String(bConfig.AZ),
					Size:             pulumi.Int(v.Size),
					Type:             pulumi.String(v.Type),
				}
				deviceP := len(v.DeviceName)
				vPath := v.VPath + v.DeviceName[deviceP-1:]
				mountPath := "/benchmark" + v.DeviceName[deviceP-1:]

				formatCommand += fmt.Sprintf("sudo mkfs -t %s %s \n", v.File, vPath)
				formatCommand += fmt.Sprintf("sudo mkdir %s \n", mountPath)
				formatCommand += fmt.Sprintf("sudo mount %s %s \n", vPath, mountPath)
				pathList = append(pathList, mountPath)
			case "st1":
				args = ebs.VolumeArgs{
					AvailabilityZone: pulumi.String(bConfig.AZ),
					Size:             pulumi.Int(v.Size),
					Type:             pulumi.String(v.Type),
				}
				deviceP := len(v.DeviceName)
				vPath := v.VPath + v.DeviceName[deviceP-1:]
				mountPath := "/benchmark" + v.DeviceName[deviceP-1:]

				formatCommand += fmt.Sprintf("sudo mkfs -t %s %s \n", v.File, vPath)
				formatCommand += fmt.Sprintf("sudo mkdir %s \n", mountPath)
				formatCommand += fmt.Sprintf("sudo mount %s %s \n", vPath, mountPath)
				pathList = append(pathList, mountPath)
			case "local":
				pathList = append(pathList, v.DeviceName)
			default:
				err = fmt.Errorf("Please config correctly~~~")
				return err

			}

			vol, err := ebs.NewVolume(ctx, "benchmark-volume-"+strconv.Itoa(k), &args, pulumi.DependsOn([]pulumi.Resource{sg}))
			if err != nil {
				return err
			}

			volList = append(volList, vol)
			deviceNameList = append(deviceNameList, v.DeviceName)

		}

		var command string
		for k := range pathList {
			command += fmt.Sprintf("sudo ./fsyncpref --path %s >> index.html \n", pathList[k])
		}

		userData := fmt.Sprintf("#!/bin/bash \n %s wget %s  \n chmod 755 fsyncpref  \n %s echo \"done\" >> index.html  \n nohup python -m SimpleHTTPServer 80 &",
			formatCommand, bConfig.BURL, command)

		ctx.Export("userData", pulumi.String(userData))

		kp, err := ec2.LookupKeyPair(ctx, &ec2.LookupKeyPairArgs{
			KeyName: pulumi.StringRef(bConfig.keyName),
			Filters: []ec2.GetKeyPairFilter{
				ec2.GetKeyPairFilter{
					Name: "tag:org",
					Values: []string{
						"matrixorigin",
					},
				},
			},
		}, nil)
		if err != nil {
			return err
		}

		instance, err := ec2.NewInstance(ctx, "benchmark-ec2", &ec2.InstanceArgs{
			VpcSecurityGroupIds: pulumi.StringArray{sg.ID()},
			Ami:                 pulumi.String(bConfig.Ami),
			AvailabilityZone:    pulumi.String(bConfig.AZ),
			InstanceType:        pulumi.String(bConfig.Instance),
			UserData:            pulumi.String(userData),
			KeyName:             pulumi.String(*kp.KeyName),
			Tags: pulumi.StringMap{
				"Name": pulumi.String("benchmark"),
			},
		}, pulumi.DependsOn([]pulumi.Resource{sg}))
		if err != nil {
			return err
		}

		for k, v := range volList {
			_, err = ec2.NewVolumeAttachment(ctx, "benchmark-volumeAttach-"+strconv.Itoa(k), &ec2.VolumeAttachmentArgs{
				DeviceName:  pulumi.String(deviceNameList[k]),
				VolumeId:    v.ID(),
				InstanceId:  instance.ID(),
				ForceDetach: pulumi.Bool(true),
				SkipDestroy: pulumi.Bool(true),
			}, pulumi.DependsOn([]pulumi.Resource{instance, v}))
			if err != nil {
				return err
			}
		}

		ctx.Export("publicIp", instance.PublicIp)
		ctx.Export("publicHostName", instance.PublicDns)

		return nil
	})
}
