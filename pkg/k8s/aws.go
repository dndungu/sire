package k8s

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
)

type Input interface {
	Validate() error
}

type Request func(aws.Config) error

func (s Request) Send(config aws.Config) error {
	return s(config)
}

type awsOp struct {
	Config        aws.Config
	Input, Output interface{}
	Region        string
	Sender        Request
}

func (provider awsOp) Do() error {
	var config, err = external.LoadDefaultAWSConfig()
	config.Region = provider.Region
	if err != nil {
		return err
	}

	if err = provider.Validate(); err != nil {
		return err
	}

	return provider.Sender.Send(config)
}

func (provider awsOp) Validate() error {
	type Validator interface {
		Validate() error
	}

	if v, ok := provider.Input.(Validator); ok {
		if err := v.Validate(); err != nil {
			return err
		}
	}
	return nil
}
