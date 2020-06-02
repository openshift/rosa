package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/servicequotas"
)

type quota struct {
	ServiceCode  string
	QuotaName    string
	QuotaCode    string
	DesiredValue *float64
}

// List of service quotas we verify for cluster installs
// to support 5 x multi zone clusters
var serviceQuotaServices = []quota{
	{
		ServiceCode:  "ec2",
		QuotaCode:    "L-0263D0A3",
		QuotaName:    "Number of EIPs - VPC EIPs",
		DesiredValue: Float64(5.0),
	},
	{
		ServiceCode:  "ec2",
		QuotaCode:    "L-1216C47A",
		QuotaName:    "Running On-Demand Standard (A, C, D, H, I, M, R, T, Z) instances",
		DesiredValue: Float64(200.0),
	},
	{
		ServiceCode:  "vpc",
		QuotaCode:    "L-F678F1CE",
		QuotaName:    "VPCs per Region",
		DesiredValue: Float64(5.0),
	},
	{
		ServiceCode:  "vpc",
		QuotaCode:    "L-A4707A72",
		QuotaName:    "Internet gateways per Region",
		DesiredValue: Float64(5.0),
	},
	{
		ServiceCode:  "vpc",
		QuotaCode:    "L-DF5E4CA3",
		QuotaName:    "Network interfaces per Region",
		DesiredValue: Float64(5000.0),
	},
	{
		ServiceCode:  "ebs",
		QuotaCode:    "L-D18FCD1D",
		QuotaName:    "General Purpose SSD (gp2) volume storage",
		DesiredValue: Float64(300.0),
	},
	{
		ServiceCode:  "ebs",
		QuotaCode:    "L-309BACF6",
		QuotaName:    "Number of EBS snapshots",
		DesiredValue: Float64(300.0),
	},
	{
		ServiceCode:  "ebs",
		QuotaCode:    "L-B3A130E6",
		QuotaName:    "Provisioned IOPS",
		DesiredValue: Float64(300000.0),
	},
	{
		ServiceCode:  "ebs",
		QuotaCode:    "L-FD252861",
		QuotaName:    "Provisioned IOPS SSD (io1) volume storage",
		DesiredValue: Float64(300.0),
	},
	{
		ServiceCode:  "elasticloadbalancing",
		QuotaCode:    "L-53DA6B97",
		QuotaName:    "Application Load Balancers per Region",
		DesiredValue: Float64(50.0),
	},
	{
		ServiceCode:  "elasticloadbalancing",
		QuotaCode:    "L-E9E9831D",
		QuotaName:    "Classic Load Balancers per Region",
		DesiredValue: Float64(20.0),
	},
}

// Float64 returns a pointer to the float64 value passed in
func Float64(v float64) *float64 {
	return &v
}

// ListServiceQuotas list available quotas for service
func ListServiceQuotas(client *awsClient, serviceCode string) ([]*servicequotas.ServiceQuota, error) {
	var serviceQuotas []*servicequotas.ServiceQuota

	// Paginate through quota results
	listServiceQuotasInput := &servicequotas.ListServiceQuotasInput{ServiceCode: &serviceCode}
	err := client.servicequotasClient.ListServiceQuotasPages(listServiceQuotasInput,
		func(page *servicequotas.ListServiceQuotasOutput, lastPage bool) bool {
			serviceQuotas = append(serviceQuotas, page.Quotas...)
			return page.NextToken != nil
		})
	if err != nil {
		return nil, err
	}

	return serviceQuotas, err
}

// GetServiceQuota extract service quota for the list of service quotas
func GetServiceQuota(serviceQuotas []*servicequotas.ServiceQuota,
	quotaCode string) (*servicequotas.ServiceQuota, error) {
	for _, serviceQuota := range serviceQuotas {
		if *serviceQuota.QuotaCode == quotaCode {
			return serviceQuota, nil
		}
	}
	return nil, fmt.Errorf("Unable to find quota with service code: %s", quotaCode)
}

// CheckQuota return quota value for quota code
func CheckQuota(client *awsClient, quota quota) (bool, error) {
	serviceQuotas, err := ListServiceQuotas(client, quota.ServiceCode)
	if err != nil {
		return false, err
	}

	serviceQuota, err := GetServiceQuota(serviceQuotas, quota.QuotaCode)
	if err != nil {
		return false, err
	}

	return HasQuota(serviceQuota, quota), nil
}

// HasQuota return a true if quota is equal or greater than our required value
func HasQuota(serviceQuota *servicequotas.ServiceQuota, quota quota) bool {
	return *serviceQuota.Value >= *quota.DesiredValue
}
