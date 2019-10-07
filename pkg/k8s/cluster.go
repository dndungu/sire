package k8s

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

type Subnet struct {
	CidrBlock, Zone string
}

type Tag struct {
	Key, Value string
}

type Policy struct {
	Description string
	Name        string
	Document    string
}

type Cluster struct {
	Name        string
	Description string
	CidrBlock   string
	DryRun      bool
	Policies    []Policy
	Subnets     []Subnet
	Tags        []Tag
	Region      string

	InternetGatewayId string
	RouteTableId      string
	PolicyIds         []string
	SecurityGroupId   string
	VpcId             string
}

type retryError struct {
	d time.Duration
	s string
}

func (err retryError) Error() string {
	return err.s
}

func (err retryError) Duration() time.Duration {
	return err.d
}

type Step interface {
	Do() error
}

func (c Cluster) Init() error {
	var steps = []Step{
		c.createVpc(),
	}

	for i := range c.Subnets {
		steps = append(steps, c.createSubnet(i))
	}

	steps = append(steps, c.createInternetGateway())
	steps = append(steps, c.attachInternetGateway())
	steps = append(steps, c.describeRouteTables())
	steps = append(steps, c.createRouteToInternet())
	steps = append(steps, c.createSecurityGroup())

	//for i := range c.Policies {
	//	steps = append(steps, c.createPolicies(i).Do)
	//}

	return do(steps...)
}

func do(steps ...Step) error {
	for i, fn := range steps {
		if err := fn.Do(); err != nil {
			if v, ok := err.(retryError); ok {
				time.Sleep(v.Duration())
				return do(steps[i:]...)
			}
			return err
		}
	}
	return nil
}

func (c *Cluster) createVpc() awsOp {
	var op awsOp
	op.Region = c.Region
	var input = ec2.CreateVpcInput{
		DryRun:    &c.DryRun,
		CidrBlock: &c.CidrBlock,
	}
	op.Input = &input
	var sender Request = func(config aws.Config) error {
		var response, err = ec2.New(config).CreateVpcRequest(&input).Send(context.Background())
		if err == nil {
			c.VpcId = *response.Vpc.VpcId
			err = c.tagResource(c.VpcId).Do()
		}
		return err
	}
	op.Sender = sender
	return op
}

func (c *Cluster) createSubnet(index int) awsOp {
	var op awsOp
	op.Region = c.Region
	var subnet = c.Subnets[index]
	var input = ec2.CreateSubnetInput{
		DryRun:           &c.DryRun,
		AvailabilityZone: &subnet.Zone,
		CidrBlock:        &subnet.CidrBlock,
		VpcId:            &c.VpcId,
	}
	op.Input = &input
	var sender Request = func(config aws.Config) error {
		var response, err = ec2.New(config).CreateSubnetRequest(&input).Send(context.Background())
		if err == nil {
			err = c.tagResource(*response.Subnet.SubnetId).Do()
		}
		return err
	}
	op.Sender = sender
	return op
}

func (c *Cluster) createInternetGateway() awsOp {
	var op awsOp
	op.Region = c.Region
	var input = ec2.CreateInternetGatewayInput{
		DryRun: &c.DryRun,
	}
	op.Input = &input
	var sender Request = func(config aws.Config) error {
		var response, err = ec2.New(config).CreateInternetGatewayRequest(&input).Send(context.Background())
		if err == nil {
			c.InternetGatewayId = *response.InternetGateway.InternetGatewayId
			err = c.tagResource(c.InternetGatewayId).Do()
		}
		return err
	}
	op.Sender = sender
	return op
}

func (c *Cluster) attachInternetGateway() awsOp {
	var op awsOp
	op.Region = c.Region
	var input = ec2.AttachInternetGatewayInput{
		DryRun:            &c.DryRun,
		InternetGatewayId: &c.InternetGatewayId,
		VpcId:             &c.VpcId,
	}
	op.Input = &input
	var sender Request = func(config aws.Config) error {
		var _, err = ec2.New(config).AttachInternetGatewayRequest(&input).Send(context.Background())
		return err
	}
	op.Sender = sender
	return op
}

func (c *Cluster) describeRouteTables() awsOp {
	var op awsOp
	op.Region = c.Region
	var sender Request = func(config aws.Config) error {
		var input = ec2.DescribeRouteTablesInput{
			DryRun: &c.DryRun,
			Filters: []ec2.Filter{
				ec2.Filter{
					Name:   aws.String("vpc-id"),
					Values: []string{c.VpcId},
				},
			},
		}
		op.Input = &input
		var response, err = ec2.New(config).DescribeRouteTablesRequest(&input).Send(context.Background())
		if err == nil {
			if len(response.RouteTables) == 0 {
				var d = 5 * time.Second
				return retryError{s: fmt.Sprintf("no route tables found, retrying in %v", d), d: d}
			}
			c.RouteTableId = *response.RouteTables[0].RouteTableId
			err = c.tagResource(c.RouteTableId).Do()
		}
		return err
	}
	op.Sender = sender
	return op
}

func (c *Cluster) createRouteToInternet() awsOp {
	var op awsOp
	op.Region = c.Region
	var sender Request = func(config aws.Config) error {
		var input = ec2.CreateRouteInput{
			DryRun:               &c.DryRun,
			DestinationCidrBlock: aws.String("0.0.0.0/0"),
			GatewayId:            &c.InternetGatewayId,
			RouteTableId:         &c.RouteTableId,
		}
		op.Input = &input
		var _, err = ec2.New(config).CreateRouteRequest(&input).Send(context.Background())
		return err
	}
	op.Sender = sender
	return op
}

func (c *Cluster) createSecurityGroup() awsOp {
	var op awsOp
	op.Region = c.Region
	var sender Request = func(config aws.Config) error {
		var input = ec2.CreateSecurityGroupInput{
			DryRun:      &c.DryRun,
			Description: &c.Description,
			GroupName:   &c.Name,
			VpcId:       &c.VpcId,
		}
		op.Input = &input
		var response, err = ec2.New(config).CreateSecurityGroupRequest(&input).Send(context.Background())
		if err == nil {
			c.SecurityGroupId = *response.GroupId
			err = c.tagResource(c.SecurityGroupId).Do()
		}
		return err
	}
	op.Sender = sender
	return op
}

func (c *Cluster) createPolicies(index int) awsOp {
	var policy = c.Policies[index]
	var op awsOp
	op.Region = c.Region
	var input = iam.CreatePolicyInput{
		Description:    &policy.Description,
		PolicyName:     &policy.Name,
		PolicyDocument: &policy.Document,
	}
	op.Input = &input
	var sender Request = func(config aws.Config) error {
		var response, err = iam.New(config).CreatePolicyRequest(&input).Send(context.Background())
		if err == nil {
			var id = *response.Policy.PolicyId
			c.PolicyIds = append(c.PolicyIds, id)
			err = c.tagResource(id).Do()
		}
		return err
	}
	op.Sender = sender
	return op
}

func (c *Cluster) createInstanceProfile(name string) func() awsOp {
	return func() awsOp {
		var op awsOp
		op.Region = c.Region
		var input = iam.CreateInstanceProfileInput{
			InstanceProfileName: &name,
		}
		op.Input = &input
		var sender Request = func(config aws.Config) error {
			var response, err = iam.New(config).CreateInstanceProfileRequest(&input).Send(context.Background())
			if err != nil {
				return err
			}
			var id = *response.InstanceProfile.InstanceProfileId
			if err = c.tagResource(id).Do(); err != nil {
				return err
			}

			return err
		}
		op.Sender = sender
		return op
	}
}

func (c *Cluster) createRoleRequest(index int) func() awsOp {
	return func() awsOp {
		var op = awsOp{Region: c.Region}
		var input = iam.CreateRoleInput{
			AssumeRolePolicyDocument: aws.String(""),
		}
		op.Input = &input
		return op
	}
}

func (c *Cluster) tagResource(id string) awsOp {
	var op awsOp
	op.Region = c.Region
	var input = ec2.CreateTagsInput{
		Resources: []string{id},
	}
	for i := range c.Tags {
		var tag = c.Tags[i]
		input.Tags = append(input.Tags, ec2.Tag{Key: &tag.Key, Value: &tag.Value})
	}
	op.Input = &input
	var sender Request = func(config aws.Config) error {
		config.Region = c.Region
		var _, err = ec2.New(config).CreateTagsRequest(&input).Send(context.Background())
		return err
	}
	op.Sender = sender

	return op
}
