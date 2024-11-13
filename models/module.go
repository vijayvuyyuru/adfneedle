package models

import (
	"context"
	"encoding/json"
	"io"
	"os"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.viam.com/rdk/components/sensor"

	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/utils/rpc"
)

var (
	Sensor          = resource.NewModel("viam-data-ml", "adfneedle", "sensor")
	errNonZeroLimit = errors.New("limit must be nonzero")
	errNoPath       = errors.New("secret path must be specified")
	errBadJSON      = errors.New("malformed json file, key `url` not found")
)

func init() {
	resource.RegisterComponent(sensor.API, Sensor,
		resource.Registration[sensor.Sensor, *Config]{
			Constructor: newAdfneedleSensor,
		},
	)
}

type Config struct {
	limit      int
	secretPath string
	// Put config attributes here

	/* if your model  does not need a config,
	   replace *Config in the init function with resource.NoNativeConfig */

	/* Uncomment this if your model does not need to be validated
	   and has no implicit dependecies. */
	resource.TriviallyValidateConfig
}

func (cfg *Config) Validate(path string) ([]string, error) {
	// Add config validation code here
	return nil, nil
}

type adfneedleSensor struct {
	name resource.Name

	logger logging.Logger
	cfg    *Config

	cancelCtx  context.Context
	cancelFunc func()

	limit      int
	secretPath string

	mongoclient *mongo.Client

	/* Uncomment this if your model does not need to reconfigure. */
	// resource.TriviallyReconfigurable

	// Uncomment this if the model does not have any goroutines that
	// need to be shut down while closing.
	resource.TriviallyCloseable
}

func newAdfneedleSensor(ctx context.Context, deps resource.Dependencies, rawConf resource.Config, logger logging.Logger) (sensor.Sensor, error) {
	conf, err := resource.NativeConfig[*Config](rawConf)
	if err != nil {
		return nil, err
	}

	url, err := validateConfigAndGetURL(conf)
	if err != nil {
		return nil, err
	}

	clientOptions := options.Client().ApplyURI(url)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		logger.Errorw("error connecting to mongo client", "error", err)
		return nil, errors.Wrap(err, "error connecting to client")
	}

	cancelCtx, cancelFunc := context.WithCancel(context.Background())

	s := &adfneedleSensor{
		name:        rawConf.ResourceName(),
		logger:      logger,
		cfg:         conf,
		cancelCtx:   cancelCtx,
		cancelFunc:  cancelFunc,
		secretPath:  conf.secretPath,
		limit:       conf.limit,
		mongoclient: client,
	}
	return s, nil
}

func validateConfigAndGetURL(conf *Config) (string, error) {
	if conf.limit == 0 {
		return "", errNonZeroLimit
	}
	if conf.secretPath == "" {
		return "", errNoPath
	}
	jsonFile, err := os.Open(conf.secretPath)
	if err != nil {
		return "", errors.Wrap(err, "failed to open json file")
	}
	defer jsonFile.Close()
	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return "", errors.Wrap(err, "failed to read from file")
	}

	var result map[string]any
	json.Unmarshal([]byte(byteValue), &result)
	url, ok := result["url"]
	if !ok {
		return "", errBadJSON
	}
	return url.(string), nil
}

func (s *adfneedleSensor) Name() resource.Name {
	return s.name
}

func (s *adfneedleSensor) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	confStruct, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		return err
	}
	if confStruct.limit == 0 {
		return errNonZeroLimit
	}
	if confStruct.secretPath == "" {
		return errNoPath
	}
	url, err := validateConfigAndGetURL(confStruct)
	if err != nil {
		return err
	}

	clientOptions := options.Client().ApplyURI(url)
	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		s.logger.Errorw("error connecting to mongo client", "error", err)
		return errors.Wrap(err, "error connecting to client")
	}
	s.limit = confStruct.limit
	s.secretPath = confStruct.secretPath
	s.mongoclient = client
	return nil
}

func (s *adfneedleSensor) NewClientFromConn(ctx context.Context, conn rpc.ClientConn, remoteName string, name resource.Name, logger logging.Logger) (sensor.Sensor, error) {
	panic("not implemented")
}

type Pipeline struct {
	Count int `bson:"count"`
}

func (s *adfneedleSensor) Readings(ctx context.Context, extra map[string]interface{}) (map[string]interface{}, error) {
	coll := s.mongoclient.Database("syncDB").Collection("data_federations")
	cursor, err := coll.Aggregate(ctx, bson.A{
		bson.D{{"$count", "count"}},
	})
	if err != nil {
		return nil, errors.Wrap(err, "error running query")
	}

	var results []Pipeline

	if err = cursor.All(context.TODO(), &results); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal cursor to struct")
	}

	return map[string]any{
		"count": results[0].Count,
		"usage": float64(results[0].Count) / float64(s.limit),
	}, nil

}

func (s *adfneedleSensor) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	panic("not implemented")
}

func (s *adfneedleSensor) Close(context.Context) error {
	if err := s.mongoclient.Disconnect(s.cancelCtx); err != nil {
		s.logger.Warnw("failed to close mongo client connection", "error", err)
	}
	s.cancelFunc()
	return nil
}
