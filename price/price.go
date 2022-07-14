package price

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/pricing"
	"github.com/aws/aws-sdk-go-v2/service/pricing/types"
)

var ec2 = "AmazonEC2"

const (
	sc = "ServiceCode"
	st = "instanceType"
	rg = "regionCode"
	op = "operatingSystem"
	va = "volumeApiName"
	vt = "volumeType"
)

func GetPrice(key, secret, region, instanceType, volumeType, volumeApiName string) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	creds := credentials.NewStaticCredentialsProvider(key, secret, "")
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithCredentialsProvider(creds), config.WithRegion(region))
	if err != nil {
		return "", err
	}
	client := pricing.NewFromConfig(cfg)

	filterSC := getFilter(sc, ec2)
	filterST := getFilter(st, instanceType)
	filterRG := getFilter(rg, region)
	filterOP := getFilter(op, "linux")

	// filter rule: https://docs.aws.amazon.com/aws-cost-management/latest/APIReference/API_pricing_GetProducts.html
	ec2pd, err := client.GetProducts(ctx, &pricing.GetProductsInput{
		ServiceCode: &ec2,
		Filters: []types.Filter{
			filterSC,
			filterST,
			filterRG,
			filterOP,
		},
		MaxResults: int32(1),
	}, nil)
	if err != nil {
		return "", err
	}

	ebspd, err := getEBSPrice(ctx, client, region, volumeType, volumeApiName)
	if err != nil {
		return "", err
	}

	ec2Price := ec2pd.PriceList[1]
	ebsPrice := ebspd.PriceList[1]

	res := fmt.Sprintf("ec2 price: %s\n ebs price: %s\n", ec2Price, ebsPrice)

	return res, nil
}

func getEBSPrice(ctx context.Context, client *pricing.Client, region, volumeType, volumeApiName string) (*pricing.GetProductsOutput, error) {
	filterSC := getFilter(sc, ec2)
	filterRG := getFilter(rg, region)
	filterVT := getFilter(vt, volumeType)
	filterVA := getFilter(va, volumeApiName)

	res, err := client.GetProducts(ctx, &pricing.GetProductsInput{
		ServiceCode: &ec2,
		Filters: []types.Filter{
			filterSC,
			filterRG,
			filterVT,
			filterVA,
		},
		MaxResults: int32(1),
	}, nil)
	if err != nil {
		return nil, err
	}
	return res, nil

}

func getFilter(field, value string) types.Filter {
	filter := types.Filter{
		Type:  types.FilterTypeTermMatch,
		Field: &field,
		Value: &value,
	}

	return filter
}
