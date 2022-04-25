package mongodb

import (
	"context"
	"fmt"
	"time"

	"sky/api/internal/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

// NamespaceExistsErrCode if the collections is created, a namespace exists error code is returned
var NamespaceExistsErrCode int32 = 48

// MongoStorage is an implementation of a timeseries store in Mongo
type MongoStorage struct {
	client     *mongo.Client
	database   string
	collection string
}

// NewMongoStorage returns a mongo storage containing timeseries data
func NewMongoStorage(ctx context.Context, databaseURI, appName, databaseName, collectionName string) (*MongoStorage, error) {
	client, err := createMongoClient(ctx, databaseURI, appName)
	if err != nil {
		return nil, err
	}

	if err := initMongo(ctx, client, databaseName, collectionName); err != nil {
		return nil, err
	}

	return &MongoStorage{
		client:     client,
		database:   databaseName,
		collection: collectionName,
	}, nil
}

func initMongo(ctx context.Context, client *mongo.Client, databaseName, collectionName string) error {

	opts := options.CreateCollection().
		SetTimeSeriesOptions(options.TimeSeries().
			SetTimeField("timestamp").
			SetGranularity("minutes")).
		SetExpireAfterSeconds(315360000)

	timeoutContext, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := client.Database(databaseName).CreateCollection(timeoutContext, collectionName, opts); err != nil {
		cmdErr, ok := err.(mongo.CommandError)
		if ok && cmdErr.Code == NamespaceExistsErrCode {
			fmt.Printf("collection already exists, do nothing \n")
		} else {
			return err
		}
	}
	return nil
}

// GetSeries returns the series of events saved in the DB for the particular filter given
func (m *MongoStorage) GetSeries(ctx context.Context, config model.Query) ([]model.Metric, error) {

	if config.Frequency != model.FrequencyNone && config.Frequency != model.FrequencyByMinutes {
		return m.getSeriesByFrequency(ctx, config)
	}

	filter := bson.D{primitive.E{Key: "timestamp", Value: primitive.M{"$lte": config.EndAt, "$gte": config.StartAt}}}
	findOptions := options.Find()
	findOptions.SetSort(bson.D{primitive.E{"timestamp", 1}}) // ordering - for display it is kept from old to new

	switch config.MetricType {
	case model.MetricTypeConcurrency:
		findOptions.SetProjection(bson.D{primitive.E{"cpu_load", 0}})
	case model.MetricTypeCPULoad:
		findOptions.SetProjection(bson.D{primitive.E{"concurrency", 0}})
	}

	var results []model.Metric
	cursor, err := m.client.Database(m.database).Collection(m.collection).Find(ctx, filter, findOptions)
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, fmt.Errorf("error while retrieving data: %w", err)
	}
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	defer cursor.Close(ctx)

	if err = cursor.All(context.Background(), &results); err != nil {
		return nil, err
	}

	return results, nil
}

// getSeriesByFrequency returns the series of events saved in the DB for the particular filter given
// it considers the frequency field: in case the data range is in second, it can return averages in minutes, hours or other aggregations
func (m *MongoStorage) getSeriesByFrequency(ctx context.Context, config model.Query) ([]model.Metric, error) {
	matchStage := bson.D{
		primitive.E{"$match", bson.D{
			primitive.E{Key: "timestamp", Value: primitive.M{"$lte": config.EndAt, "$gte": config.StartAt}},
		}},
	}

	var frequency string
	switch config.Frequency {
	case model.FrequencyBySeconds, model.FrequencyByMinutes, model.FrequencyNone:
		frequency = "minute"
	case model.FrequencyByHours:
		frequency = "hour"
	case model.FrequencyByDays:
		frequency = "day"
	case model.FrequencyByMonths:
		frequency = "month"
	case model.FrequencyByYears:
		frequency = "year"
	}

	var groupStage bson.D
	switch config.MetricType {
	case model.MetricTypeConcurrency:
		groupStage = bson.D{primitive.E{"$group",
			bson.D{primitive.E{"_id", bson.D{
				primitive.E{Key: "frequency", Value: bson.D{
					primitive.E{"$dateTrunc", primitive.M{"date": "$timestamp", "unit": frequency}}}}}},
				primitive.E{"concurrency", bson.D{primitive.E{"$avg", "$concurrency"}}}}}}
	case model.MetricTypeCPULoad:
		groupStage = bson.D{primitive.E{"$group",
			bson.D{primitive.E{"_id", bson.D{
				primitive.E{Key: "frequency", Value: bson.D{
					primitive.E{"$dateTrunc", primitive.M{"date": "$timestamp", "unit": frequency}}}}}},
				primitive.E{"cpu_load", bson.D{primitive.E{"$avg", "$cpu_load"}}}}}}
	default:
		groupStage = bson.D{primitive.E{"$group",
			bson.D{primitive.E{"_id", bson.D{
				primitive.E{Key: "frequency", Value: bson.D{
					primitive.E{"$dateTrunc", primitive.M{"date": "$timestamp", "unit": frequency}}}}}},
				primitive.E{"cpu_load", bson.D{primitive.E{"$avg", "$cpu_load"}}},
				primitive.E{"concurrency", bson.D{primitive.E{"$avg", "$concurrency"}}}}}}
	}
	var projectionStage bson.D = bson.D{
		primitive.E{"$project", bson.D{
			primitive.E{Key: "timestamp", Value: "$_id.frequency"},
			primitive.E{Key: "cpu_load", Value: "$cpu_load"},
			primitive.E{Key: "concurrency", Value: "$concurrency"},
		}},
	}

	opts := options.Aggregate().SetMaxTime(2 * time.Second)
	pipeline := mongo.Pipeline{matchStage, groupStage, projectionStage}

	var results []model.Metric
	cursor, err := m.client.Database(m.database).Collection(m.collection).Aggregate(ctx, pipeline, opts)
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, fmt.Errorf("error while retrieving data: %w", err)
	}
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	defer cursor.Close(ctx)

	if err = cursor.All(context.Background(), &results); err != nil {
		return nil, err
	}

	return results, nil
}

// GetAverage - returns the average value of a metrics for a certain time range
func (m *MongoStorage) GetAverage(ctx context.Context, config model.Query) (*model.MetricAverage, error) {
	matchStage := bson.D{
		primitive.E{"$match", bson.D{
			primitive.E{Key: "timestamp", Value: primitive.M{"$lte": config.EndAt, "$gte": config.StartAt}},
		}},
	}

	var groupStage bson.D
	switch config.MetricType {
	case model.MetricTypeConcurrency:
		groupStage = bson.D{primitive.E{"$group", bson.D{primitive.E{"_id", ""},
			primitive.E{"concurrency", bson.D{primitive.E{"$avg", "$concurrency"}}}}}}
	case model.MetricTypeCPULoad:
		groupStage = bson.D{primitive.E{"$group", bson.D{primitive.E{"_id", ""},
			primitive.E{"cpu_load", bson.D{primitive.E{"$avg", "$cpu_load"}}}}}}
	default:
		groupStage = bson.D{primitive.E{"$group", bson.D{primitive.E{"_id", ""},
			primitive.E{"cpu_load", bson.D{primitive.E{"$avg", "$cpu_load"}}},
			primitive.E{"concurrency", bson.D{primitive.E{"$avg", "$concurrency"}}}}}}
	}

	opts := options.Aggregate().SetMaxTime(2 * time.Second)
	pipeline := mongo.Pipeline{matchStage, groupStage}

	var results []model.MetricAverage
	cursor, err := m.client.Database(m.database).Collection(m.collection).Aggregate(ctx, pipeline, opts)
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, fmt.Errorf("error while retrieving data: %w", err)
	}
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	defer cursor.Close(ctx)

	if err = cursor.All(context.Background(), &results); err != nil {
		return nil, err
	}

	if len(results) > 1 {
		return nil, fmt.Errorf("only one aggregation is expected")
	}

	res := results[0]
	res.StartTime = config.StartAt
	res.EndTime = config.EndAt

	return &res, nil
}

func createMongoClient(ctx context.Context, dbURI, appName string) (*mongo.Client, error) {
	clientOptions := options.Client().
		ApplyURI(dbURI).
		SetAppName(appName).
		SetReadPreference(readpref.SecondaryPreferred()).
		SetWriteConcern(writeconcern.New(writeconcern.WMajority()))

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection to db: %w", err)
	}
	return client, nil
}
