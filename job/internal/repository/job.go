package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ziliscite/bard_narate/job/internal/domain"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type JobDTO struct {
	ID        string    `dynamodbav:"ID"`
	Status    string    `dynamodbav:"Status"`
	FileKey   string    `dynamodbav:"FileKey"`
	CreatedAt time.Time `dynamodbav:"CreatedAt"`
	UpdatedAt time.Time `dynamodbav:"UpdatedAt"`
}

func NewJobDTO(job *domain.Job) JobDTO {
	return JobDTO{
		ID:        job.ID,
		Status:    job.Status.String(),
		FileKey:   job.FileKey,
		CreatedAt: job.CreatedAt,
		UpdatedAt: job.UpdatedAt,
	}
}

func (j JobDTO) ToJob() (*domain.Job, error) {
	var status domain.JobStatus
	switch j.Status {
	case "Pending":
		status = domain.Pending
	case "Processing":
		status = domain.Processing
	case "Converting":
		status = domain.Converting
	case "Completed":
		status = domain.Completed
	case "Failed":
		status = domain.Failed
	default:
		return nil, fmt.Errorf("unknown JobStatus: %s", j.Status)
	}

	return &domain.Job{
		ID:        j.ID,
		Status:    status,
		CreatedAt: j.CreatedAt,
		UpdatedAt: j.UpdatedAt,
	}, nil
}

// JobMigrator is an interface for migrating the job table
type JobMigrator interface {
	AutoMigrate(ctx context.Context) error
	TableExists(ctx context.Context) (bool, error)
	CreateTable(ctx context.Context) error
}

type JobWriter interface {
	Save(ctx context.Context, job *domain.Job) error
	Update(ctx context.Context, job *domain.Job) error
}

type JobReader interface {
	Load(ctx context.Context, id string) (*domain.Job, error)
}

type JobDeleter interface {
	Delete(ctx context.Context, id string) error
}

type JobRepository interface {
	JobWriter
	JobReader
	JobDeleter
	JobMigrator
}

type jobRepository struct {
	t  string
	cl *dynamodb.Client
}

func NewJobRepository(dynamodbClient *dynamodb.Client, tableName string) JobRepository {
	return &jobRepository{
		cl: dynamodbClient,
		t:  tableName,
	}
}

func (j *jobRepository) AutoMigrate(ctx context.Context) error {
	exists, err := j.TableExists(ctx)
	if err != nil {
		return err
	}

	if exists {
		return nil // table already exists, no need to create it, just return nil
	}

	return j.CreateTable(ctx)
}

func (j *jobRepository) TableExists(ctx context.Context) (bool, error) {
	if _, err := j.cl.DescribeTable(
		ctx, &dynamodb.DescribeTableInput{TableName: aws.String(j.t)},
	); err != nil {
		var notFoundEx *types.ResourceNotFoundException
		switch {
		case errors.As(err, &notFoundEx):
			return false, nil
		default:
			return false, err
		}
	}

	return true, nil
}

func (j *jobRepository) CreateTable(ctx context.Context) error {
	if _, err := j.cl.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(j.t),
		AttributeDefinitions: []types.AttributeDefinition{{
			AttributeName: aws.String("id"),
			AttributeType: types.ScalarAttributeTypeS,
		}, {
			AttributeName: aws.String("status"),
			AttributeType: types.ScalarAttributeTypeS,
		}, {
			AttributeName: aws.String("file_key"),
			AttributeType: types.ScalarAttributeTypeS,
		}, {
			AttributeName: aws.String("created_at"),
			AttributeType: types.ScalarAttributeTypeS,
		}, {
			AttributeName: aws.String("updated_at"),
			AttributeType: types.ScalarAttributeTypeS,
		}},
		KeySchema: []types.KeySchemaElement{{
			AttributeName: aws.String("id"),
			KeyType:       types.KeyTypeHash,
		}},
		BillingMode: types.BillingModePayPerRequest,
	}); err != nil {
		return err
	}

	if err := dynamodb.NewTableExistsWaiter(j.cl).Wait(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(j.t),
	}, 5*time.Minute); err != nil {
		return fmt.Errorf("failed to wait for table to be created: %w", err)
	}

	return nil
}

func (j *jobRepository) Save(ctx context.Context, job *domain.Job) error {
	jobDTO := NewJobDTO(job)

	av, err := attributevalue.MarshalMap(jobDTO)
	if err != nil {
		return fmt.Errorf("failed to marshal jobDTO: %w", err)
	}

	if _, err = j.cl.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(j.t),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(ID)"),
		ReturnValues:        types.ReturnValueNone,
	}); err != nil {
		return fmt.Errorf("failed to put item: %w", err)
	}

	return nil
}

func (j *jobRepository) Load(ctx context.Context, jobID string) (*domain.Job, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(j.t),
		Key: map[string]types.AttributeValue{
			"ID": &types.AttributeValueMemberS{Value: jobID},
		},
		ConsistentRead: aws.Bool(true),
	}

	result, err := j.cl.GetItem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("job not found")
	}

	var jobDTO JobDTO
	if err = attributevalue.UnmarshalMap(result.Item, &jobDTO); err != nil {
		return nil, fmt.Errorf("failed to unmarshal jobDTO: %w", err)
	}

	return jobDTO.ToJob()
}

func (j *jobRepository) Update(ctx context.Context, job *domain.Job) error {
	jobDTO := NewJobDTO(job)

	if _, err := j.cl.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(j.t),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: jobDTO.ID},
		},
		UpdateExpression: aws.String("SET #status = :newStatus, #updatedAt = :updatedAt"),
		ExpressionAttributeNames: map[string]string{
			"#status":    "status",
			"#updatedAt": "updated_at",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":newStatus": &types.AttributeValueMemberS{Value: jobDTO.Status},
			":updatedAt": &types.AttributeValueMemberS{Value: jobDTO.UpdatedAt.Format(time.RFC3339)},
		},
		ReturnValues: types.ReturnValueUpdatedNew,
	}); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	return nil
}

func (j *jobRepository) Delete(ctx context.Context, jobID string) error {
	if _, err := j.cl.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(j.t),
		Key: map[string]types.AttributeValue{
			"ID": &types.AttributeValueMemberS{Value: jobID},
		},
		ReturnValues: types.ReturnValueNone,
	}); err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}

	return nil
}
