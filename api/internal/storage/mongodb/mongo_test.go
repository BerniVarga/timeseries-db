package mongodb

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/stretchr/testify/assert"
	"sky/api/internal/model"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	image          = "mongo"
	version        = "5.0.3"
	port           = "27017"
	username       = "user"
	password       = "pass"
	db             = "sky"
	collectionName = "metrics"
	appName        = "api"
)

var dbURL string

func TestMain(m *testing.M) {
	ctx := context.Background()

	port, err := nat.NewPort("tcp", port)
	if err != nil {
		log.Fatalf("port creation failed %v", err)
	}

	req := testcontainers.ContainerRequest{
		Image: fmt.Sprintf("%s:%s", image, version),
		Env: map[string]string{
			"MONGO_INITDB_ROOT_USERNAME": username,
			"MONGO_INITDB_ROOT_PASSWORD": password,
		},
		ExposedPorts: []string{port.Port() + "/tcp"},
		WaitingFor:   wait.ForLog("Waiting for connections").WithStartupTimeout(time.Minute),
	}

	cntr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		log.Fatalf("test container creation failed %v", err)
	}

	ip, err := cntr.Host(ctx)
	if err != nil {
		log.Fatalf("couldn't read the ip of mongo test container %v", err)
	}

	mappedPort, err := cntr.MappedPort(ctx, port+"/tcp")
	if err != nil {
		log.Fatalf("couldn't read the mapped port of the mongo test container %v", err)
	}

	dbURL = fmt.Sprintf("mongodb://%s:%s@%s:%s/?connect=direct", username, password, ip, mappedPort.Port())

	ret := m.Run()

	if err = cntr.Terminate(ctx); err != nil {
		log.Fatalf("Failed to stop mongo container: %s", err)
	}

	os.Exit(ret)
}

func TestGetSeries(t *testing.T) {
	ctx := context.Background()

	mongoDB, err := NewMongoStorage(ctx, dbURL, appName, db, collectionName)
	assert.Nil(t, err, "error initialising test db")

	now := time.Now()
	err = insertMockData(ctx, dbURL, appName, now)
	assert.Nil(t, err, "mock data has been inserted")

	query := model.Query{
		StartAt:    now.Add(time.Duration(-6) * time.Minute),
		EndAt:      now,
		MetricType: model.MetricTypeCPULoad,
		Frequency:  model.FrequencyByMinutes,
	}

	metrics, err := mongoDB.GetSeries(ctx, query)
	assert.Nil(t, err, "unexpected error while retrieving series")
	assert.Equal(t, 5, len(metrics))
}

func insertMockData(ctx context.Context, url, appName string, date time.Time) error {
	c, err := createClient(ctx, url, appName)
	if err != nil {
		return err
	}

	var metrics []interface{}
	for i := 0; i < 5; i++ {
		metric := &model.Metric{
			Timestamp:   date.Add(-time.Duration(i) * time.Minute),
			CPULoad:     rand.Float64() * 100,
			Concurrency: rand.Int31n(500000),
		}
		metrics = append(metrics, metric)
	}

	if _, err := c.Database(db).Collection(collectionName).InsertMany(ctx, metrics); err != nil {
		return err
	}
	return nil
}

func createClient(ctx context.Context, url, appName string) (*mongo.Client, error) {
	clientOptions := options.Client().SetAppName(appName)
	clientOptions.SetAuth(options.Credential{
		AuthSource: "admin",
		Username:   username,
		Password:   password,
	})

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, clientOptions.ApplyURI(url))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongo: %w", err)
	}
	return client, nil
}
