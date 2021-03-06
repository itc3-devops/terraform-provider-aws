package aws

import (
	"fmt"
	"log"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsVpnGateway() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsVpnGatewayRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"state": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"attached_vpc_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"availability_zone": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"amazon_side_asn": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"filter": ec2CustomFiltersSchema(),
			"tags":   TagsSchemaComputed(),
		},
	}
}

func dataSourceAwsVpnGatewayRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	req := &ec2.DescribeVpnGatewaysInput{}

	if id, ok := d.GetOk("id"); ok {
		req.VpnGatewayIds = aws.StringSlice([]string{id.(string)})
	}

	req.Filters = buildEC2AttributeFilterList(
		map[string]string{
			"state":             d.Get("state").(string),
			"availability-zone": d.Get("availability_zone").(string),
		},
	)
	if asn, ok := d.GetOk("amazon_side_asn"); ok {
		req.Filters = append(req.Filters, buildEC2AttributeFilterList(
			map[string]string{
				"amazon-side-asn": asn.(string),
			},
		)...)
	}
	if id, ok := d.GetOk("attached_vpc_id"); ok {
		req.Filters = append(req.Filters, buildEC2AttributeFilterList(
			map[string]string{
				"attachment.state":  "attached",
				"attachment.vpc-id": id.(string),
			},
		)...)
	}
	req.Filters = append(req.Filters, buildEC2TagFilterList(
		TagsFromMap(d.Get("tags").(map[string]interface{})),
	)...)
	req.Filters = append(req.Filters, buildEC2CustomFilterList(
		d.Get("filter").(*schema.Set),
	)...)
	if len(req.Filters) == 0 {
		// Don't send an empty filters list; the EC2 API won't accept it.
		req.Filters = nil
	}

	log.Printf("[DEBUG] Reading VPN Gateway: %s", req)
	resp, err := conn.DescribeVpnGateways(req)
	if err != nil {
		return err
	}
	if resp == nil || len(resp.VpnGateways) == 0 {
		return fmt.Errorf("no matching VPN gateway found: %#v", req)
	}
	if len(resp.VpnGateways) > 1 {
		return fmt.Errorf("multiple VPN gateways matched; use additional constraints to reduce matches to a single VPN gateway")
	}

	vgw := resp.VpnGateways[0]

	d.SetId(aws.StringValue(vgw.VpnGatewayId))
	d.Set("state", vgw.State)
	d.Set("availability_zone", vgw.AvailabilityZone)
	d.Set("amazon_side_asn", strconv.FormatInt(aws.Int64Value(vgw.AmazonSideAsn), 10))
	d.Set("tags", TagsToMap(vgw.Tags))

	for _, attachment := range vgw.VpcAttachments {
		if *attachment.State == "attached" {
			d.Set("attached_vpc_id", attachment.VpcId)
			break
		}
	}

	return nil
}
