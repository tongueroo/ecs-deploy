package client

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"

	"reflect"
)

type Client struct {
	svc          *ecs.ECS
	logger       *log.Logger
	pollInterval time.Duration
}

func New(region *string, logger *log.Logger) *Client {
	logger.Printf("client.New")
	sess := session.New(&aws.Config{Region: region})
	svc := ecs.New(sess)
	return &Client{
		svc:          svc,
		pollInterval: time.Second * 5,
		logger:       logger,
	}
}

// RegisterTaskDefinition updates the existing task definition's image.
func (c *Client) RegisterTaskDefinition(task, image, tag *string) (string, error) {
	fmt.Println("*** RegisterTaskDefinition")
	defs, err := c.GetContainerDefinitions(task)
	if err != nil {
		return "", err
	}

	fmt.Println(reflect.TypeOf(defs))
	fmt.Printf("defs %s\n", defs)
	for _, d := range defs {
		fmt.Println(reflect.TypeOf(d))
		if strings.HasPrefix(*d.Image, *image) {
			i := fmt.Sprintf("%s:%s", *image, *tag)
			fmt.Printf("new image name %s\n", i)
			d.Image = &i
		}
		memory := int64(128)
		d.Memory = &memory
	}


	fmt.Println("*** RegisterTaskDefinition 2")

	input := &ecs.RegisterTaskDefinitionInput{
		Family:               task,
		ContainerDefinitions: defs,
	}

	fmt.Println("*** RegisterTaskDefinition 3")

	resp, err := c.svc.RegisterTaskDefinition(input)
	if err != nil {
		return "", err
	}

	fmt.Println("*** RegisterTaskDefinition 4")

	return *resp.TaskDefinition.TaskDefinitionArn, nil
}

// UpdateService updates the service to use the new task definition.
func (c *Client) UpdateService(cluster, service *string, count *int64, arn *string) error {
	input := &ecs.UpdateServiceInput{
		Cluster: cluster,
		Service: service,
	}
	if *count != -1 {
		input.DesiredCount = count
	}
	if arn != nil {
		input.TaskDefinition = arn
	}
	_, err := c.svc.UpdateService(input)
	return err
}

// Wait waits for the service to finish being updated.
func (c *Client) Wait(cluster, service, arn *string) error {
	t := time.NewTicker(c.pollInterval)
	for {
		select {
		case <-t.C:
			s, err := c.GetDeployment(cluster, service, arn)
			if err != nil {
				return err
			}
			c.logger.Printf("[info] --> desired: %d, pending: %d, running: %d", *s.DesiredCount, *s.PendingCount, *s.RunningCount)
			if *s.RunningCount == *s.DesiredCount {
				return nil
			}
		}
	}
}

// GetDeployment gets the deployment for the arn.
func (c *Client) GetDeployment(cluster, service, arn *string) (*ecs.Deployment, error) {
	input := &ecs.DescribeServicesInput{
		Cluster:  cluster,
		Services: []*string{service},
	}
	output, err := c.svc.DescribeServices(input)
	if err != nil {
		return nil, err
	}
	ds := output.Services[0].Deployments
	for _, d := range ds {
		if *d.TaskDefinition == *arn {
			return d, nil
		}
	}
	return nil, nil
}

// GetContainerDefinitions get container definitions of the service.
func (c *Client) GetContainerDefinitions(task *string) ([]*ecs.ContainerDefinition, error) {
	fmt.Println("*** GetContainerDefinitions")
	output, err := c.svc.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: task,
	})
	fmt.Println("*** GetContainerDefinitions2")
	fmt.Println(task)
	// fmt.Printf("%+v\n", task)
	// fmt.Printf("%#v\n", task)
	// fmt.Printf("%T\n", task)
	fmt.Printf("%s\n", task)
	// fmt.Printf("%q\n", task)
	// fmt.Printf("%x\n", task)
	if err != nil {
		return nil, err
	}
	fmt.Println("*** GetContainerDefinitions3")
	return output.TaskDefinition.ContainerDefinitions, nil
}
